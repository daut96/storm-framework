import ctypes
import ctypes.util
import json
from typing import Dict, Optional, Union, Any
from lib.roar.callbin.calling import call_bin

# ----------------------------------------------------------------------
# Load shared object (sesuaikan nama file dan path)
# ----------------------------------------------------------------------
_lib_path = call_bin("libstls.so")
_lib = ctypes.CDLL(_lib_path)

# Definisikan signature fungsi C
_lib.storm_request.argtypes = [
    ctypes.c_char_p,  # url
    ctypes.c_char_p,  # method
    ctypes.c_char_p,  # headers_json
    ctypes.POINTER(ctypes.c_ubyte),  # body_ptr
    ctypes.c_size_t,  # body_len
]
_lib.storm_request.restype = ctypes.c_char_p

_lib.storm_free_string.argtypes = [ctypes.c_char_p]
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
    Melakukan request HTTP/2 melalui BoringSSL.
    Mengembalikan dictionary response.
    Jika response sukses -> {'success': True, 'data': str}
    Jika gagal -> {'success': False, 'error': str}
    """
    if headers is None:
        headers = {}

    # Encode ke bytes
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

    # Panggil fungsi Rust
    result_ptr = _lib.storm_request(
        url_bytes,
        method_bytes,
        headers_json_bytes,
        body_ptr,
        body_len,
    )

    if not result_ptr:
        raise RuntimeError("storm_request returned NULL pointer")

    # Ambil string hasil
    result_str = ctypes.string_at(result_ptr).decode("utf-8")
    # Bebaskan memori dari Rust
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
