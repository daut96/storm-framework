// src/tls/builder.rs
use boring::ssl::{SslContext, SslContextBuilder, SslMethod, SslVersion, SslVerifyMode};
use super::{ciphers, extensions};

pub fn build_chrome_ssl_context() -> Result<SslContext, boring::error::ErrorStack> {
    // Menggunakan TLS Client method standar dari BoringSSL
    let mut builder = SslContextBuilder::new(SslMethod::tls_client())?;

    // 1. Set Min/Max Protocol Version
    // Ini lebih idiomatik dan aman daripada menghapus opsi (NO_SSLV3, dll) satu per satu.
    builder.set_min_proto_version(Some(SslVersion::TLS1_2))?;
    builder.set_max_proto_version(Some(SslVersion::TLS1_3))?;

    // 2. Set Cipher Suites (Presisi Chrome)
    // Ingat: set_cipher_list HANYA untuk TLS <= 1.2
    builder.set_cipher_list(ciphers::chrome_tls12_ciphers())?;
    
    // Untuk TLS 1.3, WAJIB menggunakan metode set_ciphersuites()
    // Contoh string yang diharapkan: "TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256"
    // builder.set_ciphersuites(ciphers::chrome_tls13_ciphersuites())?;

    // 3. Set Curves (X25519 atau Kyber hibrida untuk Chrome 120+)
    builder.set_curves_list(ciphers::chrome_curves())?;

    // 4. ALPN (Application-Layer Protocol Negotiation)
    // Wajib diset di awal agar WAF melihat kita mendukung HTTP/2 sejak ClientHello.
    // Format C-ABI untuk ALPN adalah length-prefixed string, jadi \x02 untuk "h2" (2 byte) 
    // dan \x08 untuk "http/1.1" (8 byte).
    builder.set_alpn_protos(b"\x02h2\x08http/1.1")?;

    // 5. GREASE (Generate Random Extensions And Sustain Extensibility)
    // BoringSSL punya native support untuk GREASE, kita bisa langsung aktifkan di builder
    // tanpa perlu merakit byte GREASE manual di ekstensi.
    builder.set_grease_enabled(true);

    // 6. Mode Verifikasi Sertifikat
    // Untuk tools OSINT / Bug Bounty, seringkali kita butuh mengabaikan SSL error 
    // (misalnya saat melewati proxy/MITM lokal). Bisa disesuaikan jadi VERIFY_PEER untuk production strict.
    builder.set_verify(SslVerifyMode::NONE);

    // 7. Terapkan Ekstensi Custom lainnya (seperti Signature Algorithms, dll)
    extensions::apply_chrome_extensions(&mut builder)?;

    Ok(builder.build())
}
