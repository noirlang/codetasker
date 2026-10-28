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
    Write-Host "Please install Go 1.22+ from https://go.dev/dl/ to build CodeTasker CLI."
    exit 1
}

$BackendDir = $null
$TempDir = $null

try {
    # 1. Check if run from a local script file
    if ($MyInvocation.MyCommand.Path) {
        $ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
        $LocalCandidate = Join-Path (Split-Path -Parent $ScriptDir) "backend"
        if (Test-Path (Join-Path $LocalCandidate "cmd\codetasker\main.go")) {
            $BackendDir = $LocalCandidate
        }
    }

    # 2. Check if run from inside an existing cloned repository
    if (-not $BackendDir -and (Test-Path "backend\cmd\codetasker\main.go")) {
        $BackendDir = (Resolve-Path "backend").Path
    }

    # 3. Remote execution (irm | iex): clone or download into temp
    if (-not $BackendDir) {
        Write-Host "==> Fetching CodeTasker repository..." -ForegroundColor Yellow
        $TempBase = [System.IO.Path]::GetTempPath()
        $TempDir = Join-Path $TempBase ("codetasker-src-" + [System.Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

        if (Get-Command "git" -ErrorAction SilentlyContinue) {
            git clone --depth 1 https://github.com/noirlang/codetasker.git $TempDir --quiet
            $BackendDir = Join-Path $TempDir "backend"
        } else {
            $ZipPath = Join-Path $TempDir "codetasker.zip"
            Invoke-WebRequest -Uri "https://github.com/noirlang/codetasker/archive/refs/heads/main.zip" -OutFile $ZipPath
            Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force
            $Extracted = Join-Path $TempDir "codetasker-main\backend"
            if (Test-Path (Join-Path $Extracted "cmd\codetasker\main.go")) {
                $BackendDir = $Extracted
            } else {
                $BackendDir = Join-Path $TempDir "backend"
            }
        }
    }

    if (-not (Test-Path (Join-Path $BackendDir "cmd\codetasker\main.go"))) {
        Write-Host "Error: Could not find CodeTasker source at $BackendDir" -ForegroundColor Red
        exit 1
    }

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
    Write-Host "✓ CodeTasker CLI successfully installed to: $TargetBin" -ForegroundColor Green
    Write-Host ""
    Write-Host "To verify and get started, run:"
    Write-Host "  codetasker --help   (View all available commands)" -ForegroundColor White
    Write-Host "  codetasker scan .   (Scan local directory for TODO/FIXME)" -ForegroundColor White
    Write-Host "  codetasker repo list (List connected repositories)" -ForegroundColor White

} finally {
    if ($TempDir -and (Test-Path $TempDir)) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
