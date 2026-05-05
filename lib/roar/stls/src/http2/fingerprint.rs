// src/http2/fingerprint.rs

/// Konstanta HTTP/2 SETTINGS Frame untuk meniru Chrome 120+
/// Target Bypass: Akamai Bot Manager, Cloudflare Turnstile, Datadome
pub struct ChromeH2Settings;

impl ChromeH2Settings {
    // 1: SETTINGS_HEADER_TABLE_SIZE
    pub const HEADER_TABLE_SIZE: u32 = 65536;
    
    // 2: SETTINGS_ENABLE_PUSH (Chrome selalu set ke 0 / false)
    pub const ENABLE_PUSH: bool = false;
    
    // 3: SETTINGS_MAX_CONCURRENT_STREAMS
    pub const MAX_CONCURRENT_STREAMS: u32 = 1000;
    
    // 4: SETTINGS_INITIAL_WINDOW_SIZE
    pub const INITIAL_WINDOW_SIZE: u32 = 6291456; 
    
    // 5: SETTINGS_MAX_FRAME_SIZE (Chrome biasanya membiarkan default di 16384, tidak dikirim)
    
    // 6: SETTINGS_MAX_HEADER_LIST_SIZE (Sangat penting untuk Akamai)
    pub const MAX_HEADER_LIST_SIZE: u32 = 262144;

    // Connection-level Window Update Frame (Chrome signature)
    // Chrome mengirimkan window update sebesar ini segera setelah SETTINGS frame
    pub const CONNECTION_WINDOW_UPDATE: u32 = 15663105;

    /// Urutan pseudo-header yang wajib dipatuhi.
    /// WAF akan memblokir jika :path berada sebelum :method.
    pub fn get_pseudo_header_order() -> Vec<&'static str> {
        vec![
            ":method",
            ":authority",
            ":scheme",
            ":path",
        ]
    }

    /// WAF modern mengevaluasi urutan header standar (non-pseudo).
    /// Chrome memiliki urutan leksikal spesifik saat melakukan Fetch/XHR.
    pub fn get_standard_header_order() -> Vec<&'static str> {
        vec![
            "sec-ch-ua",
            "sec-ch-ua-mobile",
            "sec-ch-ua-platform",
            "upgrade-insecure-requests",
            "user-agent",
            "accept",
            "sec-fetch-site",
            "sec-fetch-mode",
            "sec-fetch-user",
            "sec-fetch-dest",
            "accept-encoding",
            "accept-language",
            // Header dinamis seperti 'cookie' atau 'authorization' 
            // biasanya disisipkan di antara accept dan accept-encoding,
            // atau di bagian paling bawah.
        ]
    }
}
