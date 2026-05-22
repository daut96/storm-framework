#Requires -RunAsAdministrator

# --- CONFIGURATION ---
$TOOL_NAME = "storm"
$REPO_NAME = "storm-framework"
$GITHUB_REPO = "https://github.com/StormWorld0/$REPO_NAME.git"

$INSTALL_DIR = "$env:ProgramData\$REPO_NAME"
$BIN_DIR = "$env:ProgramData\storm-bin"

Function Write-Color {
    param($Text, $Color)
    Write-Host $Text -ForegroundColor $Color
}

Function Write-ToFile {
    param($Content, $FilePath, $Append = $false)
    if ($Append) {
        Add-Content -Path $FilePath -Value $Content -NoNewline
    } else {
        Set-Content -Path $FilePath -Value $Content -NoNewline
    }
}

# -----------------------------------------------------------------------------
# --- AUTOMATIC ENVIRONMENT DETECTION & PATH CONFIGURATION ---
# -----------------------------------------------------------------------------
Write-Color "[!] Start Installation: $REPO_NAME [!]" "Green"

# Check Dependencies
$Dependencies = @("git", "python", "docker", "openssl")
foreach ($Dep in $Dependencies) {
    if (!(Get-Command $Dep -ErrorAction SilentlyContinue)) {
        Write-Color "[x] ERROR: $Dep not found. Please install it first or ensure it is in PATH." "Red"
        exit 1
    }
}

# Ensure Docker is running
$DockerStatus = docker info 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Color "[x] ERROR: Docker is not running. Start Docker Desktop first." "Red"
    exit 1
}

# Installation Directory Preparation
if (Test-Path $INSTALL_DIR) {
    Write-Color "[-] Remove old installations." "Green"
    Remove-Item -Path $INSTALL_DIR -Recurse -Force
}

Write-Color "[+] Create installation directory" "Green"

if (!(Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null
}

if ($env:GITHUB_ACTIONS -eq "true") {
    Write-Color "[*] CI Environment detected" "Green"
    Copy-Item -Path ".\*" -Destination $INSTALL_DIR -Recurse -Force
} else {
    Write-Color "[*] Production Environment detected" "Green"
    git clone $GITHUB_REPO $INSTALL_DIR
}

if ($LASTEXITCODE -ne 0) {
    Write-Color "[x] ERROR: Failed to clone repository." "Red"
    Remove-Item -Path $INSTALL_DIR -Recurse -Force
    exit 1
}

# Validate repository structure
if (!(Test-Path "$INSTALL_DIR\pyproject.toml")) {
    Write-Color "[x] ERROR: Failed to prepare repository files." "Red"
    Remove-Item -Path $INSTALL_DIR -Recurse -Force
    exit 1
}

# Docker Build Process
if (Test-Path "$INSTALL_DIR\requirements.txt") {
    Write-Color "[+] Installing Python dependencies and Building Image." "Green"

    $CREATE_DOCKERFILE = @"
FROM python:3.13-slim
RUN apt-get update && apt-get install -y git golang build-essential libpcap-dev clang cmake ffmpeg openssl cargo pkg-config libssl-dev rustc python3-dev
WORKDIR /opt/$REPO_NAME
COPY . /opt/$REPO_NAME
RUN pip install --upgrade pip setuptools wheel
RUN pip install --no-cache-dir -r /opt/$REPO_NAME/requirements.txt
"@
    Write-ToFile $CREATE_DOCKERFILE "$INSTALL_DIR\Dockerfile"

    Write-Color "[+] Initiating Docker Build Process..." "Green"

    if ($env:GITHUB_ACTIONS -eq "true") {
        # Menggunakan & untuk memastikan argumen dibaca sebagai array string oleh Docker
        & docker buildx build --tag "$REPO_NAME" --cache-from type=gha --cache-to type=gha,mode=max --load "$INSTALL_DIR"
    } else {
        # Menggunakan --tag sebagai ganti -t untuk transparansi debugging
        & docker build --tag "$REPO_NAME" "$INSTALL_DIR"
    }

    # Penangkapan error yang lebih robust
    if ($LASTEXITCODE -ne 0) {
        Write-Color "[x] FATAL ERROR: Docker build gagal dengan exit code $LASTEXITCODE" "Red"
        exit 1
    }
}

# Run binary compilation
Set-Location -Path $INSTALL_DIR

# Using Linux container path mapping (/opt/storm-framework) inside Docker
docker run --rm -e GOFLAGS="-buildvcs=false" -v "${INSTALL_DIR}:/opt/$REPO_NAME" -w "/opt/$REPO_NAME" $REPO_NAME python3 -m scripts.cpl.compiler
if ($LASTEXITCODE -ne 0) {
    Write-Color "[x] ERROR: Compilation failed or lost track of directory!" "Red"
    exit 1
}

# --- Security Identity Generation ---
if (!(Test-Path "$INSTALL_DIR\.env")) {
    Write-Color "[+] Generating unique security keys via OpenSSL..." "Green"

    # Generate keys using OpenSSL and convert to Base64 natively
    openssl genpkey -algorithm ed25519 -out "$INSTALL_DIR\temp_priv.pem" 2>$null
    openssl pkey -in "$INSTALL_DIR\temp_priv.pem" -outform DER -out "$INSTALL_DIR\temp_priv.der" 2>$null
    
    $PrivBytes = [System.IO.File]::ReadAllBytes("$INSTALL_DIR\temp_priv.der")
    $PRIV_KEY = [Convert]::ToBase64String($PrivBytes)

    openssl pkey -in "$INSTALL_DIR\temp_priv.pem" -pubout -outform DER -out "$INSTALL_DIR\temp_pub.der" 2>$null
    $PubBytes = [System.IO.File]::ReadAllBytes("$INSTALL_DIR\temp_pub.der")
    $PUB_KEY = [Convert]::ToBase64String($PubBytes)

    if ($PUB_KEY.Length -eq 59) {
        $PUB_KEY += "="
    }

    $EnvContent = "STORM_PRIVKEY=$PRIV_KEY`nSTORM_PUBKEY=$PUB_KEY`n"
    Write-ToFile $EnvContent "$INSTALL_DIR\.env"

    # Cleanup temp files
    Remove-Item "$INSTALL_DIR\temp_*.pem", "$INSTALL_DIR\temp_*.der" -Force

    Write-Color "[✓] Security identity created successfully." "Green"
}

# Go to data folder
$DataDir = "$INSTALL_DIR\data"
if (!(Test-Path $DataDir)) { New-Item -ItemType Directory -Path $DataDir | Out-Null }
Set-Location -Path $DataDir

# Generate ROOT CA TRUST STORE
if ((Test-Path "smf_ca.key") -and (Test-Path "smf_ca.crt")) {
    Write-Color "[*] CA already exists" "Green"
} else {
    Write-Color "[+] Generate CA Trust Store..." "Green"

    Remove-Item "smf_ca.key", "smf_ca.crt" -ErrorAction SilentlyContinue

    openssl genrsa -out smf_ca.key 2048
    openssl req -x509 -new -nodes -key smf_ca.key -sha256 -days 3650 -out smf_ca.crt -subj "/CN=Storm Trusted Root CA/O=StormWorld0/OU=Network-Security-Storm"

    Write-Color "[✓] Success generate CA Trust Store" "Green"
}

# Go back to root folder
Set-Location -Path $INSTALL_DIR

# Sign executed inside Docker
docker run --rm -v "${INSTALL_DIR}:/opt/$REPO_NAME" -w "/opt/$REPO_NAME" $REPO_NAME python3 -m scripts.security.sign

# Creating a Dynamic Wrapper Script (.bat instead of bash script)
if (!(Test-Path $BIN_DIR)) {
    New-Item -ItemType Directory -Force -Path $BIN_DIR | Out-Null
}

$WRAPPER_DST = "$BIN_DIR\$TOOL_NAME.bat"

$CREATE_WRAPPER = @"
@echo off
set PROJECT_DIR=$INSTALL_DIR
cd /d "%PROJECT_DIR%" || (echo [x] ERROR: Failed to access project directory. & exit /b 1)

:: Check if stdin is interactive
set TTY_FLAG=-it
echo %CMDCMDLINE% | findstr /i /c:"%COMSPEC%" >nul
if errorlevel 1 set TTY_FLAG=

set DOCKER_CMD=docker run %TTY_FLAG% --rm --network host -v "%PROJECT_DIR%:/opt/$REPO_NAME" -w "/opt/$REPO_NAME" $REPO_NAME

if "%~1"=="--update" (
    if "%~2"=="" (
        %DOCKER_CMD% ./smfupdate
        exit /b %ERRORLEVEL%
    )
)

if "%~1"=="" (
    if not exist ".\smfstart" (
        %DOCKER_CMD% ./smfupdate
    ) else (
        %DOCKER_CMD% ./smfstart
    )
    exit /b %ERRORLEVEL%
)

echo [x] Error: Command '%*' not found.
exit /b 1
"@

Write-ToFile $CREATE_WRAPPER $WRAPPER_DST

# Add BIN_DIR to System PATH if not already present
$MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($MachinePath -notmatch [regex]::Escape($BIN_DIR)) {
    [Environment]::SetEnvironmentVariable("Path", $MachinePath + ";$BIN_DIR", "Machine")
    Write-Color "[*] Added $BIN_DIR to System PATH. You may need to restart your terminal." "Blue"
}

Write-Color "####################################################" "Green"
Write-Color "[✓] INSTALLATION COMPLETE" "Green"
Write-Color "[✓] PATH STORM: $INSTALL_DIR" "Green"
Write-Color "[✓] PATH WRAPPER: $WRAPPER_DST" "Green"
Write-Color "####################################################" "Green"

