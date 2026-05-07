// src/tls/builder.rs
use crate::bssl; 
use super::extensions; 

pub struct StormSslContext {
    pub inner: *mut bssl::SSL_CTX,
}

unsafe impl Send for StormSslContext {}
unsafe impl Sync for StormSslContext {}

impl Drop for StormSslContext {
    fn drop(&mut self) {
        unsafe { if !self.inner.is_null() { bssl::SSL_CTX_free(self.inner); } }
    }
}

pub fn build_chrome_ssl_context() -> Result<StormSslContext, String> {
    let ctx = unsafe { bssl::SSL_CTX_new(bssl::TLS_method()) };
    if ctx.is_null() { return Err("FATAL: Failed to allocate SSL_CTX".to_string()); }

    let safe_ctx = StormSslContext { inner: ctx };

    unsafe {
        // 1. KUNCI VERSI: Hanya TLS 1.2 dan 1.3
        bssl::SSL_CTX_set_min_proto_version(safe_ctx.inner, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(safe_ctx.inner, bssl::TLS1_3_VERSION as u16);

        // 2. MENCEGAH SESSION TRACKING OLEH WAF
        let cache_mode = bssl::SSL_SESS_CACHE_CLIENT | bssl::SSL_SESS_CACHE_NO_INTERNAL;
        bssl::SSL_CTX_set_session_cache_mode(safe_ctx.inner, cache_mode as i32);

        // 3. AKTIFKAN EXTENSIONS CHROME (ALPS, GREASE, ALPN)
        extensions::apply_chrome_extensions(safe_ctx.inner)?;

        // 4. BUG BOUNTY MODE (IGNORE CERT ERROR)
        bssl::SSL_CTX_set_verify(safe_ctx.inner, bssl::SSL_VERIFY_NONE as i32, None);
        
        // CATATAN ENGINEER:
        // Kita TIDAK menyentuh SSL_CTX_set_strict_cipher_list.
        // C++ BoringSSL otomatis merakit Cipher & Kurva (termasuk ML-KEM) 
        // 100% persis seperti browser Google Chrome versi terbaru.
    }
        
    Ok(safe_ctx)
}
