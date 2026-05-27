# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import smf
import apps.utility.utils as utils

from apps.utility.colors import *


# Run command to run a module that we want to execute
# The workflow is as below;
#
# Command => use <module_name> or <path_module>
# Command => set <var> <val>
# Command => run
#
# After running the module will run and you just have to wait for it to finish.
def execute(args, ctx):
    current_module = ctx.current_module
    options = ctx.options
    plugin = ctx.plugin

    if not current_module:
        smf.printf(
            f"{CC.YELLOW}[!] No modules selected. (use <module>) first.{CC.RESET}"
        )
        return

    # Validate options
    required_vars = getattr(current_module, "REQUIRED_OPTIONS", {})
    ignore = {"PASS", "PATH"}

    missing = [
        key
        for key in required_vars
        if key not in ignore and not str(options.get(key, "")).strip()
    ]

    if missing:
        smf.printf(
            f"{CC.YELLOW}[!] Failed to run. Variabel null: {', '.join(missing)}{CC.RESET}"
        )
        return

    metadata = getattr(current_module, "metadata", {})

    # Path Resolution for local files
    if options.get("PASS"):
        full_path = utils.resolve_path(options["PASS"])
        if full_path:
            options["PASS"] = full_path

    # Runtime
    module_runtime = ctx.runtime(metadata=metadata, plugin_manager=plugin)

    # Execution module
    try:
        current_module.execute(options, module_runtime)
    except TypeError as e:
        # Fallback to 1 parameter options
        if "execute() takes 1 positional argument but 2 were given" in str(e):
            current_module.execute(options)
        else:
            smf.printd("TYPE ERROR COMMAND RUN", e, level="ERROR")

    except AttributeError as e:
        smf.printd("ATTRIBUTE ERROR COMMAND RUN", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] RUN ATTRIBUTE ERROR{CC.RESET}")

    except Exception as e:
        smf.printd("ERROR COMMAND RUN EXCEPTION", e, level="ERROR")
        smf.printf(f"{CC.RED}[!] ERROR DURING EXECUTION{CC.RESET}")
