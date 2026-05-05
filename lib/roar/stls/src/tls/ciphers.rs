// src/tls/ciphers.rs

/// Mengembalikan cipher suites khusus untuk TLS 1.3.
/// Fungsi ini akan dimasukkan ke `builder.set_ciphersuites()`.
/// Urutan ini mencerminkan apa yang dikirim oleh Chrome 120+ modern.
pub fn chrome_tls13_ciphersuites() -> &'static str {
    // OpenSSL/BoringSSL parser untuk TLS 1.3 menggunakan nama standar RFC
    concat!(
        "TLS_AES_128_GCM_SHA256:",
        "TLS_AES_256_GCM_SHA384:",
        "TLS_CHACHA20_POLY1305_SHA256"
    )
}

/// Mengembalikan cipher suites untuk TLS 1.2 dan fallback.
/// Fungsi ini akan dimasukkan ke `builder.set_cipher_list()`.
/// Perhatikan penamaannya menggunakan format OpenSSL, bukan RFC.
pub fn chrome_tls12_ciphers() -> &'static str {
    // Ekstraksi fallback TLS 1.2 persis seperti payload ClientHello Chrome
    concat!(
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        // Chrome juga sering mengirim cipher lawas ini di ekor ClientHello 
        // untuk kompatibilitas server lama. Akamai akan mengecek keberadaan ini.
        "ECDHE-RSA-AES128-SHA:",
        "ECDHE-RSA-AES256-SHA:",
        "AES128-GCM-SHA256:",
        "AES256-GCM-SHA384:",
        "AES128-SHA:",
        "AES256-SHA"
    )
}

/// Supported Groups (Curves) untuk proses Key Share
pub fn chrome_curves() -> &'static str {
    // WAF Modern (seperti Cloudflare Turnstile) mengecek adopsi PQC (Post-Quantum Cryptography).
    // Chrome 120+ default menggunakan X25519Kyber768Draft00 (di BoringSSL sering direpresentasikan sebagai x25519_kyber768).
    // Jika server tidak dukung PQC, otomatis fallback ke X25519.
    concat!(
        "X25519Kyber768Draft00:", // Hibrida Post-Quantum (Sangat krusial untuk trust-score tinggi)
        "X25519:",
        "P-256:",
        "P-384"
    )
}
