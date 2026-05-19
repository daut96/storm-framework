import os
import ast
import smf

from .colors import *
from rootmap import ROOT

def parse_query(query):
    """
    Memecah query menjadi base_query dan dictionary filters.
    Contoh: 'dos action:sip defaction:dos' 
    -> base_query = 'dos'
    -> filters = {'action': 'sip', 'defaction': 'dos'}
    """
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
    """
    Mengekstrak dictionary 'metadata' menggunakan AST tanpa mengeksekusi module.
    Pendekatan ini menjamin keamanan (Zero Code Execution) dan performa I/O yang cepat.
    """
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
        smf.printd("MODULE DOES NOT HAVE VALID METADATA", e, level="ERROR")
        pass
    
    return {}

def search_modules(query_str):
    modules_path = os.path.join(ROOT, "modules")
    base_query, filters = parse_query(query_str)

    smf.printf(f"\n{CC.YELLOW}[*] Searching for =>{CC.RESET}", query_str)
    smf.printf()
    
    # Menyesuaikan rasio kolom agar deskripsi bisa masuk tanpa merusak format tabel
    smf.printf(f"{CC.CYAN}{'Module Path':<30} {'Category':<12} {'Description'}{CC.RESET}")
    smf.printf(f"{CC.MAGENTA}{'-'*30} {'-'*12} {'-'*40}{CC.RESET}")
    
    count = 0

    if not os.path.exists(modules_path):
        smf.printf(f"[!] Directory not found => {modules_path}")
        return
        
    for root, dirs, files in os.walk(modules_path):
        for file in files:
            if file.endswith(".py") and file != "__init__.py":
                file_name_only = file.replace(".py", "").lower()
                rel_path = os.path.relpath(os.path.join(root, file), modules_path)
                clean_path = rel_path.replace(".py", "")
                category = rel_path.split(os.sep)[0]

                # 1. Logika matching base query (nama file harus match jika base query ada)
                if base_query and base_query not in file_name_only:
                    continue

                # 2. Load Metadata
                file_path = os.path.join(root, file)
                metadata = extract_metadata(file_path)

                # 3. Logika multi-filter (O(N) iteration matching)
                is_match = True
                for f_key, f_val in filters.items():
                    if f_key == "author":
                        authors = [a.lower() for a in metadata.get("Author", [])]
                        if not any(f_val in a for a in authors):
                            is_match = False
                            break
                            
                    elif f_key == "action":
                        # Mengambil element ke-0 karena struktur Action adalah List of Lists ["Name", {dict}]
                        actions = [a[0].lower() for a in metadata.get("Action", []) if isinstance(a, list) and len(a) > 0]
                        if not any(f_val in act for act in actions):
                            is_match = False
                            break
                            
                    elif f_key == "defaction":
                        def_action = metadata.get("DefaultAction", "").lower()
                        if f_val not in def_action:
                            is_match = False
                            break
                    else:
                        # Jika user memasukkan filter yang tidak didukung, abaikan atau jadikan false
                        is_match = False
                        break

                if not is_match:
                    continue

                # Data Normalization untuk Display (Mencegah multiline merusak tabel)
                raw_desc = metadata.get("Description", "No description provided.")
                # Hapus newline dan whitespace berlebih
                clean_desc = " ".join(raw_desc.split()) 
                
                # Truncation mekanis jika string terlalu panjang (opsional: atur 45 sesuai lebar terminal)
                if len(clean_desc) > 45:
                    clean_desc = clean_desc[:42] + "..."

                count += 1
                smf.printf(f"{CC.YELLOW}{clean_path:<30} {category:<12} {clean_desc}{CC.RESET}")

    if count == 0:
        smf.printf(f"\n{CC.YELLOW}[*] {query_str} => Not found.{CC.RESET}")
    else:
        smf.printf(f"\n{CC.YELLOW}[✓] Found {count} module(s).{CC.RESET}")
    
    smf.printf()
    
