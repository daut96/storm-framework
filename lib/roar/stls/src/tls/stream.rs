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
            
            // 2. Set SNI (Server Name Indication)
            let host_c = std::ffi::CString::new(hostname).unwrap();
            bssl::SSL_set_tlsext_host_name(ssl, host_c.as_ptr());

            // 3. Ekstrak File Descriptor dari Tokio dan berikan ke BoringSSL
            let fd = tcp.as_raw_fd();
            bssl::SSL_set_fd(ssl, fd);

            // 4. Lakukan Handshake (Karena ini Async, aslinya Anda butuh loop WANT_READ/WANT_WRITE di sini)
            // Untuk penyederhanaan kerangka awal:
            let ret = bssl::SSL_connect(ssl);
            if ret != 1 {
                return Err(io::Error::new(io::ErrorKind::Other, "Handshake TLS Gagal"));
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
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        // 1. Pastikan socket TCP siap dibaca
        match self.tcp.poll_read_ready(cx) {
            Poll::Ready(Ok(())) => {}
            Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
            Poll::Pending => return Poll::Pending,
        }

        unsafe {
            let slice = buf.unfilled_mut();
            // Panggil native Google BoringSSL read
            let read_bytes = bssl::SSL_read(self.ssl, slice.as_mut_ptr() as *mut _, slice.len() as i32);

            if read_bytes > 0 {
                buf.advance(read_bytes as usize);
                Poll::Ready(Ok(()))
            } else {
                let err_code = bssl::SSL_get_error(self.ssl, read_bytes);
                if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                    // BoringSSL bilang "Saya butuh data TCP lagi". Kembalikan Pending ke Tokio.
                    cx.waker().wake_by_ref();
                    Poll::Pending
                } else {
                    Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionAborted, "SSL_read error")))
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
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        // Logikanya sama dengan poll_read, tapi memanggil bssl::SSL_write
        // dan menangani SSL_ERROR_WANT_WRITE
        
        match self.tcp.poll_write_ready(cx) {
            Poll::Ready(Ok(())) => {}
            Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
            Poll::Pending => return Poll::Pending,
        }

        unsafe {
            let written = bssl::SSL_write(self.ssl, buf.as_ptr() as *const _, buf.len() as i32);
            if written > 0 {
                Poll::Ready(Ok(written as usize))
            } else {
                let err_code = bssl::SSL_get_error(self.ssl, written);
                if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                    cx.waker().wake_by_ref();
                    Poll::Pending
                } else {
                    Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionAborted, "SSL_write error")))
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

