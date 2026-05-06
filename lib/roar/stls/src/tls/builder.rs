// src/tls/builder.rs
use crate::bssl; // Mengacu pada bindings yang dibuat di lib.rs
use std::ptr;

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
            return Err("Failed to allocate SSL_CTX from BoringSSL".to_string());
        }

        // 2. Set Protocol Version (TLS 1.2 sampai 1.3)
        bssl::SSL_CTX_set_min_proto_version(ctx, bssl::TLS1_2_VERSION as u16);
        bssl::SSL_CTX_set_max_proto_version(ctx, bssl::TLS1_3_VERSION as u16);

        // 3. Aktifkan GREASE secara Native
        // Ini menggantikan builder.set_grease_enabled(true)
        bssl::SSL_CTX_set_grease_enabled(ctx, 1);

        // 4. Konfigurasi Cipher Suites (TLS 1.2 & TLS 1.3)
        // Chrome menggunakan urutan strict untuk menghindari fingerprinting
        let cipher_list = b"TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256\0";
        bssl::SSL_CTX_set_strict_cipher_list(ctx, cipher_list.as_ptr() as *const i8);

        // 5. Konfigurasi ALPN
        // Format: length-prefixed (0x02 untuk "h2", 0x08 untuk "http/1.1")
        let alpn = b"\x02h2\x08http/1.1";
        bssl::SSL_CTX_set_alpn_protos(ctx, alpn.as_ptr(), alpn.len() as u32);

        // 6. SOLUSI UTAMA: Implementasi ALPS (Application-Layer Protocol Settings)
        // Ini adalah fitur yang tidak ada di crate 'boring' tapi ada di hulu Google
        let alps_proto = b"h2";
        // Settings ALPS Chrome (biasanya berisi pengaturan HTTP/2 SETTINGS frame di awal)
        let alps_settings = b"\x00\x00\x00\x00\x00"; 
        bssl::SSL_CTX_add_application_settings(
            ctx,
            alps_proto.as_ptr(),
            alps_proto.len(),
            alps_settings.as_ptr(),
            alps_settings.len(),
        );

        // 7. Verifikasi Sertifikat (None untuk keperluan OSINT/Evasion)
        bssl::SSL_CTX_set_verify(ctx, bssl::SSL_VERIFY_NONE as i32, None);

        // Set Cipher list menggunakan string FFI yang sudah kita optimasi
        bssl::SSL_CTX_set_strict_cipher_list(ctx, ciphers::chrome_tls12_ciphers_ffi());

        // Set Curves menggunakan fungsi BoringSSL native (contoh FFI name: SSL_CTX_set1_curves_list)
        bssl::SSL_CTX_set1_curves_list(ctx, ciphers::chrome_curves_ffi());

        Ok(StormSslContext { inner: ctx })
    }
}
