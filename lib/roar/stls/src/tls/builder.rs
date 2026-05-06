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
        // Chrome 145+ (Mei 2026) sangat ketat pada TLS 1.2 dan 1.3
        // =================================================================
        bssl::SSL_CTX_set_min_proto_version(ctx, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(ctx, bssl::TLS1_3_VERSION as u16);

        // =================================================================
        // 3. CRYPTOGRAPHIC SUITES (PERBAIKAN KRITIS)
        // PERHATIKAN: BoringSSL Hulu tidak menggunakan SSL_CTX_set_ciphersuites.
        // Cukup satu fungsi SSL_CTX_set_strict_cipher_list untuk semua versi.
        // =================================================================
        
        // Memanggil fungsi tunggal yang sudah kita buat di ciphers.rs (OS-Aware)
        let ret_cipher = bssl::SSL_CTX_set_strict_cipher_list(ctx, ciphers::chrome_ciphers_ffi());
        if ret_cipher != 1 {
            return Err("FATAL: Failed to set unified Cipher List".to_string());
        }

        // Atur Key Share Curves (Wajib menyertakan ML-KEM untuk Chrome 145+)
        let ret_curves = bssl::SSL_CTX_set1_curves_list(ctx, ciphers::chrome_curves_ffi());
        if ret_curves != 1 {
            return Err("FATAL: Failed to set Modern Key Share Curves".to_string());
        }

        // =================================================================
        // 4. SESSION MANAGEMENT (CHROME BEHAVIOR)
        // Chrome 145+ mengandalkan pembersihan session cache yang agresif 
        // untuk mencegah tracking (Fingerprint protection).
        // =================================================================
        let cache_mode = bssl::SSL_SESS_CACHE_CLIENT | bssl::SSL_SESS_CACHE_NO_INTERNAL;
        bssl::SSL_CTX_set_session_cache_mode(ctx, cache_mode as i32);

        // =================================================================
        // 5. EXTENSIONS INJECTION
        // Menyuntikkan GREASE, ALPN (h2), ALPS (Payload HTTP/2 Settings), 
        // OCSP, SCT, dan SigAlgs (termasuk ed25519).
        // =================================================================
        extensions::apply_chrome_extensions(ctx)?;

        // =================================================================
        // 6. CERTIFICATE VERIFICATION
        // OSINT / Evasion mode: Ignore certificate errors untuk bypass proxy/MITM.
        // =================================================================
        bssl::SSL_CTX_set_verify(ctx, bssl::SSL_VERIFY_NONE as i32, None);
        
        Ok(StormSslContext { inner: ctx })
    }
}
