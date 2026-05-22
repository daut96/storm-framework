# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import sys
import ctypes
import smf

from typing import Optional
from .manager import _query_db


# Target Environment Detection
IS_WINDOWS = sys.platform.startswith("win")
IS_MACOS = sys.platform == "darwin"
# Android detected as 'linux' in Python
IS_LINUX_OR_ANDROID = sys.platform.startswith("linux") or hasattr(sys, 'getandroidapilevel')

def resolve_bin_path(query_name: str) -> str:
    """
    Mencari absolute path dari binary/shared object.
    Mendukung 'exact match' (dengan ekstensi) dan 'smart fallback' (tanpa ekstensi).
    """
    smf.printd(f"Resolving path for: {query_name}", level="DEBUG")

    # 1. Exact Match: Cek apakah nama spesifik dengan ekstensi dikirim caller
    exact_path = _query_db("filename", query_name)
    if exact_path:
        smf.printd(f"Exact match found: {exact_path}", level="DEBUG")
        return exact_path

    # 2. OS-Aware Smart Fallback: Jika caller memberikan 'nama binary saja'
    # Prioritaskan format berdasarkan OS yang sedang berjalan
    candidates = []
    if IS_WINDOWS:
        candidates = [f"{query_name}.dll", f"{query_name}.exe"]
    elif IS_MACOS:
        candidates = [f"{query_name}.dylib", f"lib{query_name}.dylib", f"{query_name}.so"]
    elif IS_LINUX_OR_ANDROID:
        candidates = [f"{query_name}.so", f"lib{query_name}.so", query_name]
    
    for candidate in candidates:
        candidate_path = _query_db("filename", candidate)
        if candidate_path:
            smf.printd(f"OS-specific resolution successful: {candidate} -> {candidate_path}", level="DEBUG")
            return candidate_path

    # Exception spesifik untuk mempermudah error handling di level caller
    smf.printd(f"Resolution failed for {query_name}. Candidates tried: {candidates}", level="ERROR")
    raise FileNotFoundError(f"Binary or Shared Object '{query_name}' not found for current OS.")

def call_bin(query_name: str) -> str:
    """Wrapper untuk backward compatibility"""
    return resolve_bin_path(query_name)

def load_shared_object(query_name: str) -> ctypes.CDLL:
    """
    Mencari jalur file dan langsung me-load nya menggunakan ctypes.
    Sangat berguna untuk integrasi C/C++ FFI secara native.
    """
    lib_path = resolve_bin_path(query_name)
    
    try:
        smf.printd(f"Loading shared object via ctypes: {lib_path}", level="INFO")
        # Menggunakan mode RTLD_LOCAL/GLOBAL dapat diatur di sini jika diperlukan pada POSIX
        return ctypes.CDLL(lib_path)
    except OSError as e:
        smf.printd(f"Failed to load shared library '{lib_path}' into memory.", e, level="CRITICAL")
        raise
        
