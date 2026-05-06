// src/tls/ciphers.rs

/// Mengembalikan cipher suites untuk TLS 1.2 dan fallback.
/// Menggunakan null-terminator (\0) di akhir string agar 100% kompatibel 
/// dengan C-ABI secara Zero-Cost, siap dilempar ke pointer C.
pub fn chrome_tls12_ciphers_ffi() -> *const std::os::raw::c_char {
    concat!(
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-AES128-SHA:",
        "ECDHE-RSA-AES256-SHA:",
        "AES128-GCM-SHA256:",
        "AES256-GCM-SHA384:",
        "AES128-SHA:",
        "AES256-SHA",
        "\0" // Null-terminator wajib untuk FFI
    ).as_ptr() as *const std::os::raw::c_char
}

/// Supported Groups (Curves) untuk proses Key Share
pub fn chrome_curves_ffi() -> *const std::os::raw::c_char {
    // UPDATE TERBARU: Hulu BoringSSL telah beralih dari KyberDraft00 ke MLKEM.
    // X25519MLKEM768 adalah standar hibrida Post-Quantum terbaru milik Google.
    concat!(
        "X25519MLKEM768:", 
        "X25519:",
        "P-256:",
        "P-384",
        "\0" // Null-terminator wajib untuk FFI
    ).as_ptr() as *const std::os::raw::c_char
}

pub fn chrome_tls13_ciphersuites_ffi() -> *const std::os::raw::c_char {
    concat!(
        "TLS_AES_128_GCM_SHA256:",
        "TLS_AES_256_GCM_SHA384:",
        "TLS_CHACHA20_POLY1305_SHA256",
        "\0" // Null-terminator untuk FFI
    ).as_ptr() as *const std::os::raw::c_char
}
