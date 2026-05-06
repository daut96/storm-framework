mod tls;
mod http2;

// Injeksi Hulu Google murni (BoringSSL via Bindgen)
#[allow(non_upper_case_globals, non_camel_case_types, non_snake_case, dead_code)]
pub mod bssl {
    include!(concat!(env!("OUT_DIR"), "/bssl_bindings.rs"));
}

use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::OnceLock;
use tokio::runtime::Runtime;
use tokio::net::TcpStream;

use tls::builder::build_chrome_ssl_context;
use http2::fingerprint::ChromeH2Settings;

// =========================================================================
// GLOBAL OPTIMIZATION (Zero-Cost Runtime)
// =========================================================================

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

#[unsafe(no_mangle)]
pub extern "C" fn storm_request(
    url_ptr: *const c_char,
    method_ptr: *const c_char,
    headers_json_ptr: *const c_char,
    body_ptr: *const u8,
    body_len: usize,
) -> *mut c_char {
    if url_ptr.is_null() || method_ptr.is_null() {
        return CString::new("ERROR: Null pointer on mandatory fields").unwrap().into_raw();
    }

    let url = unsafe { CStr::from_ptr(url_ptr).to_string_lossy().into_owned() };
    let method = unsafe { CStr::from_ptr(method_ptr).to_string_lossy().into_owned() };
    
    let headers_json = if headers_json_ptr.is_null() {
        "{}".to_string() 
    } else {
        unsafe { CStr::from_ptr(headers_json_ptr).to_string_lossy().into_owned() }
    };

    let body_bytes = if body_ptr.is_null() || body_len == 0 {
        None
    } else {
        unsafe { Some(std::slice::from_raw_parts(body_ptr, body_len).to_vec()) }
    };

    let rt = get_runtime();

    let result = rt.block_on(async move {
        match execute_dynamic_request(&url, &method, &headers_json, body_bytes).await {
            Ok(response_body) => {
                serde_json::json!({
                    "success": true,
                    "data": response_body
                }).to_string()
            },
            Err(e) => {
                serde_json::json!({
                    "success": false,
                    "error": e.to_string()
                }).to_string()
            }
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
    let uri = target_url.parse::<http::Uri>()?;
    let host = uri.host().ok_or("URL has no host")?;
    let port = uri.port_u16().unwrap_or(443);
    let target_addr = format!("{}:{}", host, port);

    // 1. KONEKSI TCP & TLS (Chrome 145+ Bare-Metal)
    let tcp_stream = TcpStream::connect(&target_addr).await?;
    let native_ctx = build_chrome_ssl_context()?;
    let tls_stream = tls::stream::StormTlsStream::connect(native_ctx.inner, tcp_stream, host).await?;

    // 2. HANDSHAKE HTTP/2 (Menggunakan Sidik Jari SETTINGS & WINDOW_UPDATE Chrome)
    let mut h2_builder = h2::client::Builder::new();
    h2_builder
        .initial_window_size(ChromeH2Settings::INITIAL_WINDOW_SIZE)
        .initial_connection_window_size(ChromeH2Settings::CONNECTION_WINDOW_UPDATE) 
        .max_concurrent_streams(ChromeH2Settings::MAX_CONCURRENT_STREAMS)
        .header_table_size(ChromeH2Settings::HEADER_TABLE_SIZE)
        .max_header_list_size(ChromeH2Settings::MAX_HEADER_LIST_SIZE)
        .enable_push(ChromeH2Settings::ENABLE_PUSH);

    let (mut client, h2_connection) = h2_builder.handshake::<_, bytes::Bytes>(tls_stream).await?;

    tokio::spawn(async move {
        if let Err(_e) = h2_connection.await { /* Connection dropped */ }
    });

    // 3. SINKRONISASI HEADER: DYNAMIC INJECT & MERGE
    let mut request_builder = http::Request::builder().method(method).uri(target_url);

    // A. Parse Custom Headers dari User (Python/Go)
    let mut user_headers: HashMap<String, String> = HashMap::new();
    if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(headers_json) {
        if let Some(obj) = parsed.as_object() {
            for (k, v) in obj {
                if let Some(v_str) = v.as_str() {
                    user_headers.insert(k.to_lowercase(), v_str.to_string());
                }
            }
        }
    }

    // B. Generate Base Chrome 145 Headers (OS-Aware: Android/Linux)
    let base_headers = ChromeH2Settings::generate_dynamic_headers();

    // C. Merge Logic: User Overrides Base, Base Fills Gaps
    for (key, default_val) in base_headers {
        if let Some(custom_val) = user_headers.remove(&key) {
            request_builder = request_builder.header(&key, custom_val);
        } else {
            request_builder = request_builder.header(&key, default_val);
        }
    }

    // D. Suntikkan sisa header unik (Cookie, Auth, dll)
    for (k, v) in user_headers {
        request_builder = request_builder.header(k, v);
    }

    let request = request_builder.body(())?;

    // 4. EXECUTION
    let has_body = body.is_some();
    let (response_future, mut send_stream) = client.send_request(request, !has_body)?;

    if let Some(payload) = body {
        send_stream.send_data(bytes::Bytes::from(payload), true)?;
    }

    let response = response_future.await?;
    let mut resp_body = response.into_body();
    let mut response_bytes = Vec::new();

    while let Some(chunk) = resp_body.data().await {
        let chunk = chunk?;
        response_bytes.extend_from_slice(&chunk);
    }

    Ok(String::from_utf8_lossy(&response_bytes).to_string())
}
