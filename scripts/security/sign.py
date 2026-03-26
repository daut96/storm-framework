def run_sign():
    try:
        from external.source.binary import sign

        sign()
        return True
    except ImportError as e:
        return f"ERROR => {e}"


if __name__ == "__main__":
    run_sign()
