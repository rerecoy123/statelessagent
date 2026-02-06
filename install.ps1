# SAME installer for Windows PowerShell
# Usage: irm https://statelessagent.com/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Force TLS 1.2 (required for older Windows)
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Colors - use [char]27 for PowerShell 5.1 compatibility
$ESC = [char]27
$Red = "$ESC[91m"
$DarkRed = "$ESC[31m"
$Dim = "$ESC[2m"
$Bold = "$ESC[1m"
$Reset = "$ESC[0m"

# Detect if terminal supports ANSI (Windows Terminal, PS7, etc)
$supportsANSI = $env:WT_SESSION -or $PSVersionTable.PSVersion.Major -ge 7 -or $env:TERM_PROGRAM
if (-not $supportsANSI) {
    $Red = ""; $DarkRed = ""; $Dim = ""; $Bold = ""; $Reset = ""
}

# Banner
Write-Host ""
Write-Host "${Red}  STATELESS AGENT${Reset}"
Write-Host "${Dim}  Every AI session starts from zero.${Reset} ${Bold}${Red}Not anymore.${Reset}"
Write-Host ""

# Step 1: Detect system
Write-Host "[1/4] Detecting your system..."
Write-Host ""

$arch = [System.Environment]::Is64BitOperatingSystem
if (-not $arch) {
    Write-Host "  SAME requires 64-bit Windows."
    Write-Host "  Please ask for help: https://discord.gg/GZGHtrrKF2"
    exit 1
}

Write-Host "  Found: Windows (64-bit)"
Write-Host "  PowerShell: $($PSVersionTable.PSVersion)"
Write-Host "  Perfect, I have a version for you."
Write-Host ""

# Step 2: Download
Write-Host "[2/4] Downloading SAME..."
Write-Host ""

$repo = "sgx-labs/statelessagent"

try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $version = $release.tag_name
} catch {
    Write-Host "  Couldn't reach GitHub to get the latest version."
    Write-Host "  Check your internet connection and try again."
    Write-Host ""
    Write-Host "  If you're behind a corporate proxy, you may need IT help."
    Write-Host "  Discord: https://discord.gg/GZGHtrrKF2"
    exit 1
}

Write-Host "  Latest version: $version"

$binaryName = "same-windows-amd64.exe"
$url = "https://github.com/$repo/releases/download/$version/$binaryName"
$tempFile = Join-Path $env:TEMP "same-download.exe"

try {
    Invoke-WebRequest -Uri $url -OutFile $tempFile -UseBasicParsing
} catch {
    Write-Host ""
    Write-Host "  Download failed. This might mean:"
    Write-Host "  - Your internet connection dropped"
    Write-Host "  - GitHub is having issues"
    Write-Host "  - Corporate firewall is blocking the download"
    Write-Host ""
    Write-Host "  Try again in a minute. If it keeps failing:"
    Write-Host "  https://discord.gg/GZGHtrrKF2"
    exit 1
}

Write-Host "  Downloaded successfully."
Write-Host ""

# Step 3: Install
Write-Host "[3/4] Installing SAME..."
Write-Host ""

$installDir = Join-Path $env:LOCALAPPDATA "Programs\SAME"

Write-Host "  I'm going to put SAME in:"
Write-Host "  $installDir"
Write-Host ""

if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    Write-Host "  Created that folder."
}

$output = Join-Path $installDir "same.exe"
Move-Item -Path $tempFile -Destination $output -Force

# Unblock the file (removes "downloaded from internet" flag)
try {
    Unblock-File -Path $output -ErrorAction SilentlyContinue
} catch {
    # Ignore - not critical
}

# Verify it works
$installedVersion = $null
try {
    $installedVersion = & $output version 2>&1
    if ($LASTEXITCODE -ne 0) { throw "exit code $LASTEXITCODE" }
    Write-Host "  Installed: $installedVersion"
} catch {
    Write-Host ""
    Write-Host "  ${Red}The program downloaded but won't run.${Reset}"
    Write-Host ""
    Write-Host "  This usually means Windows Defender or antivirus blocked it."
    Write-Host "  Try these steps:"
    Write-Host ""
    Write-Host "  1. Open Windows Security"
    Write-Host "  2. Go to Virus & threat protection > Protection history"
    Write-Host "  3. Look for 'same.exe' and click 'Allow'"
    Write-Host ""
    Write-Host "  Or manually unblock: Right-click same.exe > Properties > Unblock"
    Write-Host "  File location: $output"
    Write-Host ""
    Write-Host "  Still stuck? Discord: https://discord.gg/GZGHtrrKF2"
    exit 1
}

Write-Host ""

# Step 4: Add to PATH
Write-Host "[4/4] Setting up your terminal..."
Write-Host ""

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$installDir*") {
    $newPath = "$installDir;$currentPath"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "  Added SAME to your PATH (permanent)."
}

# Also add to current session so user can use it immediately
$env:Path = "$installDir;$env:Path"
Write-Host "  SAME is now available in this terminal session."

Write-Host ""

# Check for Ollama - try multiple detection methods
Write-Host "-----------------------------------------------------------"
Write-Host ""

$ollamaFound = $false
$ollamaHow = ""

# Method 1: Check if ollama command exists
if (Get-Command ollama -ErrorAction SilentlyContinue) {
    $ollamaFound = $true
    $ollamaHow = "command"
}

# Method 2: Check if Ollama process is running
if (-not $ollamaFound) {
    $ollamaProcess = Get-Process -Name "ollama*" -ErrorAction SilentlyContinue
    if ($ollamaProcess) {
        $ollamaFound = $true
        $ollamaHow = "process"
    }
}

# Method 3: Check if Ollama API is responding
if (-not $ollamaFound) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:11434/api/tags" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $ollamaFound = $true
            $ollamaHow = "api"
        }
    } catch {
        # API not responding
    }
}

if ($ollamaFound) {
    Write-Host "  ${Bold}[OK]${Reset} Ollama is installed and running"
    Write-Host ""
    Write-Host "  ${Bold}SAME is ready to use!${Reset}"
} else {
    Write-Host "  ${Bold}[!] Ollama is not installed yet${Reset}"
    Write-Host ""
    Write-Host "  SAME needs Ollama to work. It's free and easy:"
    Write-Host ""
    Write-Host "  1. Open: https://ollama.ai"
    Write-Host "     ${Dim}(Ctrl+click the link to open in browser)${Reset}"
    Write-Host ""
    Write-Host "  2. Click 'Download for Windows' and run the installer"
    Write-Host ""
    Write-Host "  3. After install, look for the llama icon in your system tray"
    Write-Host "     ${Dim}(bottom-right corner, may be in hidden icons)${Reset}"
    Write-Host ""
    Write-Host "  Stuck? Join our Discord: https://discord.gg/GZGHtrrKF2"
}

Write-Host ""
Write-Host "-----------------------------------------------------------"
Write-Host ""
Write-Host "  ${Bold}WHAT'S NEXT?${Reset}"
Write-Host ""
Write-Host "  1. Navigate to your project folder:"
Write-Host "     ${Dim}cd C:\Users\YourName\Documents\my-project${Reset}"
Write-Host ""
Write-Host "  2. Run the setup wizard:"
Write-Host "     ${Bold}same init${Reset}"
Write-Host ""
Write-Host "  ${Dim}You can run 'same init' right now - no need to restart the terminal!${Reset}"
Write-Host ""
Write-Host "  Questions? https://discord.gg/GZGHtrrKF2"
Write-Host ""
