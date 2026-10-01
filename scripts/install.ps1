$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\msk\bin"
$ExePath = Join-Path $InstallDir "msk.exe"
$DownloadUrl = "https://github.com/SamiAlmouss/Msk/releases/latest/download/msk.exe"

Write-Host "Installing msk..."

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Invoke-WebRequest `
    -Uri $DownloadUrl `
    -OutFile $ExePath

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

if (($UserPath -split ";") -notcontains $InstallDir) {
    $NewPath = if ([string]::IsNullOrWhiteSpace($UserPath)) {
        $InstallDir
    } else {
        "$UserPath;$InstallDir"
    }

    [Environment]::SetEnvironmentVariable(
        "Path",
        $NewPath,
        "User"
    )
}

if (($env:Path -split ";") -notcontains $InstallDir) {
    $env:Path += ";$InstallDir"
}

Write-Host "msk installed successfully."
Write-Host "Installed to: $ExePath"