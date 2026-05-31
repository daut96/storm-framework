import subprocess
import smf

from apps.utility.colors import *
from lib.roar.calling import call_bin

metadata = {
    "Name": "Bruteforce Telnet login",
    "Description": """
Matching Telnet login username and password
to find out if a Telnet is using standard login auth.
Using 2 test stages, the first with standard auth
The second stage uses the custom keyword.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Bruteforce", {"Description": "Bypass Telnet login"}],
    ],
    "DefaultAction": "Bruteforce",
    "License": "SMF License",
}

SYM_SUCCESS = "🔑"
SYM_FAILED = "🔒"

REQUIRED_OPTIONS = {
    "IP": "",
    "THREAD": "Default 1 thread",
    "PASS": "fill with wordlist",
    "USER": "fill with wordlist",
}


def output_stream(line: str) -> str:
    """
    Inspects each line of stdout and injects ANSI color codes
    based on the log pattern (keyword).
    """
    if "[ERROR]" in line or "[FATAL]" in line:
        return f"\n{CC.RED}{line}{CC.RESET}"
    elif "[INFO]" in line or "[WARNING]" in line:
        return f"{CC.YELLOW}{line}{CC.RESET}\n"
    elif "[SUCCESS]" in line:
        return f"\n{SYM_SUCCESS} {CC.GREEN}{line}{CC.RESET}"
    elif "[RESULT]" in line:
        return f"{SYM_FAILED} {CC.CYAN}{line}{CC.RESET}"
    elif "[SUCC]" in line:
        return f"\n{CC.GREEN}{line}{CC.RESET}"

    return line  # return line = Standard output


def execute(options):

    # Take input
    ip = options.get("IP")
    thread = options.get("THREAD")
    user = options.get("USER")
    passw = options.get("PASS")

    # Binary path
    bin_path = call_bin("telnet_brute_ak")

    # Binary validation
    if not bin_path:
        smf.printf(f"{CC.RED}[!] Binary not found.{CC.RESET}", bin_path)
        return

    # Enter the required data
    cmd = [
        bin_path,
        "-target",
        ip,
        "-threads",
        thread,
        "-users",
        user,
        "-password",
        passw,
    ]

    smf.printf(f"{CC.CYAN}[*] Starting Telnet Bruteforce => {ip}:23{CC.RESET}\n\n")

    # bufsize=1 (Line buffered) ensures every \n is sent directly to Python's stdout
    # without waiting for the OS memory buffer to fill up.
    process = subprocess.Popen(
        cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1
    )

    try:  # Loop of output stream
        for line in process.stdout:
            stream_line = output_stream(line)
            smf.printf(stream_line, end="", flush=True)

    # Monitor CTRL + C
    except KeyboardInterrupt:
        smf.printf(f"\n{CC.YELLOW}[*] Proxy stopped.{CC.RESET}")

    # Catch error exception
    except Exception as e:
        smf.printf(f"\n{CC.RED}[!] Error =>{CC.RESET}", e)
        smf.printd(f"Exception Telnet Bruteforce", e, level="ERROR")

    # Stop the binary process
    finally:
        if process.poll() is None:  # Check the process in the background
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()

        smf.printf(
            f"{CC.GREEN}[*] Telnet brute daemon successfully stopped and cleaned up.{CC.RESET}"
        )
