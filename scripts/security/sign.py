import json
import hashlib
import base64
from rootmap import ROOT
from app.utility.spin import StormSpin
from cryptography.hazmat.primitives.asymmetric import ed25519


def run_sign():
    try:
        from external.source.binary import sign

        sign()
        return True
    except ImportError as e:
        return f"ERROR => {e}"
        
if __name__ == "__main__":
    run_sign()
