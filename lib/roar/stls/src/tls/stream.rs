// src/tls/stream.rs
use crate::bssl;
use std::io;
use std::os::fd::AsRawFd;
use std::pin::Pin;
use std::task::{Context, Poll};
use tokio::io::{AsyncRead, AsyncWrite, ReadBuf};
use tokio::net::TcpStream;

pub struct StormTlsStream {
    tcp: TcpStream,
    ssl: *mut bssl::SSL,
}

impl StormTlsStream {
    pub async fn connect(ctx: *mut bssl::SSL_CTX, tcp: TcpStream, hostname: &str) -> io::Result<Self> {
        unsafe {
            // 1. Buat object SSL untuk koneksi ini
            let ssl = bssl::SSL_new(ctx);
            if ssl.is_null() {
                return Err(io::Error::new(io::ErrorKind::Other, "FATAL: Gagal membuat objek SSL"));
            }
            
            // 2. Set SNI (Server Name Indication)
            let host_c = std::ffi::CString::new(hostname).unwrap();
            bssl::SSL_set_tlsext_host_name(ssl, host_c.as_ptr());

            // 3. Ekstrak File Descriptor dari Tokio dan berikan ke BoringSSL
            let fd = tcp.as_raw_fd();
            bssl::SSL_set_fd(ssl, fd);

            // ========================================================
            // THE ASYNC HANDSHAKE LOOP (Solusi Blocking C-Pointer)
            // ========================================================
            loop {
                let ret = bssl::SSL_do_handshake(ssl);
                
                if ret == 1 {
                    // Handshake sukses (ServerHello, Key Exchange, Finished selesai)
                    break; 
                }

                let err_code = bssl::SSL_get_error(ssl, ret);
                
                if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                    // BoringSSL menunggu paket dari server. Suruh Tokio menunggu.
                    tcp.readable().await?;
                } else if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                    // Buffer OS penuh, tunggu sampai bisa menulis lagi.
                    tcp.writable().await?;
                } else {
                    // WAF mendeteksi anomali atau sertifikat gagal!
                    bssl::SSL_free(ssl); // Cleanup sebelum return error
                    return Err(io::Error::new(
                        io::ErrorKind::ConnectionAborted,
                        format!("TLS Handshake Gagal! SSL_error_code: {}", err_code)
                    ));
                }
            }

            Ok(Self { tcp, ssl })
        }
    }
}

// ----------------------------------------------------------------------
// IMPLEMENTASI ASYNC READ (Menerima respons HTTP/2 dari server)
// ----------------------------------------------------------------------
impl AsyncRead for StormTlsStream {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        loop {
            // 1. Pastikan socket TCP siap dibaca oleh Epoll
            match self.tcp.poll_read_ready(cx) {
                Poll::Ready(Ok(())) => {}
                Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                Poll::Pending => return Poll::Pending,
            }

            unsafe {
                let slice = buf.unfilled_mut();
                // 2. Panggil native Google BoringSSL read
                let read_bytes = bssl::SSL_read(self.ssl, slice.as_mut_ptr() as *mut _, slice.len() as i32);

                if read_bytes > 0 {
                    buf.advance(read_bytes as usize);
                    return Poll::Ready(Ok(()));
                } 

                let err_code = bssl::SSL_get_error(self.ssl, read_bytes);
                
                if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                    // PERBAIKAN FATAL: Bersihkan status "Ready" dari Tokio agar tidak Spin-Loop 100% CPU
                    let _ = self.tcp.poll_read_ready(cx);
                    continue; // Ulangi loop dan kembali ke status Pending di atas
                } else if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                    let _ = self.tcp.poll_write_ready(cx);
                    return Poll::Pending;
                } else if err_code == bssl::SSL_ERROR_ZERO_RETURN as i32 {
                    // Koneksi ditutup dengan bersih (Clean Shutdown)
                    return Poll::Ready(Ok(()));
                } else {
                    return Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionReset, "BoringSSL_read error")));
                }
            }
        }
    }
}

// ----------------------------------------------------------------------
// IMPLEMENTASI ASYNC WRITE (Mengirim payload HTTP/2 ke server)
// ----------------------------------------------------------------------
impl AsyncWrite for StormTlsStream {
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        loop {
            match self.tcp.poll_write_ready(cx) {
                Poll::Ready(Ok(())) => {}
                Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                Poll::Pending => return Poll::Pending,
            }

            unsafe {
                let written = bssl::SSL_write(self.ssl, buf.as_ptr() as *const _, buf.len() as i32);
                
                if written > 0 {
                    return Poll::Ready(Ok(written as usize));
                } 
                
                let err_code = bssl::SSL_get_error(self.ssl, written);
                
                if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                    // PERBAIKAN FATAL: Mencegah Spin-Loop saat buffer kirim penuh
                    let _ = self.tcp.poll_write_ready(cx);
                    continue;
                } else if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                    let _ = self.tcp.poll_read_ready(cx);
                    return Poll::Pending;
                } else {
                    return Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionAborted, "BoringSSL_write error")));
                }
            }
        }
    }

    fn poll_flush(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        Poll::Ready(Ok(()))
    }

    fn poll_shutdown(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        unsafe { bssl::SSL_shutdown(self.ssl); }
        Poll::Ready(Ok(()))
    }
}

impl Drop for StormTlsStream {
    fn drop(&mut self) {
        unsafe {
            if !self.ssl.is_null() {
                bssl::SSL_free(self.ssl);
            }
        }
    }
}
