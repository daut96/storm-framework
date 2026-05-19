import os
import ast
import sqlite3
import smf

from .colors import *
from rootmap import ROOT


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


def extract_metadata(file_path):
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            source_code = f.read()
            tree = ast.parse(source_code, filename=file_path)
            for node in ast.walk(tree):
                if isinstance(node, ast.Assign):
                    for target in node.targets:
                        if isinstance(target, ast.Name) and target.id == "metadata":
                            return ast.literal_eval(node.value)
    except Exception:
        pass
    return {}


def search_modules(query_str):
    # Sesuaikan path database Anda
    db_path = os.path.join(ROOT, "sqlite", "cached", "cache.db")
    base_query, filters = parse_query(query_str)

    smf.printf(f"\n{CC.YELLOW}[*] Searching for =>{CC.RESET} {query_str}")
    smf.printf()
    smf.printf(
        f"{CC.CYAN}{'Module Path':<30} {'Category':<12} {'Description'}{CC.RESET}"
    )
    smf.printf(f"{CC.MAGENTA}{'-'*30} {'-'*12} {'-'*40}{CC.RESET}")

    count = 0
    supported_filters = {"author", "action", "defaction"}

    # 1. Fast fail untuk filter yang tidak valid
    if filters and not all(f_key in supported_filters for f_key in filters.keys()):
        smf.printf(f"\n{CC.YELLOW}[!] Invalid filter keys detected.{CC.RESET}\n")
        return

    # 2. Query ke SQLite (Menggantikan os.walk)
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()

        if base_query:
            # Gunakan LIKE untuk partial match pada module_name
            cursor.execute(
                "SELECT path, category, module_name FROM module_cache WHERE module_name LIKE ?",
                (f"%{base_query}%",),
            )
        else:
            cursor.execute("SELECT path, category, module_name FROM module_cache")

        rows = cursor.fetchall()
        conn.close()
    except sqlite3.Error as e:
        smf.printf(f"[!] Database Error: {e}")
        return

    # 3. Filter Evaluation (hanya untuk file yang lolos query DB)
    for row in rows:
        file_path, category, module_name = row

        # Jika ada filter, kita harus ekstrak metadata
        metadata = {}
        if filters or True:  # Kita tetap butuh metadata untuk menampilkan Description
            metadata = extract_metadata(file_path)

        if filters and not metadata:
            continue

        is_valid = True
        for f_key, f_val in filters.items():
            if f_key == "author":
                authors = [str(a).lower() for a in metadata.get("Author", [])]
                if not any(f_val in a for a in authors):
                    is_valid = False
                    break
            elif f_key == "action":
                actions = [
                    str(a[0]).lower()
                    for a in metadata.get("Action", [])
                    if isinstance(a, list) and len(a) > 0
                ]
                if not any(f_val in act for act in actions):
                    is_valid = False
                    break
            elif f_key == "defaction":
                def_action = str(metadata.get("DefaultAction", "")).lower()
                if f_val not in def_action:
                    is_valid = False
                    break

        if not is_valid:
            continue

        # Format output
        raw_desc = metadata.get("Description", "No description provided.")
        clean_desc = " ".join(raw_desc.split())
        if len(clean_desc) > 45:
            clean_desc = clean_desc[:42] + "..."

        count += 1
        # Menggunakan module_name dari DB (lebih bersih)
        smf.printf(
            f"{CC.YELLOW}{module_name:<30} {category:<12} {clean_desc}{CC.RESET}"
        )

    if count == 0:
        smf.printf(f"\n{CC.YELLOW}[*] {query_str} => Not found.{CC.RESET}")
    else:
        smf.printf(f"\n{CC.YELLOW}[✓] Found {count} module(s).{CC.RESET}")

    smf.printf()
