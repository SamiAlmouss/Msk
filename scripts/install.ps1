$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\msk\bin"
$ExePath = Join-Path $InstallDir "msk.exe"
$DownloadUrl = "https://github.com/SamiAlmouss/Msk/releases/latest/download/msk.exe"

try {
    Write-Host "Installing msk..." -ForegroundColor Cyan

    # Create installation directory
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

    # Download latest release
    Invoke-WebRequest `
        -Uri $DownloadUrl `
        -OutFile $ExePath `
        -UseBasicParsing

    # Get current User PATH
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

    # Normalize PATH entries
    $PathEntries = @()

    if (-not [string]::IsNullOrWhiteSpace($UserPath)) {
        $PathEntries = $UserPath -split ";" |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }

    # Add install directory to User PATH if missing
    if ($PathEntries -notcontains $InstallDir) {

        $NewPath = if ([string]::IsNullOrWhiteSpace($UserPath)) {
            $InstallDir
        }
        else {
            "$UserPath;$InstallDir"
        }

        [Environment]::SetEnvironmentVariable(
            "Path",
            $NewPath,
            "User"
        )
    }

    # Add to current PowerShell session
    if (($env:Path -split ";") -notcontains $InstallDir) {
        $env:Path += ";$InstallDir"
    }

    # Verify executable exists
    if (-not (Test-Path $ExePath)) {
        throw "msk.exe was not found after installation."
    }

    Clear-Host
    Write-Host ""

    # ============================================================
    # MSK Logo
    # M = Green
    # S = Cyan
    # K = Red
    # ============================================================

    # Line 1
    Write-Host "╭─╮     ╭─╮ " -NoNewline -ForegroundColor Green
    Write-Host "╭───────╮ "   -NoNewline -ForegroundColor Cyan
    Write-Host "╭─╮ ╭───╮"    -ForegroundColor Red

    # Line 2
    Write-Host "│ ╰─╮ ╭─╯ │ " -NoNewline -ForegroundColor Green
    Write-Host "│ ╭─────╯ "   -NoNewline -ForegroundColor Cyan
    Write-Host "│ ╰─╯ ╭─╯"    -ForegroundColor Red

    # Line 3
    Write-Host "│   ╰─╯   │ " -NoNewline -ForegroundColor Green
    Write-Host "│ ╰─────╮ "   -NoNewline -ForegroundColor Cyan
    Write-Host "│   ╭─╯"      -ForegroundColor Red

    # Line 4
    Write-Host "│ ╭─────╮ │ " -NoNewline -ForegroundColor Green
    Write-Host "╰─────╮ │ "   -NoNewline -ForegroundColor Cyan
    Write-Host "│   ╰─╮"      -ForegroundColor Red

    # Line 5
    Write-Host "│ │     │ │ " -NoNewline -ForegroundColor Green
    Write-Host "╭─────╯ │ "   -NoNewline -ForegroundColor Cyan
    Write-Host "│ ╭─╮ ╰─╮"    -ForegroundColor Red

    # Line 6
    Write-Host "╰─╯     ╰─╯ " -NoNewline -ForegroundColor Green
    Write-Host "╰───────╯ "   -NoNewline -ForegroundColor Cyan
    Write-Host "╰─╯ ╰───╯"    -ForegroundColor Red

    Write-Host ""
    Write-Host "msk installed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installed to:" -ForegroundColor DarkGray
    Write-Host "  $ExePath" -ForegroundColor White
    Write-Host ""
    Write-Host "Try:" -ForegroundColor DarkGray
    Write-Host "  msk --help" -ForegroundColor Yellow
    Write-Host ""

}
catch {
    Write-Host ""
    Write-Host "Installation failed!" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""
    exit 1
}