import requests
import json
import os
import smf

from apps.utility.colors import C
from rootmap import ROOT


def check_update():
    # Url to github data json
    url = "https://raw.githubusercontent.com/StormWorld0/storm-framework/main/data/data.json"
    try:  # Request get data json
        latest_version = requests.get(url, timeout=3).json()["version"]
        # Get local json data
        data = os.path.join(ROOT, "data", "data.json")

        # View contents and search for versions
        with open(data) as f:
            VERSION = json.load(f)["version"]

        # Compare current version with github
        if latest_version > VERSION:
            smf.printf(f"{C.SUCCESS}[!] Current version => v{VERSION}")
            smf.printf(f"{C.SUCCESS}[!] Update available => v{latest_version}")
            smf.printf(f"{C.SUCCESS}[-] Type => storm update")
            smf.printf()

    except requests.exceptions.RequestException as e:
        smf.printd("ERROR CONNECTION CHECK UPDATE =>", e, level="ERROR")
        smf.printf("CONNECTION TIMEOUT CHECK UPDATE")

    except Exception as e:
        smf.printd("ERROR CHECK UPDATE", e, level="ERROR")
        smf.printf(f"ERROR CHECK UPDATE")
