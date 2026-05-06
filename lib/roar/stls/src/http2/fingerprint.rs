// src/http2/fingerprint.rs
use std::collections::HashMap;
use std::env;

pub struct ChromeH2Settings;

impl ChromeH2Settings {
    // Tetap pertahankan konstanta L4/L6 sebagai baseline
    pub const HEADER_TABLE_SIZE: u32 = 65536;
    pub const ENABLE_PUSH: bool = false;
    pub const MAX_CONCURRENT_STREAMS: u32 = 1000;
    pub const INITIAL_WINDOW_SIZE: u32 = 6291456; 
    pub const MAX_HEADER_LIST_SIZE: u32 = 262144;
    pub const CONNECTION_WINDOW_UPDATE: u32 = 15663105;

    pub fn get_pseudo_header_order() -> Vec<&'static str> {
        vec![":method", ":authority", ":scheme", ":path"]
    }

    /// Menghasilkan Base Headers secara dinamis berdasarkan OS saat ini (Android/Linux).
    /// Versi target: Chrome 145 (Mei 2026).
    pub fn generate_dynamic_headers() -> Vec<(String, String)> {
        let os = env::consts::OS;
        let chrome_v = "145"; // Target versi Mei 2026
        
        // Deteksi spesifik untuk Android (Termux) vs Linux Desktop/Server
        let (platform, ua, mobile) = match os {
            "android" => (
                "\"Android\"",
                format!("Mozilla/5.0 (Linux; Android 15; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{}.0.0.0 Mobile Safari/537.36", chrome_v),
                "?1"
            ),
            "linux" => (
                "\"Linux\"",
                format!("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{}.0.0.0 Safari/537.36", chrome_v),
                "?0"
            ),
            _ => (
                "\"Linux\"",
                format!("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{}.0.0.0 Safari/537.36", chrome_v),
                "?0"
            ),
        };

        let sec_ch_ua = format!("\"Not(A:Brand\";v=\"99\", \"Google Chrome\";v=\"{0}\", \"Chromium\";v=\"{0}\"", chrome_v);

        vec![
            ("sec-ch-ua".to_string(), sec_ch_ua),
            ("sec-ch-ua-mobile".to_string(), mobile.to_string()),
            ("sec-ch-ua-platform".to_string(), platform.to_string()),
            ("upgrade-insecure-requests".to_string(), "1".to_string()),
            ("user-agent".to_string(), ua),
            ("accept".to_string(), "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7".to_string()),
            ("sec-fetch-site".to_string(), "none".to_string()),
            ("sec-fetch-mode".to_string(), "navigate".to_string()),
            ("sec-fetch-user".to_string(), "?1".to_string()),
            ("sec-fetch-dest".to_string(), "document".to_string()),
            ("accept-encoding".to_string(), "gzip, deflate, br, zstd".to_string()),
            ("accept-language".to_string(), "en-US,en;q=0.9".to_string()),
        ]
    }
}
