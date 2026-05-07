import smf
import json
import time

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
    smf.printf(f"[*] Status      : Sending requests through the Rust engine...\n")

    # Simulasi header browser yang natural
    
    start_time = time.time()

    try:
        # Menembak target menggunakan GET method
        response = stls.get(target_url,
                            headers = {"Content-Type": "application/json"}
                            body = {"Platform": "Chrome"}
        )
        elapsed_time = time.time() - start_time

        smf.printf(
            f"[+] Request Successful! (Travel time: {elapsed_time:.2f} second)\n"
        )

        data = response.json()

        if not data:
            smf.printf("Empty response from server")
            return

        outer = data

        # parse inner JSON
        outer["data"] = json.loads(outer["data"])

        http_version = outer.get("http_version", "Unknown")
        ja3_hash = outer.get("tls", {}).get("ja3_hash", "Not detected")
        ja4_hash = outer.get("tls", {}).get("ja4", "Not detected")
        akamai_fp = outer.get("http2", {}).get(
            "akamai_fingerprint_hash", "Not detected"
        )

        smf.printf(f"HTTP Version : {http_version}")
        smf.printf(f"JA3          : {ja3_hash}")
        smf.printf(f"JA4          : {ja4_hash}")
        smf.printf(f"Akamai FP    : {akamai_fp}")

        smf.printf("[*] TLS Details:")
        smf.printf(json.dumps(outer, indent=4))

    except KeyboardInterrupt:
        return
    except Exception as e:
        smf.printf("Outer exception error", e)
