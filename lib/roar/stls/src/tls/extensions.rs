// src/tls/extensions.rs
use crate::bssl;

/// Mengonfigurasi ekstensi TLS langsung ke memori BoringSSL 
/// untuk membuat fingerprint identik dengan Chrome versi terbaru (145+).
pub fn apply_chrome_extensions(ctx: *mut bssl::SSL_CTX) -> Result<(), String> {
    unsafe {
        // 1. GREASE (Generate Random Extensions And Sustain Extensibility)
        bssl::SSL_CTX_set_grease_enabled(ctx, 1);

        // 2. ALPN (Application-Layer Protocol Negotiation)
        let alpn_protos = b"\x02h2\x08http/1.1";
        let alpn_res = bssl::SSL_CTX_set_alpn_protos(
            ctx, 
            alpn_protos.as_ptr(), 
            alpn_protos.len() as u32
        );
        if alpn_res != 0 {
            return Err("FATAL: Failed to inject ALPN into SSL_CTX".to_string());
        }

        // 3. OCSP Stapling (status_request extension)
        bssl::SSL_CTX_enable_ocsp_stapling(ctx);

        // 4. Signed Certificate Timestamps (SCT / Extension 18)
        bssl::SSL_CTX_enable_signed_cert_timestamps(ctx);

        // 5. Signature Algorithms (Extension 13)
        // PERBAIKAN: Menambahkan 'ed25519' di awal, sesuai standar Chrome modern.
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
            "\0" // Null-terminator wajib
        );
        let sig_res = bssl::SSL_CTX_set1_sigalgs_list(ctx, sigalgs.as_ptr() as *const i8);
        if sig_res != 1 {
            return Err("FATAL: Failed to set Signature Algorithms".to_string());
        }

        // 6. ALPS (Application-Layer Protocol Settings)
        let alps_proto = b"h2";
        
        // PERBAIKAN FATAL: ALPS tidak boleh kosong! 
        // Chrome mengirimkan representasi biner dari HTTP/2 SETTINGS di dalam ekstensi ini.
        // Byte sequence di bawah ini adalah representasi dari:
        // - HEADER_TABLE_SIZE: 65536 (0x0001 -> 0x00010000)
        // - ENABLE_PUSH: 0 (0x0002 -> 0x00000000)
        // - MAX_CONCURRENT_STREAMS: 1000 (0x0003 -> 0x000003E8)
        // - INITIAL_WINDOW_SIZE: 6291456 (0x0004 -> 0x00600000)
        // - MAX_HEADER_LIST_SIZE: 262144 (0x0006 -> 0x00040000)
        let alps_settings: [u8; 30] = [
            0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // HEADER_TABLE_SIZE
            0x00, 0x02, 0x00, 0x00, 0x00, 0x00, // ENABLE_PUSH
            0x00, 0x03, 0x00, 0x00, 0x03, 0xe8, // MAX_CONCURRENT_STREAMS
            0x00, 0x04, 0x00, 0x60, 0x00, 0x00, // INITIAL_WINDOW_SIZE
            0x00, 0x06, 0x00, 0x04, 0x00, 0x00, // MAX_HEADER_LIST_SIZE
        ]; 
        
        let alps_res = bssl::SSL_CTX_add_application_settings(
            ctx,
            alps_proto.as_ptr(),
            alps_proto.len(),
            alps_settings.as_ptr(),
            alps_settings.len(),
        );
        if alps_res != 1 {
            return Err("FATAL: Failed to inject ALPS (Application-Layer Protocol Settings) extension".to_string());
        }

        Ok(())
    }
}
