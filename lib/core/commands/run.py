# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import copy
import smf
import apps.utility.utils as utils

from apps.utility.colors import *


# Run command to run a module that we want to execute
# The workflow is as below;
#
# Command => use <module_name>
# Command => set <var> <val>
# Command => run
#
# After running the module will run and you just have to wait for it to finish.
def execute(args: list[str], ctx: "Context") -> None:
    current_module = ctx.current_module
    options = ctx.options
    plugin = ctx.plugin

    if not current_module:
        smf.printf(f"{CC.YELLOW}[!] No modules selected. (use <module>) first.{CC.RESET}")
        return

    required_vars = getattr(current_module, "REQUIRED_OPTIONS", {})
    missing = [key for key in required_vars.keys() if not str(options.get(key, "")).strip()]

    if missing:
        smf.printf(f"{CC.YELLOW}[!] Failed to run. Variabel null: {', '.join(missing)}{CC.RESET}")
        smf.printf()
        return

    metadata = getattr(current_module, "metadata", {})
    is_handled_by_plugin = False

    # ==========================================
    # [PERBAIKAN] Data Preparation & Sanitization
    # ==========================================
    
    # 1. Filtering: Hanya ambil options yang memiliki nilai (tidak string kosong)
    # Ini mencegah pengiriman key irrelevant seperti "SUBDOM": "" ke plugin.
    sanitized_options = {k: v for k, v in options.items() if str(v).strip()}
    
    # 2. Defensive Copying: Buat instance baru yang terpisah di memori
    # Menggunakan deepcopy jika ada nested data, atau sekadar copy() untuk flat dict
    payload_options = copy.deepcopy(sanitized_options)

    try:
        # Lempar payload_options yang sudah aman dan bersih ke plugin
        broadcast_results = plugin.broadcast(
            "pre_execute", 
            metadata=metadata, 
            module=current_module,
            options=payload_options
        )
        
        if isinstance(broadcast_results, dict):
            for plugin_name, result in broadcast_results.items():
                if not isinstance(result, dict):
                    continue
                
                # Evaluasi hasil mutasi yang sah (melalui kontrak kembalian)
                if "modified_options" in result and isinstance(result["modified_options"], dict):
                    # Update dict options UTAMA dengan hasil mutasi plugin
                    # (Hanya menimpa key yang dimodifikasi, tidak menghilangkan key lain)
                    options.update(result["modified_options"])
                    smf.printd("PLUGIN", f"Options strictly mutated by {plugin_name}", level="DEBUG")

                if result.get("handled") is True:
                    smf.printf(f"{CC.GREEN}[+] Execution successfully handled by plugin: {plugin_name}{CC.RESET}")
                    is_handled_by_plugin = True
                    break

    except Exception as e:
        smf.printd("PLUGIN BROADCAST ERROR", e, level="ERROR")

    # ==========================================
    # Fallback Logic (Tidak berubah)
    # ==========================================
    
    if is_handled_by_plugin:
        return

    try:
        if options.get("PASS"):
            full_path = utils.resolve_path(options["PASS"])
            if full_path:
                options["PASS"] = full_path

        current_module.execute(options)

    except AttributeError as e:
        smf.printd("ATTRIBUTE ERROR COMMAND RUN", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] RUN ATTRIBUTE ERROR{CC.RESET}")
    except Exception as e:
        smf.printd("ERROR COMMAND RUN EXCEPTION", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] ERROR DURING EXECUTION{CC.RESET}")
