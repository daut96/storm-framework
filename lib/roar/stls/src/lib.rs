mod tls;
mod http2;

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_void};
use std::sync::OnceLock;
use tokio::runtime::Runtime;
use tokio::net::TcpStream;
use boring::ssl::Ssl;

use tls::builder::build_chrome_ssl_context;
use http2::fingerprint::ChromeH2Settings;

// =========================================================================
// GLOBAL OPTIMIZATION (Zero-Cost Runtime)
// =========================================================================

/// Menggunakan OnceLock untuk memastikan Tokio Runtime hanya di-spawn SEKALI 
/// selama siklus hidup shared library (menghilangkan overhead thread-creation).
static TOKIO_RT: OnceLock<Runtime> = OnceLock::new();

fn get_runtime() -> &'static Runtime {
    TOKIO_RT.get_or_init(|| {
        tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .expect("Fatal: Failed to initialize Tokio multi-thread runtime")
    })
}

// =========================================================================
// C-ABI / FFI BOUNDARY
// =========================================================================

/// FFI Signature yang lebih kaya.
/// - method_ptr: "GET", "POST", "PUT", dll.
/// - headers_json_ptr: Serialisasi JSON dari dictionary header (mudah dipassing dari Python/Go).
/// - body_ptr & body_len: Array of bytes untuk payload (mendukung binary maupun teks).
#[unsafe(no_mangle)]
pub extern "C" fn storm_request(
    url_ptr: *const c_char,
    method_ptr: *const c_char,
    headers_json_ptr: *const c_char,
    body_ptr: *const u8,
    body_len: usize,
) -> *mut c_char {
    // 1. Validasi Input Dasar
    if url_ptr.is_null() || method_ptr.is_null() {
        return CString::new("ERROR: Null pointer on mandatory fields").unwrap().into_raw();
    }

    // 2. Marshalling C-Types ke Rust-Types
    let url = unsafe { CStr::from_ptr(url_ptr).to_string_lossy().into_owned() };
    let method = unsafe { CStr::from_ptr(method_ptr).to_string_lossy().into_owned() };
    
    let headers_json = if headers_json_ptr.is_null() {
        "{}\"".to_string() // Default empty JSON object
    } else {
        unsafe { CStr::from_ptr(headers_json_ptr).to_string_lossy().into_owned() }
    };

    let body_bytes = if body_ptr.is_null() || body_len == 0 {
        None
    } else {
        unsafe { Some(std::slice::from_raw_parts(body_ptr, body_len).to_vec()) }
    };

    // 3. Panggil Global Tokio Runtime
    let rt = get_runtime();

    // 4. Eksekusi Engine Asinkronus
    let result = rt.block_on(async move {
        match execute_dynamic_request(&url, &method, &headers_json, body_bytes).await {
            Ok(response_body) => response_body,
            Err(e) => format!("ERROR: STLS Failure - {}", e),
        }
    });

    CString::new(result).unwrap_or_else(|_| CString::new("ERROR: Encoding failed").unwrap()).into_raw()
}

#[unsafe(no_mangle)]
pub extern "C" fn storm_free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe { let _ = CString::from_raw(s); }
    }
}

// =========================================================================
// ASYNCHRONOUS EVASION ENGINE
// =========================================================================

async fn execute_dynamic_request(
    target_url: &str,
    method: &str,
    headers_json: &str,
    body: Option<Vec<u8>>,
) -> Result<String, Box<dyn std::error::Error>> {
    // 1. Parsing Target
    let uri = target_url.parse::<http::Uri>()?;
    let host = uri.host().ok_or("URL has no host")?;
    let port = uri.port_u16().unwrap_or(443);
    let target_addr = format!("{}:{}", host, port);

    // 2. Setup TLS & Koneksi
    let ctx = build_chrome_ssl_context()?;
    let tcp_stream = TcpStream::connect(&target_addr).await?;
    
    let mut ssl = Ssl::new(&ctx)?;
    ssl.set_hostname(host)?; 

    let tls_stream = tokio_boring::SslStreamBuilder::new(ssl, tcp_stream).connect().await?;

    // 3. Negosiasi HTTP/2
    let mut h2_builder = h2::client::Builder::new();
    h2_builder
        .initial_window_size(ChromeH2Settings::INITIAL_WINDOW_SIZE)
        .max_concurrent_streams(ChromeH2Settings::MAX_CONCURRENT_STREAMS)
        .header_table_size(ChromeH2Settings::HEADER_TABLE_SIZE)
        .enable_push(false);

    let (mut client, h2_connection) = h2_builder.handshake::<_, bytes::Bytes>(tls_stream).await?;

    tokio::spawn(async move {
        if let Err(_e) = h2_connection.await {
            // Background connection handler (bisa pasang logging di sini)
        }
    });

    // 4. Membangun HTTP Request secara Dinamis
    let mut request_builder = http::Request::builder()
        .method(method)
        .uri(target_url);

    // Parsing Header dari JSON (memudahkan interface multi-bahasa)
    if let Ok(parsed_headers) = serde_json::from_str::<serde_json::Value>(headers_json) {
        if let Some(obj) = parsed_headers.as_object() {
            for (key, value) in obj {
                if let Some(val_str) = value.as_str() {
                    request_builder = request_builder.header(key, val_str);
                }
            }
        }
    }

    // Default Pseudo-Headers H2 (jika tidak di-supply oleh user)
    // Di produksi, biasanya kita men-cek apakah key "user-agent" sudah ada sebelum menambahkan default.
    let request = request_builder.body(())?;

    // 5. Logika Pengiriman dengan/tanpa Body (End-Of-Stream logic)
    let has_body = body.is_some();
    // Jika has_body = true, end_of_stream saat mengirim head harus false!
    let (response_future, mut send_stream) = client.send_request(request, !has_body)?;

    // 6. Streaming Body Payload (Jika ada method POST/PUT)
    if let Some(payload) = body {
        // Mengirim data chunk, parameter 'true' di akhir berarti ini chunk terakhir (EOS)
        send_stream.send_data(bytes::Bytes::from(payload), true)?;
    }

    // 7. Tunggu & Kumpulkan Response
    let response = response_future.await?;
    let mut resp_body = response.into_body();
    let mut response_bytes = Vec::new();

    while let Some(chunk) = resp_body.data().await {
        let chunk = chunk?;
        response_bytes.extend_from_slice(&chunk);
    }

    Ok(String::from_utf8_lossy(&response_bytes).to_string())
                                                                 }
