// src/tls/ciphers.rs

/// Supported Groups (Curves) untuk proses Key Share
/// Standar Chrome 140+ (Menggunakan Post-Quantum ML-KEM FIPS 203)
pub fn chrome_curves_ffi() -> *const std::os::raw::c_char {
    concat!(
        "X25519MLKEM768:", 
        "X25519:",
        "P-256:",
        "P-384",
        "\0" // Null-terminator wajib untuk FFI
    ).as_ptr() as *const std::os::raw::c_char
}

/// ====================================================================
/// KONDISI 1: JIKA DI-COMPILE UNTUK ANDROID (TERMUX)
/// ====================================================================
#[cfg(target_os = "android")]
pub fn chrome_ciphers_ffi() -> *const std::os::raw::c_char {
    // Android / ARM Chrome Behavior: Prioritaskan ChaCha20_Poly1305
    concat!(
        // TLS 1.3 Ciphers
        "TLS_CHACHA20_POLY1305_SHA256:",
        "TLS_AES_128_GCM_SHA256:",
        "TLS_AES_256_GCM_SHA384:",
        // TLS 1.2 Ciphers
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "AES128-GCM-SHA256:",
        "AES256-GCM-SHA384:",
        "\0"
    ).as_ptr() as *const std::os::raw::c_char
}

/// ====================================================================
/// KONDISI 2: JIKA DI-COMPILE UNTUK LINUX/WINDOWS/MACOS STANDAR
/// ====================================================================
#[cfg(not(target_os = "android"))]
pub fn chrome_ciphers_ffi() -> *const std::os::raw::c_char {
    // Desktop Chrome Behavior: Prioritaskan AES (Hardware AES-NI)
    concat!(
        // TLS 1.3 Ciphers
        "TLS_AES_128_GCM_SHA256:",
        "TLS_AES_256_GCM_SHA384:",
        "TLS_CHACHA20_POLY1305_SHA256:",
        // TLS 1.2 Ciphers
        "ECDHE-ECDSA-AES128-GCM-SHA256:",
        "ECDHE-RSA-AES128-GCM-SHA256:",
        "ECDHE-ECDSA-AES256-GCM-SHA384:",
        "ECDHE-RSA-AES256-GCM-SHA384:",
        "ECDHE-ECDSA-CHACHA20-POLY1305:",
        "ECDHE-RSA-CHACHA20-POLY1305:",
        "AES128-GCM-SHA256:",
        "AES256-GCM-SHA384:",
        "\0"
    ).as_ptr() as *const std::os::raw::c_char
}
