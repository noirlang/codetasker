# ==============================================================================
# CodeTasker CLI — Windows Installation Script (PowerShell)
# ==============================================================================

$ErrorActionPreference = "Stop"

Write-Host "
         ⣶  ⣠⠖⠉⠙⢶⢰⠏⢹⡏⢳
        ⢠⡏ ⣼⠃   ⠘  ⢸⡇⠈
   ⣠⣾⠃  ⣼⠃⢸⡇       ⢸⡇ ⠘⢷⣄
 ⣠⡾⠋   ⢀⡟ ⢸⠃       ⢸⡇   ⠙⢷⣄
⣾⡋     ⢸⠇ ⢸⡆       ⢸⡇     ⢙⣷
⠈⠻⣦⡀   ⣿  ⠘⣧       ⢸⡇   ⢀⣴⠟⠉
  ⠈⠻⣦⡀⢰⡇   ⠹⣆   ⢀⡇ ⢸⡇ ⢀⣴⠟⠁
    ⠈⠁⢼⠁    ⠈⠓⠒⠒⠉⠁ ⠛⠃ ⠈⠁
" -ForegroundColor White

Write-Host "Building and installing CodeTasker CLI for Windows..." -ForegroundColor White

# Check Go
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH." -ForegroundColor Red
    Write-Host "Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$BackendDir = Join-Path $RepoRoot "backend"

$InstallDir = Join-Path $env:LOCALAPPDATA "CodeTasker\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$TargetBin = Join-Path $InstallDir "codetasker.exe"

Write-Host "==> Compiling binary from source..." -ForegroundColor Yellow
Push-Location $BackendDir
try {
    go build -ldflags="-s -w" -o $TargetBin ./cmd/codetasker
} finally {
    Pop-Location
}

# Update User PATH if needed
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "==> Adding $InstallDir to User PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

Write-Host ""
Write-Host "✓ CodeTasker CLI successfully installed to: $TargetBin" -ForegroundColor White
Write-Host ""
Write-Host "To verify and get started, run:"
Write-Host "  codetasker --help   (View all available commands)" -ForegroundColor White
Write-Host "  codetasker scan .   (Scan local directory for TODO/FIXME)" -ForegroundColor White
Write-Host "  codetasker repo list (List connected repositories)" -ForegroundColor White
