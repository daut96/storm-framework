import requests
import argparse
import smf

from apps.utility.colors import *
from urllib3.exceptions import InsecureRequestWarning


metadata = {
    # Unique Identification & Attribution Module
    "Name": "Enterprise Unauthenticated File Creation Vulnerability",
    "Description": """
CVE-2026-20253 is a critical vulnerability in Splunk Enterprise
and Splunk Cloud Platform that allows an unauthenticated
remote attacker to create or truncate arbitrary files
through an exposed PostgreSQL sidecar service endpoint.

Because the vulnerable functionality lacks authentication controls,
attackers can perform file operations without valid credentials.

Successful exploitation may result in:

- Arbitrary file creation
- File truncation
- Data destruction
- Service disruption
- Potential privilege escalation
- Potential system compromise
    """,
    "Author": ["zxelzy"],
    "License": "SMF LICENSE",
    "Date": "2026-06-20",
    "Action": [
        ["Shoot EndPoints", {"Description": "Sending HEADERS with a custom PATH"}],
        ["RESPONSE VALIDATION", {"Description": "Reading response codes and determining vulnerabilities"}],
    ],
    "DefaultAction": "Scanning",

    # Vulnerability Intelligence
    "Vulnerability": {
        "CVE": "CVE-2026-20253",
        "Severity": "CRITICAL",
        "Published": "2026-06-10",
        "Updated": "2026-06-19",
        "References": [
            "https://nvd.nist.gov/vuln/detail/CVE-2026-20253",
            "https://www.cve.org/CVERecord?id=CVE-2026-20253",
            "https://www.tenable.com/cve/CVE-2026-20253",
            "https://github.com/0xBlackash/CVE-2026-20253"
        ]
    }
}
# Variabel OPTIONS
REQUIRED_OPTIONS = {
    "DOMAIN": "Splunk (example: splunk.com)",
    "PORT": "Splunk management port (example: 8089)",
}

# Disable SSL warnings
requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


def check_splunk(host: str, port: int = 8089, prefix: str = "en-US"):
    smf.printf(f"[+] {CC.BLUE}Target =>{CC.RESET} {host}:{port}")
    smf.printf(f"[*] {CC.YELLOW}Probing PostgreSQL sidecar endpoints...{CC.RESET}\n")

    endpoints = [
        f"/{prefix}/splunkd/__raw/v1/postgres/recovery/backup",
        f"/{prefix}/splunkd/__raw/v1/postgres/recovery/restore",
    ]

    vulnerable = False

    for endpoint in endpoints:
        url = f"https://{host}:{port}{endpoint}"
        try:
            smf.printf(f"{CC.CYAN}   ↪ Testing:{CC.RESET} {endpoint}", end=" ")

            r = requests.head(url, verify=False, timeout=10, allow_redirects=True)
            status = r.status_code

            if status in (200, 400, 500):
                smf.printf(f"⚠️{CC.RED}  VULNERABLE (HTTP {status}){CC.RESET}")
                vulnerable = True
            elif status == 401:
                smf.printf(f"✅{CC.GREEN} Protected (HTTP 401){CC.RESET}")
            elif status == 404:
                smf.printf(f"ℹ️{CC.YELLOW}  Not Found (HTTP 404){CC.RESET}")
            else:
                smf.printf(f"{CC.YELLOW}HTTP {status}{CC.RESET}")

        except requests.exceptions.RequestException as e:
            smf.printf(f"{CC.YELLOW}Connection failed:{CC.RESET} {str(e)[:60]}")
            smf.printd("CONNECTION FAILED", e, level="WARN")
            
    smf.printf("\n" + "=" * 78)

    if vulnerable:
        smf.printf(f"🚨{CC.RED}  YOUR SPLUNK INSTANCE IS LIKELY VULNERABLE!{CC.RESET}")
        smf.printf(f"{CC.RED}    Immediate action required!{CC.RESET}")
        smf.printf(f"\n{CC.YELLOW}Recommended Fix:")
        smf.printf("   • Upgrade to Splunk Enterprise 10.2.4 / 10.0.7+")
        smf.printf("   • Restrict access to port 8089 (firewall / network isolation)")
    else:
        smf.printf(
            f"✅{CC.GREEN} No obvious exposure detected on tested endpoints.{CC.RESET}"
        )
        smf.printf(
            f"{CC.YELLOW}   Still strongly recommended to update Splunk!{CC.RESET}"
        )

    smf.printf(f"\n{CC.CYAN}Stay safe & patch fast! - 0xBlackash{CC.RESET}")


# ------ EndPoints main ------ #
def execute(options):
    host = options.get("DOMAIN")
    port = int(options.get("PORT"))
    prefix = "en-US"

    check_splunk(host, port, prefix)
