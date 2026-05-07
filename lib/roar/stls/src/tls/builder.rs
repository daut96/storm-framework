// src/tls/builder.rs
use crate::bssl; 
use super::{ciphers, extensions}; 

pub struct StormSslContext {
    pub inner: *mut bssl::SSL_CTX,
}

// Menjamin Tokio bahwa objek ini aman dipindah antar-thread pekerja (Worker Threads)
unsafe impl Send for StormSslContext {}
unsafe impl Sync for StormSslContext {}

impl Drop for StormSslContext {
    fn drop(&mut self) {
        unsafe {
            if !self.inner.is_null() {
                bssl::SSL_CTX_free(self.inner);
            }
        }
    }
}

pub fn build_chrome_ssl_context() -> Result<StormSslContext, String> {
    // 1. ENGINE ALLOCATION
    let ctx = unsafe { 
        let method = bssl::TLS_method();
        bssl::SSL_CTX_new(method)
    };

    if ctx.is_null() {
        return Err("FATAL: Failed to allocate SSL_CTX from BoringSSL".to_string());
    }

    // PERBAIKAN KRITIS: Bungkus pointer SEKARANG, sebelum melakukan pengecekan error apa pun.
    // Mulai dari baris ini, jika terjadi `return Err`, memori `ctx` dijamin tidak akan bocor.
    let safe_ctx = StormSslContext { inner: ctx };

    unsafe {
        // =================================================================
        // 2. PROTOCOL VERSION BOUNDARIES
        // Menggunakan konstanta bawaan BoringSSL lebih aman daripada magic number 0x0303
        // =================================================================
        bssl::SSL_CTX_set_min_proto_version(safe_ctx.inner, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(safe_ctx.inner, bssl::TLS1_3_VERSION as u16);

        // =================================================================
        // 3. CRYPTOGRAPHIC SUITES
        // =================================================================
        let ret_cipher = bssl::SSL_CTX_set_strict_cipher_list(safe_ctx.inner, ciphers::chrome_ciphers_ffi());
        if ret_cipher != 1 {
            return Err("FATAL: Failed to set unified Cipher List".to_string());
        }

        let ret_curves = bssl::SSL_CTX_set1_curves_list(safe_ctx.inner, ciphers::chrome_curves_ffi());
        if ret_curves != 1 {
            return Err("FATAL: Failed to set Modern Key Share Curves".to_string());
        }

        // =================================================================
        // 4. SESSION MANAGEMENT (CHROME BEHAVIOR)
        // =================================================================
        let cache_mode = bssl::SSL_SESS_CACHE_CLIENT | bssl::SSL_SESS_CACHE_NO_INTERNAL;
        bssl::SSL_CTX_set_session_cache_mode(safe_ctx.inner, cache_mode as i32);

        // =================================================================
        // 5. EXTENSIONS INJECTION
        // Jika fungsi ini gagal (Err), Rust akan otomatis drop(safe_ctx). Aman!
        // =================================================================
        extensions::apply_chrome_extensions(safe_ctx.inner)?;

        // =================================================================
        // 6. CERTIFICATE VERIFICATION
        // =================================================================
        bssl::SSL_CTX_set_verify(safe_ctx.inner, bssl::SSL_VERIFY_NONE as i32, None);
    }
        
    Ok(safe_ctx)
}
