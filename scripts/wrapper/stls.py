import ctypes
import ctypes.util
import json
from typing import Dict, Optional, Union, Any
from lib.roar.callbin.calling import call_bin

# ----------------------------------------------------------------------
# Load shared object
# ----------------------------------------------------------------------
_lib_path = call_bin("libstls.so")
_lib = ctypes.CDLL(_lib_path)

# 1. PERBAIKAN FATAL: Gunakan c_void_p untuk pointer yang dialokasikan secara manual
_lib.storm_request.argtypes = [
    ctypes.c_char_p,  # url
    ctypes.c_char_p,  # method
    ctypes.c_char_p,  # headers_json
    ctypes.POINTER(ctypes.c_ubyte),  # body_ptr
    ctypes.c_size_t,  # body_len
]
# Jangan gunakan c_char_p di restype!
_lib.storm_request.restype = ctypes.c_void_p

# Menerima c_void_p murni
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
    
    if headers is None:
        headers = {}

    url_bytes = url.encode("utf-8")
    method_bytes = method.upper().encode("utf-8")
    headers_json_bytes = json.dumps(headers).encode("utf-8")

    body_ptr = ctypes.POINTER(ctypes.c_ubyte)()
    body_len = 0
    body_bytes = None

    if body is not None:
        if isinstance(body, str):
            body_bytes = body.encode("utf-8")
        else:
            body_bytes = body
        body_len = len(body_bytes)
        body_ptr = (ctypes.c_ubyte * body_len).from_buffer_copy(body_bytes)

    # 2. Panggil fungsi Rust (result_ptr sekarang adalah alamat memori 64-bit murni/int)
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
        # 3. Cast pointer murni menjadi C-String, lalu baca valuenya menjadi bytes
        result_bytes = ctypes.cast(result_ptr, ctypes.c_char_p).value
        # Decode ke Python string
        result_str = result_bytes.decode("utf-8")
    finally:
        # 4. SANGAT PENTING: Kembalikan pointer asli ke Rust untuk di-free!
        # Blok finally memastikan memori tetap di-free meskipun terjadi error saat decode
        _lib.storm_free_string(result_ptr)

    # Parse JSON response
    try:
        response = json.loads(result_str)
    except json.JSONDecodeError as e:
        raise RuntimeError(f"Invalid JSON response from Rust: {result_str}") from e

    return response

# ----------------------------------------------------------------------
# Public API: GET, POST, PUT, DELETE, dll.
# ----------------------------------------------------------------------
def get(url: str, headers: Optional[Dict[str, str]] = None) -> str:
    """
    Melakukan HTTP GET request.
    Mengembalikan response body sebagai string jika success.
    Melempar exception jika gagal.
    """
    resp = _request("GET", url, headers)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def post(
    url: str,
    headers: Optional[Dict[str, str]] = None,
    body: Optional[Union[bytes, str]] = None,
) -> str:
    """HTTP POST request."""
    resp = _request("POST", url, headers, body)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def put(
    url: str,
    headers: Optional[Dict[str, str]] = None,
    body: Optional[Union[bytes, str]] = None,
) -> str:
    """HTTP PUT request."""
    resp = _request("PUT", url, headers, body)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def delete(
    url: str,
    headers: Optional[Dict[str, str]] = None,
    body: Optional[Union[bytes, str]] = None,
) -> str:
    """HTTP DELETE request."""
    resp = _request("DELETE", url, headers, body)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def patch(
    url: str,
    headers: Optional[Dict[str, str]] = None,
    body: Optional[Union[bytes, str]] = None,
) -> str:
    """HTTP PATCH request."""
    resp = _request("PATCH", url, headers, body)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def head(
    url: str,
    headers: Optional[Dict[str, str]] = None,
) -> str:
    """HTTP HEAD request. Response body biasanya kosong."""
    resp = _request("HEAD", url, headers)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))


def options(
    url: str,
    headers: Optional[Dict[str, str]] = None,
) -> str:
    """HTTP OPTIONS request."""
    resp = _request("OPTIONS", url, headers)
    if resp.get("success"):
        return resp["data"]
    else:
        raise RuntimeError(resp.get("error", "Unknown error"))
