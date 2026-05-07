// src/tls/ciphers.rs

/// Supported Groups (Curves) untuk proses Key Share
pub fn chrome_curves_ffi() -> *const std::os::raw::c_char {
    concat!(
        "X25519:",
        "P-256:",
        "P-384",
        "\0" 
    ).as_ptr() as *const std::os::raw::c_char
}

/// ====================================================================
/// KONDISI 1: JIKA DI-COMPILE UNTUK ANDROID (TERMUX)
/// ====================================================================
#[cfg(target_os = "android")]
pub fn chrome_ciphers_ffi() -> *const std::os::raw::c_char {
    // PERBAIKAN 2: Hapus TLS 1.3. BoringSSL akan menginjeksikannya secara otomatis.
    concat!(
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "AES128-GCM-SHA256:",
        // PERBAIKAN 3: Hapus titik dua (:) pada cipher terakhir!
        "AES256-GCM-SHA384",
        "\0"
    ).as_ptr() as *const std::os::raw::c_char
}

/// ====================================================================
/// KONDISI 2: JIKA DI-COMPILE UNTUK LINUX/WINDOWS/MACOS STANDAR
/// ====================================================================
#[cfg(not(target_os = "android"))]
pub fn chrome_ciphers_ffi() -> *const std::os::raw::c_char {
    // Desktop Chrome Behavior
    concat!(
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        "AES128-GCM-SHA256:",
        // PERBAIKAN 3: Hapus titik dua (:) pada cipher terakhir!
        "AES256-GCM-SHA384",
        "\0"
    ).as_ptr() as *const std::os::raw::c_char
}
