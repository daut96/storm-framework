// src/tls/extensions.rs
use crate::bssl;

// ALPS Payload (HTTP/2 Settings) -> Ini yang akan menembus Cloudflare OpenAI!
static ALPS_PROTO: &[u8] = b"h2";
static ALPS_SETTINGS: &[u8] = &[
    0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 
    0x00, 0x03, 0x00, 0x00, 0x03, 0xe8, 0x00, 0x04, 0x00, 0x60, 0x00, 0x00, 
    0x00, 0x06, 0x00, 0x04, 0x00, 0x00, 
];

pub fn apply_chrome_extensions(ctx: *mut bssl::SSL_CTX) -> Result<(), String> {
    unsafe {
        // 1. GREASE (Fitur Chrome untuk membingungkan WAF)
        bssl::SSL_CTX_set_grease_enabled(ctx, 1);

        // 2. ALPN (Wajib untuk HTTP/2)
        let alpn_protos = b"\x02h2\x08http/1.1";
        bssl::SSL_CTX_set_alpn_protos(ctx, alpn_protos.as_ptr(), alpn_protos.len() as usize);

        // 3. Sertifikat Transparansi & OCSP
        bssl::SSL_CTX_enable_ocsp_stapling(ctx);
        bssl::SSL_CTX_enable_signed_cert_timestamps(ctx);
        
        // 4. (Opsional tapi direkomendasikan) Permintaan Ekstensi Sertifikat Terkompresi
        // Chrome modern menghemat bandwidth dengan ini. Cloudflare mengeceknya.
        // bssl::SSL_CTX_set_cert_compression_algs(ctx, ...); // Jika modul kompresi aktif di C++
        
        Ok(())
    }
}

// Fungsi ini akan dipanggil SETELAH koneksi TCP terbentuk, tepat sebelum handshake.
// Tapi untuk kemudahan arsitektur saat ini, kita sudah menggunakan default C++.
pub fn apply_alps_extension(ssl: *mut bssl::SSL) -> Result<(), String> {
    unsafe {
        bssl::SSL_add_application_settings(
            ssl, ALPS_PROTO.as_ptr(), ALPS_PROTO.len(),
            ALPS_SETTINGS.as_ptr(), ALPS_SETTINGS.len(),
        );
        Ok(())
    }
}
