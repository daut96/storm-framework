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
        smf.printf(
            f"{CC.YELLOW}[!] No modules selected. (use <module>) first.{CC.RESET}"
        )
        return

    # 1. Validasi opsi wajib
    required_vars = getattr(current_module, "REQUIRED_OPTIONS", {})
    missing = [
        key for key in required_vars.keys() if not str(options.get(key, "")).strip()
    ]

    if missing:
        smf.printf(
            f"{CC.YELLOW}[!] Failed to run. Variabel null: {', '.join(missing)}{CC.RESET}"
        )
        smf.printf()
        return

    metadata = getattr(current_module, "metadata", {})
    is_handled_by_plugin = False

    # 2. Data Preparation via Defensive Copying
    sanitized_options = {k: v for k, v in options.items() if str(v).strip()}
    payload_options = copy.deepcopy(sanitized_options)

    # 3. Bangun objek runtime terisolasi dari Context Factory
    #    Ini menghasilkan objek 'RuntimeContext' yang siap disuntikkan ke modul.
    module_runtime = ctx.runtime(metadata=metadata, plugin_manager=plugin)

    try:
        # [PERBAIKAN]: Jalankan hook pre_execute via broadcast ke sistem plugin
        broadcast_results = plugin.broadcast(
            "pre_execute",
            metadata=metadata,
            module=current_module,
            options=payload_options,
        )

        # 4. Evaluasi hasil interupsi/mutasi dari plugin
        if isinstance(broadcast_results, dict):
            for plugin_name, result in broadcast_results.items():
                if not isinstance(result, dict):
                    continue

                # Cek jika ada modifikasi options statis dari pintu depan (pre_execute)
                if "modified_options" in result and isinstance(
                    result["modified_options"], dict
                ):
                    options.update(result["modified_options"])
                    smf.printd(
                        "PLUGIN",
                        f"Options strictly mutated by {plugin_name}",
                        level="DEBUG",
                    )

                # Short-circuit jika plugin mengambil alih eksekusi sepenuhnya
                if result.get("handled") is True:
                    smf.printf(
                        f"{CC.GREEN}[+] Execution successfully handled by plugin: {plugin_name}{CC.RESET}"
                    )
                    is_handled_by_plugin = True
                    break

    except Exception as e:
        smf.printd("PLUGIN BROADCAST ERROR", e, level="ERROR")

    # ==========================================
    # Fallback Logic (Clean Injection)
    # ==========================================

    if is_handled_by_plugin:
        return

    try:
        if options.get("PASS"):
            full_path = utils.resolve_path(options["PASS"])
            if full_path:
                options["PASS"] = full_path

        # Jalankan modul dengan menyuntikkan options dan runtime terisolasi
        current_module.execute(options, module_runtime)

    except TypeError as e:
        # Backward compatibility untuk modul-modul lama yang belum migrasi ke parameter runtime
        if "execute() takes 1 positional argument but 2 were given" in str(e):
            smf.printd(
                "LEGACY MODULE DETECTED",
                "Fallback to 1-parameter execution style.",
                level="WARN",
            )
            current_module.execute(options)
        else:
            smf.printd("TYPE ERROR COMMAND RUN", e, level="ERROR")

    except AttributeError as e:
        smf.printd("ATTRIBUTE ERROR COMMAND RUN", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] RUN ATTRIBUTE ERROR{CC.RESET}")

    except Exception as e:
        smf.printd("ERROR COMMAND RUN EXCEPTION", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] ERROR DURING EXECUTION{CC.RESET}")
