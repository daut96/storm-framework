// src/tls/stream.rs
use crate::bssl;
use crate::tls::extensions;

use std::io;
use std::os::fd::AsRawFd;
use std::pin::Pin;
use std::task::{Context, Poll};
use tokio::io::{AsyncRead, AsyncWrite, ReadBuf};
use tokio::net::TcpStream;

// 1. Cabut impl Sync! Objek SSL tidak boleh diakses concurrent secara mentah.
// HTTP/2 h2 crate harus menggunakan internal Mutex jika ingin split stream, 
// atau kita serahkan sinkronisasi pada &mut mutabilitas Rust.
pub struct SslPtr(pub *mut bssl::SSL);

unsafe impl Send for SslPtr {}
// Dihapus: unsafe impl Sync for SslPtr {}

pub struct StormTlsStream {
    pub tcp: TcpStream,
    pub ssl: SslPtr,
}

// 2. Proteksi Drop tingkat tinggi (RAII)
impl Drop for StormTlsStream {
    fn drop(&mut self) {
        unsafe {
            if !self.ssl.0.is_null() {
                // BoringSSL aman menerima SSL_free walau state belum selesai
                bssl::SSL_free(self.ssl.0);
            }
        }
    }
}

impl StormTlsStream {
    pub async fn connect(ctx: *mut bssl::SSL_CTX, tcp: TcpStream, hostname: &str) -> io::Result<Self> {
        let ssl_ptr = unsafe { bssl::SSL_new(ctx) };
        if ssl_ptr.is_null() {
            return Err(io::Error::new(io::ErrorKind::Other, "FATAL: Out of memory for SSL object"));
        }

        // Langsung bungkus ke dalam struct. Jika terjadi error (?), Drop akan otomatis memanggil SSL_free!
        // Mencegah memory leak & MTE UAF.
        let stream = Self { tcp, ssl: SslPtr(ssl_ptr) };

        unsafe {
            if let Err(e) = extensions::apply_alps_extension(stream.ssl.0) {
                return Err(io::Error::new(io::ErrorKind::Other, e));
            }
            
            let host_c = std::ffi::CString::new(hostname).unwrap();
            bssl::SSL_set_tlsext_host_name(stream.ssl.0, host_c.as_ptr());
            bssl::SSL_set_fd(stream.ssl.0, stream.tcp.as_raw_fd());

            // THE ASYNC HANDSHAKE LOOP
            loop {
                let ret = bssl::SSL_do_handshake(stream.ssl.0);
                if ret == 1 {
                    break; // Handshake Sukses
                }

                let err_code = bssl::SSL_get_error(stream.ssl.0, ret);
                if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                    stream.tcp.readable().await?;
                } else if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                    stream.tcp.writable().await?;
                } else {
                    return Err(io::Error::new(
                        io::ErrorKind::ConnectionAborted,
                        format!("TLS Handshake Failed! Code: {}", err_code)
                    ));
                }
            }
        }

        Ok(stream)
    }
}

// ----------------------------------------------------------------------
// IMPLEMENTASI ASYNC READ (Mencegah Tokio Deadlock & MTE)
// ----------------------------------------------------------------------
impl AsyncRead for StormTlsStream {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        let this = self.get_mut();

        loop {
            // PERBAIKAN FATAL: Selalu panggil BoringSSL DULU sebelum cek socket Tokio!
            // Ini untuk menguras data yang tertinggal di internal buffer BoringSSL.
            let read_bytes = unsafe { 
                let slice = buf.unfilled_mut();
                bssl::SSL_read(this.ssl.0, slice.as_mut_ptr() as *mut _, slice.len() as i32) 
            };

            if read_bytes > 0 {
                // Kita beri tahu Tokio bahwa memori sebesar read_bytes sudah 
                // diinisialisasi (ditulis) dengan aman oleh BoringSSL
                unsafe { buf.assume_init(read_bytes as usize); }
                buf.advance(read_bytes as usize);
                return Poll::Ready(Ok(()));
            } 

            let err_code = unsafe { bssl::SSL_get_error(this.ssl.0, read_bytes) };
            
            if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                // Buffer internal C kosong, baru kita tunggu TCP Socket dari OS
                match this.tcp.poll_read_ready(cx) {
                    Poll::Ready(Ok(())) => continue, // Socket ada data, ulangi SSL_read
                    Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                    Poll::Pending => return Poll::Pending, // OS nyuruh nunggu
                }
            } else if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                // Renegotiation (jarang di HTTP/2 tapi harus dihandle)
                match this.tcp.poll_write_ready(cx) {
                    Poll::Ready(Ok(())) => continue,
                    Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                    Poll::Pending => return Poll::Pending,
                }
            } else if err_code == bssl::SSL_ERROR_ZERO_RETURN as i32 {
                return Poll::Ready(Ok(())); // Clean EOF
            } else {
                return Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionReset, "BSSL_read error")));
            }
        }
    }
}

// ----------------------------------------------------------------------
// IMPLEMENTASI ASYNC WRITE
// ----------------------------------------------------------------------
impl AsyncWrite for StormTlsStream {
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<io::Result<usize>> {
        let this = self.get_mut();
        
        loop {
            // Sama dengan read, prioritas eksekusi ke internal state C dulu
            let written = unsafe { 
                bssl::SSL_write(this.ssl.0, buf.as_ptr() as *const _, buf.len() as i32) 
            };
            
            if written > 0 {
                return Poll::Ready(Ok(written as usize));
            } 
            
            let err_code = unsafe { bssl::SSL_get_error(this.ssl.0, written) };
            
            if err_code == bssl::SSL_ERROR_WANT_WRITE as i32 {
                match this.tcp.poll_write_ready(cx) {
                    Poll::Ready(Ok(())) => continue,
                    Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                    Poll::Pending => return Poll::Pending,
                }
            } else if err_code == bssl::SSL_ERROR_WANT_READ as i32 {
                match this.tcp.poll_read_ready(cx) {
                    Poll::Ready(Ok(())) => continue,
                    Poll::Ready(Err(e)) => return Poll::Ready(Err(e)),
                    Poll::Pending => return Poll::Pending,
                }
            } else {
                return Poll::Ready(Err(io::Error::new(io::ErrorKind::ConnectionAborted, "BSSL_write error")));
            }
        }
    }

    fn poll_flush(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        Poll::Ready(Ok(()))
    }

    fn poll_shutdown(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<io::Result<()>> {
        let this = self.get_mut();
        unsafe { bssl::SSL_shutdown(this.ssl.0); }
        Poll::Ready(Ok(()))
    }
}
                  
