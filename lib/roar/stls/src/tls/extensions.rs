// src/tls/extensions.rs
use crate::bssl;

/// Mengonfigurasi ekstensi TLS langsung ke memori BoringSSL 
/// untuk membuat fingerprint identik dengan Chrome versi terbaru.
pub fn apply_chrome_extensions(ctx: *mut bssl::SSL_CTX) -> Result<(), String> {
    unsafe {
        // 1. GREASE (Generate Random Extensions And Sustain Extensibility)
        // Disuntikkan secara dinamis oleh BoringSSL untuk mengecoh middlebox statis.
        bssl::SSL_CTX_set_grease_enabled(ctx, 1);

        // 2. ALPN (Application-Layer Protocol Negotiation)
        // Memberitahu server kita mendukung HTTP/2 sejak awal handshake.
        let alpn_protos = b"\x02h2\x08http/1.1";
        let alpn_res = bssl::SSL_CTX_set_alpn_protos(
            ctx, 
            alpn_protos.as_ptr(), 
            alpn_protos.len() as u32
        );
        // Catatan FFI: Fungsi ALPN BoringSSL mengembalikan 0 jika sukses.
        if alpn_res != 0 {
            return Err("FATAL: Failed to inject ALPN into SSL_CTX".to_string());
        }

        // 3. OCSP Stapling (status_request extension)
        // Wajib untuk meniru Chrome agar WAF seperti Akamai/Datadome tidak curiga.
        bssl::SSL_CTX_enable_ocsp_stapling(ctx);

        // 4. Signed Certificate Timestamps (SCT / Extension 18)
        // Bukti Certificate Transparency yang selalu diminta oleh browser Chrome asli.
        bssl::SSL_CTX_enable_signed_cert_timestamps(ctx);

        // 5. Signature Algorithms (Extension 13)
        // Trik Zero-Cost FFI: Tambahkan \0 di akhir string agar kompatibel dengan C-ABI.
        let sigalgs = concat!(
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

        // 6. ALPS (Application-Layer Protocol Settings) - EKSEKUSI FITUR EKSKLUSIF
        // Fitur ini tidak ada di rust-boring. Dengan ini, Storm Framework resmi
        // setara dengan Chromium engine dalam manipulasi handshake.
        let alps_proto = b"h2";
        // Di sini Anda bisa menyuntikkan payload frame SETTINGS HTTP/2 mentah 
        // yang disepakati lebih awal. Untuk default evasion, string kosong sudah memicu
        // ekstensi ALPS muncul di paket ClientHello.
        let alps_settings = b""; 
        
        let alps_res = bssl::SSL_CTX_add_application_settings(
            ctx,
            alps_proto.as_ptr(),
            alps_proto.len(),
            alps_settings.as_ptr(),
            alps_settings.len(),
        );
        if alps_res != 1 {
            return Err("FATAL: Failed to inject ALPS (Application-Layer Protocol Settings) extension)".to_string());
        }

        Ok(())
    }
}
