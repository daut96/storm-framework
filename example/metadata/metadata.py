metadata = {
    # 1. Unique Identification & Attribution Module
    "Name": "Exploit/Scan Module Name",
    "Description": "Complete explanation of what this module does, its impact, and its scope.",
    "Author": ["Your Name"],
    "License": "SMF LICENSE",
    "Action": [
        ["Bruteforce", {"Description": "Bypass Telnet login"}],
        ["DPI", {"Description": "Deep Packet Inspection"}]
    ],
    "DefaultAction": "check_vulnerability",
    
    # 2. Vulnerability Intelligence (optional)
    "Vulnerability": {
        "CVE": "CVE-XXXX-XXXX",
        "Severity": "CRITICAL",  # CRITICAL/HIGH/MEDIUM/LOW
        "Published": "YYYY-MM-DD",
        "Updated": "YYYY-MM-DD",
        "References": [
            "https://nvd.nist.gov/vuln/detail/CVE-XXXX-XXXX"
        ]
    },
    
    # 4. Optional Features (Evasion / Data Obfuscation)
    "Transforms": {
        "Base64": True,
        "XOR": True,
        "SessionAware": False
    }
}

