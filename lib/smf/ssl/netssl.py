import requests
import urllib3


def storm_ssl(method, url, **kwargs):
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    try:
        kwargs.setdefault("verify", False)
        kwargs.setdefault("timeout", 15)

        return requests.request(method, url, **kwargs)

    except requests.exceptions.SSLError as e:
        return f"SSL_HANDSHAKE_FAILED: {e}"
    except Exception as e:
        return f"CONNECTION_ERROR: {e}"
