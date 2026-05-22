# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import os
import sqlite3
from pathlib import Path
from typing import Optional, Tuple

import smf
from rootmap import ROOT

_BASE_DIR = Path(ROOT).resolve()
_DB_PATH = _BASE_DIR / "lib" / "sqlite" / "cached" / "cache.db"

# Ekstensi Shared Object/Binary yang diizinkan untuk multi-OS
_VALID_EXTENSIONS = {".so", ".dll", ".dylib", ".exe", ".bin"}

def _get_db_connection() -> sqlite3.Connection:
    """Manage connections and initialize SQLite database schemas."""
    if not _DB_PATH.parent.exists():
        _DB_PATH.parent.mkdir(parents=True, exist_ok=True)
        smf.printd(f"The cache.db directory is created on {_DB_PATH.parent}", level="INFO")

    conn = sqlite3.connect(_DB_PATH)

    try:
        # Menggunakan 'filename' (termasuk ekstensi) sebagai PK untuk mencegah kolisi 
        # antara 'app.exe' dan 'app.dll'
        conn.execute("""
            CREATE TABLE IF NOT EXISTS binary_cache (
                filename TEXT PRIMARY KEY,
                stem TEXT NOT NULL,
                path TEXT NOT NULL,
                category TEXT,
                last_mtime REAL
            )
        """)
        conn.execute("CREATE INDEX IF NOT EXISTS idx_bin_filename ON binary_cache(filename)")
        conn.execute("CREATE INDEX IF NOT EXISTS idx_bin_stem ON binary_cache(stem)")
    except sqlite3.Error as e:
        smf.printd("Failed to initialize SQLite cache schema", e, level="CRITICAL")
        raise

    return conn

def sync_bin() -> None:
    """
    Scans the filesystem and synchronizes the binary paths to cache.db.
    Supports Shared Objects (.so, .dll, .dylib) without enforcing X_OK.
    """
    smf.printd("Starting binary cache synchronization scan...", level="INFO")
    search_targets = {
        "module": _BASE_DIR / "external" / "source" / "out" / "module",
        "core": _BASE_DIR / "external" / "source" / "out" / "core",
    }

    found_on_disk = {}

    for cat, folder in search_targets.items():
        if not folder.exists():
            smf.printd(f"Target directory not found and skipped: {folder}", level="WARN")
            continue

        for entry in folder.rglob("*"):
            if not entry.is_file():
                continue

            # Logika Filter:
            # 1. Jika memiliki ekstensi library/binary yang valid (mengabaikan X_OK).
            # 2. Jika tidak berekstensi (Linux/macOS binary), maka WAJIB memiliki X_OK.
            suffix = entry.suffix.lower()
            is_valid_so = suffix in _VALID_EXTENSIONS
            is_linux_executable = (suffix == "") and os.access(entry, os.X_OK)

            if is_valid_so or is_linux_executable:
                found_on_disk[entry.name] = {
                    "stem": entry.stem, # Nama tanpa ekstensi (e.g., 'libcrypto' dari 'libcrypto.so')
                    "path": str(entry.absolute()),
                    "category": cat,
                    "mtime": entry.stat().st_mtime,
                }
                smf.printd(f"Binary extracted: {entry.name} ({cat})", level="DEBUG")

    try:
        with _get_db_connection() as conn:
            cursor = conn.cursor()

            # Clean stale data
            cursor.execute("SELECT filename FROM binary_cache")
            cached_names = {row[0] for row in cursor.fetchall()}

            removed = [(name,) for name in cached_names if name not in found_on_disk]
            if removed:
                cursor.executemany("DELETE FROM binary_cache WHERE filename = ?", removed)
                smf.printd(f"Cleaned {len(removed)} binaries no longer in cache.", level="INFO")

            # UPSERT Logic
            updated_count = 0
            for filename, data in found_on_disk.items():
                cursor.execute("""
                    INSERT INTO binary_cache (filename, stem, path, category, last_mtime)
                    VALUES (?, ?, ?, ?, ?)
                    ON CONFLICT(filename) DO UPDATE SET
                        stem = excluded.stem,
                        path = excluded.path,
                        category = excluded.category,
                        last_mtime = excluded.last_mtime
                    WHERE last_mtime != excluded.last_mtime
                """, (filename, data["stem"], data["path"], data["category"], data["mtime"]))

                if cursor.rowcount > 0:
                    updated_count += 1

            conn.commit()
            smf.printd(
                f"DB sync complete. Total: {len(found_on_disk)}, Updated/Inserted: {updated_count}", 
                level="INFO"
            )

    except sqlite3.Error as e:
        smf.printd("I/O error during DB sync transaction.", e, level="ERROR")
        raise

def _query_db(query_column: str, query_value: str) -> Optional[str]:
    """Helper method for exact lookups in the database."""
    try:
        with _get_db_connection() as conn:
            cursor = conn.cursor()
            cursor.execute(f"SELECT path FROM binary_cache WHERE {query_column} = ?", (query_value,))
            row = cursor.fetchone()
            return row[0] if row else None
    except sqlite3.Error as e:
        smf.printd(f"DB error looking up {query_value}", e, level="CRITICAL")
        raise
          
