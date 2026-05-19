# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import os
import importlib.util
import textwrap
import smf

from apps.utility.colors import *


# For those who like CVE collections, this logic is definitely needed
# Because this will produce output that is neat in structure and style.
# For ease of reading, and to differentiate between Description, name, ID, etc.
# The most important thing is to make sure that the CVE uses the example data format that has been provided.
# Otherwise the output will be messy and not according to storm rules.
def execute(args: list[str], ctx: "Context") -> None:
    query = args[0] if args else ""
    if not query:
        smf.printf(f"{CC.YELLOW}[!] Enter file name to info!{CC.RESET}")
        return

    # This is a special logic to know where the CVE is located.
    # Make sure CVE is always in the vulnerability folder
    vuln_path = "modules/"
    found_path = ""
    for root, dirs, files in os.walk(vuln_path):
        if f"{query}.py" in files:
            found_path = os.path.join(root, f"{query}.py")
            break

    filename = os.path.basename(found_path).lower()
    if found_path:
        if filename.startswith("cve"):
            # Used to display cve // vulnerability information
            # Command => info <cve_name>
            try:
                spec = importlib.util.spec_from_file_location("temp_mod", found_path)
                mod = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(mod)

                # --- GET DICTIONARY CVE_INFO ---
                info = mod.metadata
                width = 55

                smf.printf()
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")
                smf.printf(
                    f"{CC.CYAN}{'STORM VULNERABILITY KNOWLEDGE BASE':^55}{CC.RESET}"
                )
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")

                smf.printf(
                    f"{CC.CYAN}{'ID CVE':<13} : {CC.YELLOW}{info['cve']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'NAME':<13} : {CC.YELLOW}{info['name']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'LEVEL':<13} : {CC.YELLOW}{info['severity']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'PUBLISHED':<13} : {CC.YELLOW}{info['published']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'UPDATED':<13} : {CC.YELLOW}{info['updated']}{CC.RESET}"
                )
                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")

                smf.printf(f"{CC.YELLOW}DESCRIPTION{CC.RESET}   :")
                desc = textwrap.fill(
                    info["description"].strip(),
                    width=width - 2,
                    initial_indent=" ",
                    subsequent_indent=" ",
                )
                smf.printf(f"{CC.YELLOW}{desc}{CC.RESET}")

                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")
                smf.printf(f"{CC.CYAN}REFERENCES{CC.RESET}    :")
                for link in info["URL"]:
                    smf.printf(f" - {CC.YELLOW}{link}{CC.RESET}")
                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")

                smf.printf(
                    f"{CC.CYAN}{'SCANNER':<13} : {CC.YELLOW}{info['scanner']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'EXPLOIT':<13} : {CC.YELLOW}{info['exploit']}{CC.RESET}"
                )
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")
                smf.printf()

            except Exception as e:
                smf.printd("FAILED TO READ INFORMATION CVE", e, level="ERROR")
                smf.printf(f"{CC.YELLOW}[!] Failed to read CVE{CC.RESET}")

        else:
            # To display information about a specific module
            # Command => info <modules_name>
            try:
                spec = importlib.util.spec_from_file_location("temp_mod", found_path)
                mod = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(mod)

                # --- GET DICTIONARY MOD_INFO ---
                info = mod.metadata
                width = 55
                label_w = 13

                smf.printf()
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")
                smf.printf(f"{CC.CYAN}{'STORM INFORMATION MODULES':^55}{CC.RESET}")
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")

                smf.printf(
                    f"{CC.CYAN}{'NAME':<13} : {CC.YELLOW}{info['Name']}{CC.RESET}"
                )
                smf.printf(f"{CC.CYAN}DESCRIPTION{CC.RESET}   :")
                desc = textwrap.fill(
                    info["Description"].strip(),
                    width=width - 2,
                    initial_indent=" ",
                    subsequent_indent=" ",
                )
                smf.printf(f"{CC.YELLOW}{desc}{CC.RESET}")

                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")
                authors = info.get("Author", [])
                first_auth = authors[0] if authors else "Unknown"
                smf.printf(
                    f"{CC.CYAN}{'AUTHOR':<{label_w}} : - {CC.YELLOW}{first_auth}{CC.RESET}"
                )
                for extra in authors[1:]:
                    smf.printf(f"{' '*(label_w)} : - {CC.YELLOW}{extra}{CC.RESET}")

                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")
                smf.printf(f"{CC.CYAN}{'ACTION':<13}{CC.RESET}")
                for action in info.get("Action", []):
                    name = action[0]
                    desc = action[1].get("Description", "")
                    smf.printf(f"  > {CC.YELLOW}{name:<9} : {desc}{CC.RESET}")

                smf.printf(f"{CC.MAGENTA}{'-'*width}{CC.RESET}")
                smf.printf(
                    f"{CC.CYAN}{'DefAction':<13} : {CC.YELLOW}{info['DefaultAction']}{CC.RESET}"
                )
                smf.printf(
                    f"{CC.CYAN}{'LICENSE':<13} : {CC.YELLOW}{info['License']}{CC.RESET}"
                )
                smf.printf(f"{CC.MAGENTA}{'='*width}{CC.RESET}")
                smf.printf()

            except Exception as e:
                smf.printd("FAILED TO READ INFORMATION MODULE", e, level="ERROR")
                smf.printf(f"{CC.YELLOW}[!] Failed to read MODULE{CC.RESET}")
    else:
        smf.printf(f"{CC.YELLOW}[!] WARN => {query} > not found.{CC.RESET}")
