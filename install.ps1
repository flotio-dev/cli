# Flotio CLI installer for Windows
# Run in PowerShell:
#   irm https://raw.githubusercontent.com/flotio-dev/cli/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "flotio-dev/cli"
$Binary = "flotio"

# --- Detect architecture ---
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $GoArch = "amd64" }
    "ARM64" { $GoArch = "arm64" }
    default {
        Write-Error "Unsupported architecture: $Arch"
        exit 1
    }
}

# --- Download latest release ---
$Url = "https://github.com/${Repo}/releases/latest/download/flotio-windows-${GoArch}.exe"
Write-Host "Downloading $Url..."

$TempDir = Join-Path $env:TEMP "flotio-install"
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null
$Dest = Join-Path $TempDir "flotio.exe"

try {
    Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing
} catch {
    Write-Error "Failed to download. Make sure a release exists (https://github.com/${Repo}/releases)"
    exit 1
}

# --- Install ---
$InstallDir = Join-Path $env:LOCALAPPDATA "flotio\bin"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$Target = Join-Path $InstallDir "flotio.exe"
Move-Item -Force $Dest $Target

Write-Host "✓ flotio installed to $Target"

# --- PATH check ---
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "⚠ $InstallDir is not in your PATH."
    Write-Host "  Run this to add it (new terminal required after):"
    Write-Host "  [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$InstallDir', 'User')"
}

# Cleanup
Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
