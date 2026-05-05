import ctypes
import json
import os
import smf
from typing import Dict, Optional, Union
from lib.roar.callbin.calling import call_bin


class STLSResponse:
    """Objek sederhana untuk menampung hasil response."""

    def __init__(self, text: str):
        self.text = text

    def json(self):
        return json.loads(self.text)


class STLS:
    def __init__(self):
        # Memanggil binary dari helper eksternal
        lib_path = call_bin("libstls.so")

        # 1. Load Shared Library
        if not lib_path or not os.path.exists(lib_path):
            smf.printd("STLS NOT FOUND", str(lib_path), level="ERROR")
            raise FileNotFoundError(f"STLS library not found or call_bin failed.")

        self._lib = ctypes.CDLL(lib_path)

        # 2. Definisi Signature Fungsi: storm_request
        self._lib.storm_request.argtypes = [
            ctypes.c_char_p,  # url
            ctypes.c_char_p,  # method
            ctypes.c_char_p,  # headers_json
            ctypes.c_char_p,  # body_ptr
            ctypes.c_size_t,  # body_len
        ]
        self._lib.storm_request.restype = ctypes.POINTER(ctypes.c_char)

        # 3. Definisi Signature Fungsi: storm_free_string
        self._lib.storm_free_string.argtypes = [ctypes.POINTER(ctypes.c_char)]
        self._lib.storm_free_string.restype = None

    def request(
        self,
        method: str,
        url: str,
        headers: Optional[Dict[str, str]] = None,
        data: Optional[Union[str, bytes]] = None,
    ) -> STLSResponse:

        # Marshalling Data ke format C
        method_c = method.upper().encode("utf-8")
        url_c = url.encode("utf-8")
        headers_str = json.dumps(headers or {}).encode("utf-8")

        # Handling Body Data
        body_c = None
        body_len = 0
        if data:
            if isinstance(data, str):
                body_c = data.encode("utf-8")
            else:
                body_c = data
            body_len = len(body_c)

        # 4. Eksekusi Request ke Mesin Rust
        result_ptr = self._lib.storm_request(
            url_c, method_c, headers_str, body_c, body_len
        )

        try:
            if not result_ptr:
                smf.printd("STLS returned a null pointer", level="WARN")
                raise Exception("STLS returned a null pointer")

            raw_result = ctypes.string_at(result_ptr).decode("utf-8")

            if raw_result.startswith("ERROR:"):
                smf.printd("Error internal STLS", raw_result, level="ERROR")
                raise Exception(raw_result)

            return STLSResponse(raw_result)

        finally:
            # 5. PENTING: Mencegah Memory Leak
            if result_ptr:
                self._lib.storm_free_string(result_ptr)


# Singleton instance untuk kemudahan penggunaan
_default_stls = None


def request(method: str, url: str, **kwargs):
    global _default_stls
    if _default_stls is None:
        # KOREKSI: Panggil tanpa argumen karena path sudah di-handle call_bin() di __init__
        _default_stls = STLS()
    return _default_stls.request(method, url, **kwargs)


# === Syntactic Sugar Methods ===
def get(url: str, **kwargs):
    return request("GET", url, **kwargs)


def post(url: str, **kwargs):
    return request("POST", url, **kwargs)


def put(url: str, **kwargs):
    return request("PUT", url, **kwargs)


def delete(url: str, **kwargs):
    return request("DELETE", url, **kwargs)


def patch(url: str, **kwargs):
    return request("PATCH", url, **kwargs)


def head(url: str, **kwargs):
    return request("HEAD", url, **kwargs)


def options(url: str, **kwargs):
    return request("OPTIONS", url, **kwargs)
