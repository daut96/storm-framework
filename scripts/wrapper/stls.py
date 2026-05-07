import ctypes
import json
import base64
import zlib
from typing import Dict, Optional, Union, Any
from lib.roar.callbin.calling import call_bin

# Import opsional untuk kompresi modern (standar Chrome)
try:
    import brotli
except ImportError:
    brotli = None

try:
    import zstandard as zstd
except ImportError:
    zstd = None

# ----------------------------------------------------------------------
# Load shared object
# ----------------------------------------------------------------------
_lib_path = call_bin("libstls.so")
_lib = ctypes.CDLL(_lib_path)

# Mencegah Truncation MTE dengan c_void_p
_lib.storm_request.argtypes = [
    ctypes.c_char_p,  
    ctypes.c_char_p,  
    ctypes.c_char_p,  
    ctypes.POINTER(ctypes.c_ubyte),  
    ctypes.c_size_t,  
]
_lib.storm_request.restype = ctypes.c_void_p

_lib.storm_free_string.argtypes = [ctypes.c_void_p]
_lib.storm_free_string.restype = None

# ----------------------------------------------------------------------
# Helper untuk memanggil fungsi Rust
# ----------------------------------------------------------------------
def _request(
    method: str,
    url: str,
    headers: Optional[Dict[str, str]] = None,
    body: Optional[Union[bytes, str]] = None,
) -> Dict[str, Any]:
    """
    Melakukan request HTTP/2 melalui Evasion Engine BoringSSL.
    Mengembalikan dict: {'status': int, 'headers': dict, 'body': str}
    """
    if headers is None:
        headers = {}

    url_bytes = url.encode("utf-8")
    method_bytes = method.upper().encode("utf-8")
    headers_json_bytes = json.dumps(headers).encode("utf-8")

    body_ptr = ctypes.POINTER(ctypes.c_ubyte)()
    body_len = 0
    
    if body is not None:
        body_bytes = body.encode("utf-8") if isinstance(body, str) else body
        body_len = len(body_bytes)
        body_ptr = (ctypes.c_ubyte * body_len).from_buffer_copy(body_bytes)

    # Eksekusi FFI
    result_ptr = _lib.storm_request(
        url_bytes,
        method_bytes,
        headers_json_bytes,
        body_ptr,
        body_len,
    )

    if not result_ptr:
        raise RuntimeError("storm_request returned NULL pointer")

    try:
        # Cast ke string dan decode
        result_bytes = ctypes.cast(result_ptr, ctypes.c_char_p).value
        result_str = result_bytes.decode("utf-8")
    finally:
        # GARANSI BEBAS LEAK: Selalu kembalikan pointer ke Rust
        _lib.storm_free_string(result_ptr)

    # ------------------------------------------------------------------
    # Parsing & Dekompresi Logika
    # ------------------------------------------------------------------
    try:
        response_wrapper = json.loads(result_str)
    except json.JSONDecodeError as e:
        raise RuntimeError(f"Invalid JSON response from Rust: {result_str}") from e

    if not response_wrapper.get("success"):
        raise RuntimeError(response_wrapper.get("error", "Unknown Rust error"))

    # Karena Rust membungkus data dalam string JSON (via to_string()), kita parse lagi
    inner_data = response_wrapper.get("data", "{}")
    if isinstance(inner_data, str):
        try:
            inner_data = json.loads(inner_data)
        except json.JSONDecodeError:
            # Fallback jika Rust hanya mengembalikan body raw
            return {"status": 200, "headers": {}, "body": inner_data}

    status_code = inner_data.get("status", 200)
    resp_headers = inner_data.get("headers", {})
    body_b64 = inner_data.get("body_base64", "")

    # Decode Base64 ke raw bytes
    raw_body = base64.b64decode(body_b64)

    # Dekompresi otomatis berdasarkan Header dari Server Target
    encoding = resp_headers.get("content-encoding", "").lower()
    try:
        if encoding == "br":
            if brotli is None:
                raise RuntimeError("Brotli encoding received but 'brotli' module is missing. (pip install brotli)")
            final_body = brotli.decompress(raw_body)
        elif encoding == "zstd":
            if zstd is None:
                raise RuntimeError("Zstd encoding received but 'zstandard' module is missing. (pip install zstandard)")
            final_body = zstd.ZstdDecompressor().decompress(raw_body)
        elif encoding in ("gzip", "deflate"):
            # MAX_WBITS | 32 otomatis mendeteksi gzip atau zlib header
            final_body = zlib.decompress(raw_body, zlib.MAX_WBITS | 32)
        else:
            final_body = raw_body
    except Exception as e:
        raise RuntimeError(f"Decompression failed for encoding '{encoding}': {e}")

    # Kembalikan sebagai Dictionary komprehensif
    return {
        "status": status_code,
        "headers": resp_headers,
        "body": final_body.decode("utf-8", errors="replace")
    }

# ----------------------------------------------------------------------
# Public API: Mengembalikan Full Context (Status, Headers, Body)
# ----------------------------------------------------------------------
def get(url: str, headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    return _request("GET", url, headers)

def post(url: str, headers: Optional[Dict[str, str]] = None, body: Optional[Union[bytes, str]] = None) -> Dict[str, Any]:
    return _request("POST", url, headers, body)

def put(url: str, headers: Optional[Dict[str, str]] = None, body: Optional[Union[bytes, str]] = None) -> Dict[str, Any]:
    return _request("PUT", url, headers, body)

def delete(url: str, headers: Optional[Dict[str, str]] = None, body: Optional[Union[bytes, str]] = None) -> Dict[str, Any]:
    return _request("DELETE", url, headers, body)

def patch(url: str, headers: Optional[Dict[str, str]] = None, body: Optional[Union[bytes, str]] = None) -> Dict[str, Any]:
    return _request("PATCH", url, headers, body)

def head(url: str, headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    return _request("HEAD", url, headers)

def options(url: str, headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    return _request("OPTIONS", url, headers)
            
