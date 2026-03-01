import os
import subprocess
import sys
from pathlib import Path

def setup_ghunt(base_path):
    print(f"[*] Starting GHunt Environment Setup at: {base_path}")
    
    venv_dir = base_path / "venv"
    # Menyesuaikan path executable berdasarkan OS
    if os.name == "nt":  # Windows
        python_exe = venv_dir / "Scripts" / "python.exe"
        pip_exe = venv_dir / "Scripts" / "pip.exe"
    else:  # Linux/Mac
        python_exe = venv_dir / "bin" / "python"
        pip_exe = venv_dir / "bin" / "pip"

    try:
        # 1. Buat Virtual Environment
        if not venv_dir.exists():
            print("[+] Creating Virtual Environment...")
            subprocess.run([sys.executable, "-m", "venv", str(venv_dir)], check=True)
        else:
            print("[!] Venv already exists. Skipping creation.")

        # 2. Upgrade Pip & Install Requirements
        print("[+] Installing/Updating GHunt dependencies...")
        # Kita install langsung 'ghunt' dari PyPI atau dari folder jika ada requirements.txt
        subprocess.run([str(pip_exe), "install", "--upgrade", "pip"], check=True)
        subprocess.run([str(pip_exe), "install", "ghunt"], check=True)

        # 3. Install Playwright (Wajib untuk GHunt login/scraping)
        print("[+] Installing Playwright browsers...")
        subprocess.run([str(python_exe), "-m", "playwright", "install", "chromium"], check=True)

        print("\n[✔] GHunt Installation Complete!")
        return True

    except Exception as e:
        print(f"\n[✘] Installation Failed: {e}")
        return False

# Jalankan setup
# base_path = Path(ROOT) / "script" / "ghunt"
# setup_ghunt(base_path)

