import subprocess
import sys
import smf
import threading

from apps.utility.colors import *

from lib.roar.callbin.calling import call_bin

MOD_INFO = {
    "Name": "Forward proxy http",
    "Description": """

""",
    "Author": ["zxelzy"],
    "Action": [
        ["Proxy", {"Description": "Reading headers and bodies"}],
        ["DPI", {"Description": "Perform inspection in body header"}],
    ],
    "DefaultAction": "Forward Proxy",
    "License": "SMF License",
}


def stream_reader(pipe, color, is_error=False):
    try:
        for line in iter(pipe.readline, ""):
            if not line:
                break

            line = line.rstrip()

            # custom color output
            sys.stdout.write(f"{color}{line}{CC.RESET}\n")
            sys.stdout.flush()

            # debug stderr
            if is_error:
                smf.printf(
                    f"{color}{line}{CC.RESET}", file=sys.stdout, flush=True, end=""
                )
                smf.printd("Stderr https_proxy", line, level="DEBUG")

    except Exception as e:
        smf.printf(f"{CC.RED}Error exception https_proxy =>{CC.RESET}", e)
        smf.printd("Exception https_proxy", e, level="ERROR")

    finally:
        pipe.close()


def execute(options):
    binary = call_bin("https_prox")

    if not binary:
        smf.printf(f"{CC.YELLOW}[!] Binary not found =>{CC.RESET}", binary)
        return

    cmd = [binary]

    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
        universal_newlines=True,
    )

    try:
        stdout_thread = threading.Thread(
            target=stream_reader,
            args=(process.stdout, CC.GREEN, False),
            daemon=True,
        )

        stderr_thread = threading.Thread(
            target=stream_reader,
            args=(process.stderr, CC.RED, True),
            daemon=True,
        )

        stdout_thread.start()
        stderr_thread.start()

        process.wait()

        stdout_thread.join()
        stderr_thread.join()

    except KeyboardInterrupt:
        smf.printf(f"{CC.YELLOW}[*] Stopping proxy...{CC.RESET}")

        process.terminate()
        process.wait()
