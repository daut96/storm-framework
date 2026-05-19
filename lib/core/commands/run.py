# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
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
        smf.printf(
            f"{CC.YELLOW}[!] No modules selected. (use <module>) first.{CC.RESET}"
        )
        return

    # 1. Validasi requirement module
    required_vars = getattr(current_module, "REQUIRED_OPTIONS", {})
    missing = [
        key for key in required_vars.keys() if not str(options.get(key, "")).strip()
    ]

    if missing:
        smf.printf(f"{CC.YELLOW}[!] Failed to run. Variabel null: {', '.join(missing)}{CC.RESET}")
        smf.printf()
        return

    # 2. Ambil metadata dinamis dari module
    metadata = getattr(current_module, "metadata", {})
    
    # Flag penanda apakah eksekusi sudah diambil alih oleh plugin
    is_handled_by_plugin = False

    try:
        # 3. Lempar event 'pre_execute' ke semua plugin aktif.
        # Kita juga passing 'options' agar plugin bisa membaca atau memodifikasi payload.
        broadcast_results = plugin.broadcast(
            "pre_execute", 
            metadata=metadata, 
            module=current_module,
            options=options
        )
        
        # 4. Evaluasi kembalian dari manager.broadcast (Dict[plugin_name, plugin_result])
        if isinstance(broadcast_results, dict):
            for plugin_name, result in broadcast_results.items():
                if not isinstance(result, dict):
                    continue
                
                # Jika plugin memodifikasi payload sebelum fallback ke module normal
                if "modified_options" in result and isinstance(result["modified_options"], dict):
                    options.update(result["modified_options"])
                    smf.printd("PLUGIN", f"Options modified by {plugin_name}", level="DEBUG")

                # Jika plugin mengambil alih eksekusi secara penuh
                if result.get("handled") is True:
                    smf.printf(f"{CC.GREEN}[+] Execution successfully handled by plugin: {plugin_name}{CC.RESET}")
                    is_handled_by_plugin = True
                    break # Short-circuit, hentikan iterasi plugin lain

    except Exception as e:
        smf.printd("PLUGIN BROADCAST ERROR", e, level="ERROR")
        # Jika broadcast gagal, kita tetap membiarkan sistem melanjutkan ke fallback (fail-open)

    # 5. Logika Fallback
    # Jika ada plugin yang me-return handled: True, hentikan eksekusi command run di sini.
    if is_handled_by_plugin:
        return

    # Jika tidak ada plugin yang menangani (atau plugin absen), fallback ke eksekusi module normal
    try:
        if options.get("PASS"):
            full_path = utils.resolve_path(options["PASS"])
            if full_path:
                options["PASS"] = full_path

        # Eksekusi fungsi utama module
        current_module.execute(options)

    except AttributeError as e:
        smf.printd("ATTRIBUTE ERROR COMMAND RUN", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] RUN ATTRIBUTE ERROR{CC.RESET}")

    except Exception as e:
        smf.printd("ERROR COMMAND RUN EXCEPTION", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] ERROR DURING EXECUTION{CC.RESET}")
        
