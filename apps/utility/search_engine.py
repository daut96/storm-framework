import os
import sqlite3
import smf

from .colors import *
from rootmap import ROOT
from typing import List

DB_PATH = os.path.join(ROOT, "lib", "sqlite", "cached", "cache.db")


def _get_db_connection():
    # Helper for Read-Only DB connections
    return sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True, check_same_thread=False)


def parse_query(query):
    parts = query.split()
    base_queries = []
    filters = {}
    for part in parts:
        if ":" in part:
            key, val = part.split(":", 1)
            filters[key.lower()] = val.lower()
        else:
            base_queries.append(part.lower())
    return " ".join(base_queries), filters


# ---------------------------------------------------------
# SHOW FUNCTION
# ---------------------------------------------------------
def show_modules(category: str) -> List[str]:
    """
    Mengembalikan daftar module_name berdasarkan kategori.
    Sangat cepat karena menggunakan SQL Index 'idx_category'.
    """
    smf.printd(f"Executing category fetch for", category, level="DEBUG")
    try:
        conn = _get_db_connection()
        cursor = conn.cursor()
        cursor.execute(
            "SELECT module_name FROM module_cache WHERE category = ?", (category,)
        )
        results = [row[0] for row in cursor.fetchall()]
        conn.close()
        return results
    except Exception as e:
        smf.printd(
            f"Failed to execute lookup for category: {category}", e, level="ERROR"
        )
        return []


# ---------------------------------------------------------
# SEARCH FUNCTION WITH DYNAMIC SQL
# ---------------------------------------------------------
def search_modules(query):
    base_query, filters = parse_query(query)

    smf.printf(f"\n{CC.YELLOW}[*] Searching for =>{CC.RESET} {query}")
    smf.printf()
    smf.printf(
        f"{CC.CYAN}{'Module Path':<35} {'Category':<15} {'Description'}{CC.RESET}"
    )
    smf.printf(f"{CC.MAGENTA}{'-'*35} {'-'*15} {'-'*45}{CC.RESET}")

    supported_filters = {"author", "action", "defaction"}

    if filters and not all(f_key in supported_filters for f_key in filters.keys()):
        smf.printf(
            f"{CC.YELLOW}[!] Invalid filter detected. Supported: author, action, defaction{CC.RESET}\n"
        )
        return

    # Membangun Raw SQL Query secara dinamis berdasarkan input user
    query_parts = []
    sql_params = []

    if base_query:
        query_parts.append("module_name LIKE ?")
        sql_params.append(f"%{base_query}%")

    for f_key, f_val in filters.items():
        if f_key == "author":
            query_parts.append("LOWER(author) LIKE ?")
            sql_params.append(f"%{f_val}%")
        elif f_key == "action":
            query_parts.append("LOWER(actions) LIKE ?")
            sql_params.append(f"%{f_val}%")
        elif f_key == "defaction":
            query_parts.append("LOWER(default_action) LIKE ?")
            sql_params.append(f"%{f_val}%")

    # Finalisasi SQL String
    sql_query = "SELECT module_name, category, description FROM module_cache"
    if query_parts:
        # Menggunakan AND untuk strict evaluation (semua filter harus match)
        sql_query += " WHERE " + " AND ".join(query_parts)

    try:
        conn = _get_db_connection()
        cursor = conn.cursor()
        cursor.execute(sql_query, sql_params)
        rows = cursor.fetchall()
        conn.close()
    except sqlite3.Error as e:
        smf.printf("{CC.RED}[!] Database Read Error{CC.RESET}")
        smf.printd("Database Read Error", e, level="ERROR")
        return

    # Render Tabel
    count = 0
    for row in rows:
        module_name, category, description = row

        # Fallback jika deskripsi kosong dari database
        desc = description if description else "No description provided."
        if len(desc) > 50:
            desc = desc[:47] + "..."

        count += 1
        smf.printf(f"{CC.YELLOW}{module_name:<35}{CC.RESET} {category:<15} {desc}")

    if count == 0:
        smf.printf(f"\n{CC.YELLOW}[!] {query} => Not found{CC.RESET}\n")
    else:
        smf.printf(f"\n{CC.GREEN}[✓] Found {count} modules{CC.RESET}\n")
