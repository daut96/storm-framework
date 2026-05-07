// src/tls/extensions.rs
use crate::bssl;

// =====================================================================
// DATA STATIS (Aman dari MTE, dialokasikan di .rodata binary)
// =====================================================================

static ALPS_PROTO: &[u8] = b"h2";

static ALPS_SETTINGS: &[u8] = &[
    0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 
    0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 
    0x00, 0x03, 0x00, 0x00, 0x03, 0xe8, 
    0x00, 0x04, 0x00, 0x60, 0x00, 0x00, 
    0x00, 0x06, 0x00, 0x04, 0x00, 0x00, 
];

pub fn apply_chrome_extensions(ctx: *mut bssl::SSL_CTX) -> Result<(), String> {
    unsafe {
        bssl::SSL_CTX_set_grease_enabled(ctx, 1);

        let alpn_protos = b"\x02h2\x08http/1.1";
        let alpn_res = bssl::SSL_CTX_set_alpn_protos(
            ctx, 
            alpn_protos.as_ptr(), 
            alpn_protos.len() as usize
        );
        if alpn_res != 0 {
            return Err("FATAL: Failed to inject ALPN into SSL_CTX".to_string());
        }

        bssl::SSL_CTX_enable_ocsp_stapling(ctx);
        bssl::SSL_CTX_enable_signed_cert_timestamps(ctx);

        // Bagus! Penambahan "\0" di akhir adalah praktik FFI yang sangat presisi.
        let sigalgs = concat!(
            "ed25519:",
            "ecdsa_secp256r1_sha256:",
            "rsa_pss_rsae_sha256:",
            "rsa_pkcs1_sha256:",
            "ecdsa_secp384r1_sha384:",
            "rsa_pss_rsae_sha384:",
            "rsa_pkcs1_sha384:",
            "rsa_pss_rsae_sha512:",
            "rsa_pkcs1_sha512",
            "\0" 
        );
        
        let sig_res = bssl::SSL_CTX_set1_sigalgs_list(ctx, sigalgs.as_ptr() as *const _);
        if sig_res != 1 {
            return Err("FATAL: Failed to set Signature Algorithms".to_string());
        }

        Ok(())
    }
}

pub fn apply_alps_extension(ssl: *mut bssl::SSL) -> Result<(), String> {
    unsafe {
        // Menggunakan pointer statis yang valid selama program berjalan
        let alps_res = bssl::SSL_add_application_settings(
            ssl, 
            ALPS_PROTO.as_ptr(),
            ALPS_PROTO.len(),
            ALPS_SETTINGS.as_ptr(),
            ALPS_SETTINGS.len(),
        );

        if alps_res != 1 {
            return Err("FATAL: Failed to inject ALPS".to_string());
        }
        Ok(())
    }
}
