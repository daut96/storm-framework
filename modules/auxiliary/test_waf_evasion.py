import smf
import json
import time
import sys

from scripts.wrapper import stls

REQUIRED_OPTIONS = {"URL": ""}


def execute(options):

    # Peet.ws adalah standar industri untuk memvalidasi TLS/HTTP2 fingerprint
    target_url = options.get("URL")
    
    smf.printf("==================================================")
    smf.printf("[*] STORM FRAMEWORK - STLS EVASION TEST")
    smf.printf("==================================================")
    smf.printf(f"[*] Target      : {target_url}")
    smf.printf(f"[*] Time        : {time.strftime('%Y-%m-%d %H:%M:%S')}")
    smf.printf(f"[*] Status      : Mengirim request melalui mesin Rust...\n")

    # Simulasi header browser yang natural
    headers = {
        "accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
        "accept-encoding": "gzip, deflate, br, zstd",
        "accept-language": "en-US,en;q=0.9",
        "sec-ch-ua": '"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"',
        "sec-ch-ua-mobile": "?0",
        "sec-ch-ua-platform": '"Windows"',
        "sec-fetch-dest": "document",
        "sec-fetch-mode": "navigate",
        "sec-fetch-site": "none",
        "sec-fetch-user": "?1",
        "upgrade-insecure-requests": "1",
        "user-agent": "Mozilla/5.0 (Linux; Android 15; Mobile) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    }

    start_time = time.time()
    
    try:
        # Menembak target menggunakan GET method
        response = stls.get(target_url, headers=headers)
        elapsed_time = time.time() - start_time
        
        smf.printf(f"[+] Request Successful! (Travel time: {elapsed_time:.2f} second)\n")
        
        # Parsing JSON response dari Peet.ws
        data = response.json()
        
        # Ekstraksi metrik krusial
        http_version = data.get("http_version", "Unknown")
        ja3_hash = data.get("tls", {}).get("ja3_hash", "Not detected")
        ja4_hash = data.get("tls", {}).get("ja4", "Not detected")
        akamai_fp = data.get("http2", {}).get("akamai_fingerprint_hash", "Not detected")
        
        smf.printf("==================================================")
        smf.printf("[+] FINGERPRINT ANALYSIS (WAF EVASION METRICS)")
        smf.printf("==================================================")
        smf.printf(f"    HTTP Version Negotiated : {http_version}")
        smf.printf(f"    JA3 Fingerprint Hash    : {ja3_hash}")
        smf.printf(f"    JA4 Fingerprint Hash    : {ja4_hash}")
        smf.printf(f"    Akamai H2 Fingerprint   : {akamai_fp}")
        smf.printf("==================================================\n")
        
        smf.printf("[*] Full details of TLS Ciphers & Extensions the server sees:")
        # Menampilkan detail raw TLS dari server (hanya bagian tls)
        smf.printf(json.dumps(data.get("tls", {}), indent=4))

    except Exception as e:
        smf.printf(f"\n[!] FAIL! STLS engine crashed or experienced an error:")
        smf.printf(f"    {e}")
