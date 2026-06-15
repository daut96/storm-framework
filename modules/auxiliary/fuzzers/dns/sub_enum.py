import subprocess
import threading
import smf

from apps.utility.colors import *
from lib.roar.calling import call_bin

metadata = {
    "Name": "Searching for subdomains",
    "Description": """
Perform a scan on the specified subdomain
to search and find subdomains that allow for
exploited.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Sub Enumeration", {"Description": "Searching for valid subdomains"}],
        ["Scanner", {"Description": "Searching for sensitive subdomains"}],
    ],
    "DefaultAction": "Scanner",
    "License": "SMF License",
    "Date": "2026-04-20",
}
REQUIRED_OPTIONS = {
    "DOMAIN": "ex: google.com",
    "SUBDOM": "Path to wordlist subdomain",
    "THREAD": "default 1",
}


def output_stream(line: str) -> str:
    """Color log stdout"""
    if "[INFO] =>" in line:
        return f"{CC.YELLOW}{line}{CC.RESET}\n"

    if "FOUND =>" in line:
        return f"[✓] {CC.GREEN}{line}{CC.RESET}\n"

    if "[✓]" in line:
        return f"\n{CC.YELLOW}{line}{CC.RESET}\n"

    if "[!]" in line:
        return f"{CC.RED}{line}{CC.RESET}\n"

    return f"{line}\n"


def execute(options):
    target_domain = options.get("DOMAIN")
    wordlist_path = options.get("SUBDOM")
    threads = str(options.get("THREAD"))

    binary = call_bin("dns_sub_enum")

    if not binary:
        smf.printf(f"{CC.YELLOW}[!] WARN => Binary not found at >{CC.RESET}", binary)
        return

    smf.printf(
        f"\n{CC.YELLOW}[*] Starting SUBDOMAIN ENUMERATION for => {target_domain}{CC.RESET}"
    )
    smf.printf()

    cmd = [binary, "-d", target_domain, "-w", wordlist_path, "-c", threads]

    process = None
    try:
        process = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1
        )

        # Thread for parsing valid results (stdout)
        for line in iter(process.stdout.readline, ""):
            cleaned_line = line.rstrip("\r\n")
            if line:
                stream_line = output_stream(cleaned_line)
                smf.printf(stream_line)

        process.stdout.close()
        process.wait()
        
    except KeyboardInterrupt:
        smf.printf("\n[✓] Sub Enumeration is stopped")

    except Exception as e:
        smf.printf(f"{CC.RED}[!] An IPC module error occurred{CC.RESET}")
        smf.printd("Subenum IPC error", e, level="ERROR")

    finally:
        if process.poll() is None:  # Check the process in the background
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
        smf.printf(
            f"[✓]{CC.GREEN} Path Enumeration daemon successfully stopped and cleaned up.{CC.RESET}"
        )
