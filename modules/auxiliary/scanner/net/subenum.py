import subprocess
import threading
import smf

from apps.utility.colors import *
from lib.roar.callbin.calling import call_bin

MOD_INFO = {
    "Name": "Searching for subdomains",
    "Description": """
Perform a scan on the specified subdomain
to search and find subdomains that allow for
exploited.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Scanner", {"Description": "Searching for sensitive subdomains"}],
    ],
    "DefaultAction": "Scanner",
    "License": "SMF License",
}
REQUIRED_OPTIONS = {"DOMAIN": "ex: google.com", "SUBDOM": "subdomain", "THREAD": "100"}


def execute(options):
    target_domain = options.get("DOMAIN")
    wordlist_path = options.get("SUBDOM")
    threads = str(options.get("THREAD"))

    binary = call_bin("subenum")

    if not binary:
        smf.printf(f"{CC.YELLOW}[!] WARN => Binary not found at >{CC.RESET}", binary)
        return

    smf.printf(
        f"\n{CC.YELLOW}[*] Starting SUBDOMAIN ENUMERATION for {target_domain}{CC.RESET}"
    )
    smf.printf()

    cmd = [binary, "-d", target_domain, "-w", wordlist_path, "-c", threads]

    try:
        process = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1
        )

        # Thread for parsing valid results (stdout)
        def read_stdout(pipe):
            for line in iter(pipe.readline, ""):
                if line:
                    smf.printf(f"{CC.GREEN}[✓] {line.strip()}{CC.RESET}")
            pipe.close()

        # Thread for parsing info/error (stderr)
        def read_stderr(pipe):
            for line in iter(pipe.readline, ""):
                if line:
                    smf.printf(f"{CC.YELLOW}{line.strip()}{CC.RESET}")
            pipe.close()

        stdout_thread = threading.Thread(target=read_stdout, args=(process.stdout,))
        stderr_thread = threading.Thread(target=read_stderr, args=(process.stderr,))

        stdout_thread.start()
        stderr_thread.start()

        process.wait()
        stdout_thread.join()
        stderr_thread.join()

    except Exception as e:
        smf.printf(f"{CC.RED}[!] An IPC module error occurred{CC.RESET}")
        smf.printd("Subenum IPC error", e, level="ERROR")
