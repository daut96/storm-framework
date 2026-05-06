// src/tls/builder.rs
use crate::bssl; // Mengacu pada bindings yang dibuat di lib.rs
use std::ptr;

// Wajib import modul saudara agar kompilator tidak panic
use super::{ciphers, extensions}; 

pub struct StormSslContext {
    pub inner: *mut bssl::SSL_CTX,
}

// Implementasi Drop agar tidak terjadi memory leak pada pointer C
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
        // 1. Inisialisasi Context menggunakan method TLS Client murni Google
        let method = bssl::TLS_client_method();
        let ctx = bssl::SSL_CTX_new(method);

        if ctx.is_null() {
            return Err("FATAL: Failed to allocate SSL_CTX from BoringSSL".to_string());
        }

        // 2. Set Protocol Version (Wajib TLS 1.2 sampai 1.3 untuk Chrome modern)
        bssl::SSL_CTX_set_min_proto_version(ctx, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(ctx, bssl::TLS1_3_VERSION as u16);

        // 3. Konfigurasi Ciphers dan Curves 
        // Langsung menggunakan string FFI Zero-Cost yang sudah kita optimasi
        bssl::SSL_CTX_set_strict_cipher_list(ctx, ciphers::chrome_tls12_ciphers_ffi());
        bssl::SSL_CTX_set1_curves_list(ctx, ciphers::chrome_curves_ffi());

        // 4. Delegasi Ekstensi (GREASE, ALPN, ALPS, OCSP, SCT, SigAlgs)
        // Semua kompleksitas manipulasi frame didelegasikan ke modul extensions.rs
        extensions::apply_chrome_extensions(ctx)?;

        // 5. Verifikasi Sertifikat 
        // None untuk OSINT/Evasion agar tidak gagal saat bertemu proxy/MITM lokal
        bssl::SSL_CTX_set_verify(ctx, bssl::SSL_VERIFY_NONE as i32, None);
        
        Ok(StormSslContext { inner: ctx })
    }
}
