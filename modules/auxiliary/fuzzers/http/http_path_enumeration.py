# GPL License.
# Copyright (c) 2026 Storm Framework
# See LICENSE file in the project root for full license information.
import subprocess
import os
import smf
import re

from rootmap import ROOT
from apps.utility.colors import *
from lib.roar.calling import call_bin

metadata = {
    "Name": "Path enumeration fuzzing",
    "Description": """
Performs fuzz on the specified url path,
It has two mechanisms: a recursive crawler and 
a combination of a wordlist and a recursive crawler.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Path Enumeration", {"Description": "Perform a search on the URL path"}],
        ["Fuzzing", {"Description": "fuzz URL path list"}],
        ["Crawler", {"Description": "Crawl html for JS and dissect recursively"}],
    ],
    "DefaultAction": "Fuzzing",
    "License": "SMF License",
}
REQUIRED_OPTIONS = {
    "URL": "",
    "PATH": "File wordlist url path (opsional)",
    "THREAD": "Default 1",
}
RESULT_PATTERN = re.compile(
    r"^\[RESULT\]\s+(?:\[(?P<source>[^\]]+)\]\s+)?Path:\s+(?P<path>[^ ]+)\s+\|\s+Status:\s+(?P<status>\d+)\s+\|\s+Size:\s+(?P<size>\d+)\s+\|\s+Type:\s+(?P<type>[^ ]+)"
)


def output_stream(line: str) -> str:
    """
    Color injection for captured logs
    """
    clean_line = line.strip()
    if not clean_line:
        return line

    match = RESULT_PATTERN.match(clean_line)
    if match:
        data = match.groupdict()
        status_code = int(data["status"])

        raw_source = data.get("source")
        type = data.get("type")
        source_engine = raw_source if raw_source else "WORDLIST"

        # Risk Level Coloring / HTTP State Classification
        if 200 <= status_code < 300:
            # 2xx = Success / File Open / Valid Target
            color_status = f"{CC.GREEN}{status_code}{CC.RESET}"
        elif 300 <= status_code < 400:
            # 3xx = Redirects / Potential Routing Bypass
            color_status = f"{CC.YELLOW}{status_code}{CC.RESET}"
        elif 400 <= status_code < 500:
            # 4xx = Forbidden / Unauthorized
            color_status = f"{CC.CYAN}{status_code}{CC.RESET}"
        else:
            # 5xx = Server Error / Potential Crash / Vulnerability Indicator
            color_status = f"{CC.RED}{status_code}{CC.RESET}"

        # Color the type
        if type == "Not":
            color_type = f"{CC.CYAN}{type}{CC.RESET}"
        elif type == "Soft":
            color_type = f"{CC.YELLOW}{type}{CC.RESET}"
        elif type == "Error":
            color_type = f"{CC.RED}{type}{CC.RESET}"
        else:
            color_type = f"{CC.GREEN}{type}{CC.RESET}"

        # Color the source
        if source_engine == "JS":
            color_source = f"{CC.YELLOW}{source_engine:^8}{CC.RESET}"
        elif source_engine == "HTML":
            color_source = f"{CC.CYAN}{source_engine:^8}{CC.RESET}"
        else:
            color_source = f"{CC.WHITE}{source_engine:^8}{CC.RESET}"

        # Log streaming output design
        formatted_line = (
            f"[{CC.GREEN}RESULT{CC.RESET}] "
            f"[{color_source}] "
            f"Path: {CC.MAGENTA}{data['path']:<45}{CC.RESET} | "
            f"Status: {color_status} | "
            f"Size: {CC.WHITE}{data['size']:<8}{CC.RESET} | "
            f"Type: {color_type}"
        )
        return f"{formatted_line}\n"

    if "Error =>" in clean_line:
        return f"[{CC.RED}SYSTEM ERROR{CC.RESET}] {CC.RED}{clean_line.split('=>')[1].strip()}{CC.RESET}\n"

    if "Error" in clean_line:
        return f"{CC.RED}{clean_line}{CC.RESET}"

    # Anomaly Detection / Soft 404
    if "Warning =>" in clean_line:
        return f"[{CC.YELLOW}ANOMALY WARN{CC.RESET}] {CC.YELLOW}{clean_line.split('=>')[1].strip()}{CC.RESET}\n"

    # Detect COMBO or JIT mode info
    if "[INFO] Mode =>" in clean_line:
        return f"{CC.YELLOW}{clean_line}{CC.RESET}\n\n"

    # Normal info detection
    if "[INFO]" in clean_line:
        return f"{CC.YELLOW}{clean_line}{CC.RESET}\n"

    # Detection error
    if "[ERROR]" in clean_line:
        return f"{CC.RED}{clean_line}{CC.RESET}\n"

    return line


def execute(options, runtime):
    url = options.get("URL")
    wordl = options.get("PATH")
    thread = options.get("THREAD")

    # Specifying the Linkfinder regex path
    regex = os.path.join(ROOT, "external", "source", "regex", "rexgo.txt")

    # Call binary
    bin = call_bin("http_path_enum")

    # Binary validation
    if not bin:
        smf.printf(f"[!] Binary => path_enum >> not found")
        return

    # Setting up cmd dynamically
    cmd = [bin, "-url", url, "-threads", thread, "-regex", regex]

    # Wordlist input validation
    if wordl and os.path.exists(wordl):
        cmd.extend(["-wordlist", wordl])
    elif wordl:
        smf.printf(f"[!] Wordlist {wordl} no match. Fallback to automatic.")

    smf.printf(f"[~]{CC.CYAN} Running fuzzing to => {CC.RESET}{url}\n")

    # Run subprocess
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
        smf.printf("\n[✓] path enumeration stopped.")

    # Catch error exception
    except Exception as e:
        smf.printf(f"\n[!]{CC.RED} Error pathenum =>{CC.RESET}", e)
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
            f"\n[✓]{CC.GREEN} Path Enumeration daemon successfully stopped and cleaned up.{CC.RESET}"
        )
