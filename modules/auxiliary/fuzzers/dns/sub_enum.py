import subprocess
import smf
import sys

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
    if "STATUS" in line:
        return f"[INFO] {CC.YELLOW}=> {line}{CC.RESET}\n"

    if "FOUND =>" in line:
        return f"[✓] {CC.GREEN}{line}{CC.RESET}"

    if "Enumeration" in line:
        return f"\n[✓] {CC.YELLOW}{line}{CC.RESET}"

    if "Error" in line:
        return f"[!] {CC.RED}{line}{CC.RESET}"

    return line


def render_progress_bar(percent: int, width: int = 30) -> str:
    """Menggambar visualisasi progress bar kustom"""
    pos = int((percent * width) / 100)
    bar = "■" * pos + " " * (width - pos)
    # Tampilan ala apt/modern CLI
    return f"\r\033[K[Progress] \033[36m[{bar}] {percent}%\033[0m"

def execute(options):
    target_domain = options.get("DOMAIN")
    wordlist_path = options.get("SUBDOM")
    threads = str(options.get("THREAD"))

    binary = call_bin("dns_sub_enum")

    if not binary:
        smf.printf(f"[!] {CC.YELLOW}WARN => Binary not found at >{CC.RESET}", binary)
        return

    smf.printf(
        f"\n[*] {CC.YELLOW}Starting SUBDOMAIN ENUMERATION for =>{CC.RESET}", target_domain
    )
    smf.printf()

    cmd = [binary, "-d", target_domain, "-w", wordlist_path, "-c", threads]

    process = None
    current_bar = ""  # Menyimpan state visual progress bar saat ini

    try:
        process = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1
        )

        for line in iter(process.stdout.readline, ""):
            cleaned_line = line.rstrip("\r\n")
            if not cleaned_line:
                continue

            # INTERCEPT: Jika baris adalah data progress dari Go
            if cleaned_line.startswith("PROGRESS =>"):
                try:
                    # Ambil angka progress, misal dari "PROGRESS => 45"
                    percent = int(cleaned_line.split("=>")[1].strip())
                    current_bar = render_progress_bar(percent)
                    
                    # Cetak langsung ke terminal tanpa newline
                    sys.stdout.write(current_bar)
                    sys.stdout.flush()
                except (ValueError, IndexError):
                    pass
                continue  # Skip proses log normal

            # JIKA LOG NORMAL: Lakukan langkah penyelamatan agar tidak tabrakan
            stream_line = output_stream(cleaned_line)

            # 1. Hapus progress bar yang sedang menggantung di baris bawah
            if current_bar:
                sys.stdout.write("\r\033[K")
                sys.stdout.flush()

            # 2. Cetak log normal hasil parsing output_stream
            # Memastikan log diakhiri newline agar kursor turun ke bawah
            if not stream_line.endswith("\n"):
                stream_line += "\n"
            smf.printf(stream_line)

            # 3. Cetak ulang progress bar di baris paling bawah yang baru
            if current_bar:
                sys.stdout.write(current_bar)
                sys.stdout.flush()

        process.stdout.close()
        process.wait()

    except KeyboardInterrupt:
        # Bersihkan baris progress saat interrupt sebelum mencetak pesan stop
        if current_bar:
            sys.stdout.write("\r\033[K")
        smf.printf("\n[✓] Sub Enumeration is stopped")

    except Exception as e:
        if current_bar:
            sys.stdout.write("\r\033[K")
        smf.printf(f"\n[!] {CC.RED}An IPC module error occurred{CC.RESET}")
        smf.printd("Subenum IPC error", e, level="ERROR")

    finally:
        if process and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
            
        smf.printf(
            f"[✓] {CC.GREEN}Path Enumeration daemon successfully stopped and cleaned up.{CC.RESET}"
        )
