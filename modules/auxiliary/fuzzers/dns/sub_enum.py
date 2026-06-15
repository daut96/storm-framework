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
log_lock = threading.Lock()


def output_stream(line: str) -> str:
    """Color log stdout"""
    if "[INFO] =>" in line:
        return f"{CC.YELLOW}{line}{CC.RESET}"

    if "FOUND =>" in line:
        return f"[✓] {CC.GREEN}{line}{CC.RESET}"

    if "[*]" in line:
        return f"{CC.YELLOW}{line}{CC.RESET}"

    return line


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
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1
        )

        # Thread for parsing valid results (stdout)
        def read_stdout(pipe):
            for line in iter(pipe.readline, ""):
                cleaned_line = line.rstrip("\r\n")
                if cleaned_line:
                    stream_line = output_stream(cleaned_line)
                    with log_lock:
                        smf.printf(stream_line)
            pipe.close()

        # Thread for parsing info/error (stderr)
        def read_stderr(pipe):
            for line in iter(pipe.readline, ""):
                cleaned_line = line.rstrip("\r\n")
                if cleaned_line:
                    with log_lock:
                        smf.printf(f"{CC.YELLOW}{cleaned_line}{CC.RESET}\n")
            pipe.close()

        stdout_thread = threading.Thread(target=read_stdout, args=(process.stdout,))
        stderr_thread = threading.Thread(target=read_stderr, args=(process.stderr,))

        stdout_thread.start()
        stderr_thread.start()

        process.wait()
        stdout_thread.join()
        stderr_thread.join()

    except KeyboardInterrupt:
        with log_lock:
            smf.printf("\n[✓] Sub Enumeration is stopped")

    except Exception as e:
        with log_lock:
            smf.printf(f"{CC.RED}[!] An IPC module error occurred{CC.RESET}")
        smf.printd("Subenum IPC error", e, level="ERROR")

    finally:
        if process.poll() is None:  # Check the process in the background
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
        with log_lock:
            smf.printf(
                f"[✓]{CC.GREEN} Path Enumeration daemon successfully stopped and cleaned up.{CC.RESET}"
            )
