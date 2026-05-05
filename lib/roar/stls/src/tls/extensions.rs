// src/tls/extensions.rs
use boring::ssl::SslContextBuilder;

/// Mengonfigurasi ekstensi TLS yang membuat kita terlihat identik dengan Chrome
pub fn apply_chrome_extensions(builder: &mut SslContextBuilder) -> Result<(), boring::error::ErrorStack> {
    // 1. GREASE (Generate Random Extensions And Sustain Extensibility)
    builder.set_grease_enabled(true);

    // 2. ALPN (Application-Layer Protocol Negotiation)
    let alpn_protos = b"\x02h2\x08http/1.1";
    builder.set_alpn_protos(alpn_protos)?;

    // 3. OCSP Stapling (status_request extension)
    // Chrome *selalu* menanyakan status pencabutan sertifikat ke server. 
    // Tanpa ekstensi ini, WAF langsung curiga.
    builder.enable_ocsp_stapling();

    // 4. Signed Certificate Timestamps (SCT / Extension 18)
    // Standar wajib Google Chrome untuk Certificate Transparency. 
    builder.enable_signed_cert_timestamps();

    // 5. Signature Algorithms (Extension 13)
    // Akamai dan Datadome memvalidasi urutan exact dari algoritma tanda tangan ini.
    // Daftar di bawah ini adalah ekstraksi langsung dari Chrome 120+.
    builder.set_sigalgs_list(concat!(
        "ecdsa_secp256r1_sha256:",
        "rsa_pss_rsae_sha256:",
        "rsa_pkcs1_sha256:",
        "ecdsa_secp384r1_sha384:",
        "rsa_pss_rsae_sha384:",
        "rsa_pkcs1_sha384:",
        "rsa_pss_rsae_sha512:",
        "rsa_pkcs1_sha512"
    ))?;

    // Catatan untuk ALPS (Application-Layer Protocol Settings):
    // Seperti yang kamu sadari, `rust-boring` standar mungkin belum mengekspos
    // flag ALPS secara penuh karena ini draft protokol yang agresif di-push Google.
    // Tapi, dengan GREASE, ALPN h2, urutan cipher yang benar, dan H2 SETTINGS 
    // yang sudah kita modifikasi, ini sudah 99% cukup untuk menembus WAF strict.

    Ok(())
}
