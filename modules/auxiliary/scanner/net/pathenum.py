# GPL License.
# Copyright (c) 2026 Storm Framework
# See LICENSE file in the project root for full license information.
import subprocess
import os
import smf

from apps.utility.colors import *
from lib.roar.calling import call_bin

metadata = {
    "Name": "Path enumeration fuzzing",
    "Description": """
Fuzzing URL paths uses two different 
mechanisms: automatic or via wordlist.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Path Enumeration", {"Description": "Perform a search on the URL path"}],
        ["Fuzzing", {"Description": "fuzz URL path list"}],
    ],
    "DefaultAction": "Fuzzing",
    "License": "SMF License",
}
REQUIRED_OPTIONS = {
    "URL": "",
    "PATH": "File wordlist url path (opsional)",
    "THREAD": "Default 1",
}


def output_stream(line: str) -> str:
    """
    Color injection for captured logs
    """
    if "[ERROR]" in line or "[FATAL]" in line:
        return f"\n{CC.RED}{line}{CC.RESET}"
    elif "[WARN]" in line or "[WARNING]" in line:
        return f"{CC.YELLOW}{line}{CC.RESET}"
    elif "[START]" in line:
        return f"{CC.CYAN}{line}{CC.RESET}\n"
    elif "[INIT]" in line:
        return f"{CC.CYAN}{line}{CC.RESET}"
    elif "[RESULT]" in line:
        return f"\n{CC.GREEN}{line}{CC.RESET}"

    return line  # return line = Standard output


def execute(options, runtime):
    url = options.get("URL")
    wordl = options.get("PATH")
    thread = options.get("THREAD")

    bin = call_bin("path_enum")

    if not os.path.exists(bin):
        smf.printf(f"[!] Binary => {bin} >> not found")
        return

    # Membangun argument baris perintah secara dinamis
    cmd = [bin, "-url", url, "-threads", thread]

    if wordl and os.path.exists(wordl):
        cmd.extend(["-wordlist", wordl])
    elif wordl:
        smf.printf(f"[!] Wordlist {wordl} no match. Fallback to automatic.")

    smf.printf(f"[*] Launching Subprocess: {' '.join(cmd)}")

    # Eksekusi proses dengan pipe stdout untuk streaming data real-time
    process = subprocess.Popen(
        cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1
    )

    # Loop of output stream
    try:
        for line in process.stdout:
            stream_line = output_stream(line)
            smf.printf(stream_line, end="", flush=True)

    # Capture CTRL + C
    except KeyboardInterrupt:
        smf.printf("\n[!] path enumeration stopped.")

    # Catch error exception
    except Exception as e:
        smf.printf(f"\n{CC.RED}[!] Error pathenum =>{CC.RESET}", e)
        smf.printd(f"Exception pathenum", e, level="ERROR")

    # Stop the binary process
    finally:
        if process.poll() is None:  # Check the process in the background
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()

        smf.printf(
            f"{CC.GREEN}[*] Path Enumeration daemon successfully stopped and cleaned up.{CC.RESET}"
        )
