// src/tls/builder.rs
use crate::bssl; 
use std::ptr;
use super::{ciphers, extensions}; 

pub struct StormSslContext {
    pub inner: *mut bssl::SSL_CTX,
}

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
    unsafe {
        // =================================================================
        // 1. ENGINE ALLOCATION
        // =================================================================
        let method = bssl::TLS_client_method();
        let ctx = bssl::SSL_CTX_new(method);

        if ctx.is_null() {
            return Err("FATAL: Failed to allocate SSL_CTX from BoringSSL".to_string());
        }

        // =================================================================
        // 2. PROTOCOL VERSION BOUNDARIES
        // Chrome 120+ sangat strict menolak TLS 1.1 ke bawah
        // =================================================================
        bssl::SSL_CTX_set_min_proto_version(ctx, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(ctx, bssl::TLS1_3_VERSION as u16);

        // =================================================================
        // 3. CRYPTOGRAPHIC SUITES (PERBAIKAN FATAL)
        // BoringSSL membedakan pemanggilan fungsi untuk TLS 1.2 dan TLS 1.3!
        // =================================================================
        // Eksekusi untuk TLS <= 1.2 (Fallback)
        let ret_12 = bssl::SSL_CTX_set_strict_cipher_list(ctx, ciphers::chrome_tls12_ciphers_ffi());
        if ret_12 != 1 {
            return Err("Failed to set TLS 1.2 Cipher List".to_string());
        }

        // Eksekusi khusus untuk TLS 1.3 (Sering menyebabkan handshake gagal jika terlewat)
        let ret_13 = bssl::SSL_CTX_set_ciphersuites(ctx, ciphers::chrome_tls13_ciphersuites_ffi());
        if ret_13 != 1 {
            return Err("Failed to set TLS 1.3 Ciphersuites".to_string());
        }

        // Atur Key Share Curves (Termasuk Post-Quantum ML-KEM)
        let ret_curves = bssl::SSL_CTX_set1_curves_list(ctx, ciphers::chrome_curves_ffi());
        if ret_curves != 1 {
            return Err("Failed to set Key Share Curves".to_string());
        }

        // =================================================================
        // 4. SESSION MANAGEMENT (CHROME BEHAVIOR)
        // Chrome menggunakan memory cache kliennya sendiri, bukan internal OpenSSL.
        // Konfigurasi ini mengurangi fingerprint anomali pada Session Ticket.
        // =================================================================
        let cache_mode = bssl::SSL_SESS_CACHE_CLIENT | bssl::SSL_SESS_CACHE_NO_INTERNAL;
        bssl::SSL_CTX_set_session_cache_mode(ctx, cache_mode as i32);

        // =================================================================
        // 5. EXTENSIONS INJECTION (The "Missing" Code)
        // Di sinilah GREASE, ALPN (h2), ALPS, OCSP, SCT, dan SigAlgs disuntikkan.
        // Kita mendelegasikannya agar file builder.rs tidak menjadi file raksasa (monolithic).
        // =================================================================
        extensions::apply_chrome_extensions(ctx)?;

        // =================================================================
        // 6. CERTIFICATE VERIFICATION
        // OSINT / Evasion mode: Ignore certificate errors
        // =================================================================
        bssl::SSL_CTX_set_verify(ctx, bssl::SSL_VERIFY_NONE as i32, None);
        
        Ok(StormSslContext { inner: ctx })
    }
}
        
