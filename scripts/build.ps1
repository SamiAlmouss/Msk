[CmdletBinding()]
param(
    [string]$Version = 'dev',
    [string]$Commit = 'unknown',
    [string]$Date = ([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')),
    [string]$Go = 'go'
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^(dev|v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?)$' -or $Commit -notmatch '^[A-Za-z0-9._-]+$' -or $Date -notmatch '^[0-9TZ:.-]+$') { throw 'Invalid build metadata.' }
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$oldOS, $oldArch, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
Push-Location $root
try {
    [IO.Directory]::CreateDirectory($dist) | Out-Null
    $checksums = @()
    foreach ($arch in @('amd64', 'arm64')) {
        $stage = Join-Path $dist "windows_$arch"
        [IO.Directory]::CreateDirectory($stage) | Out-Null
        $env:GOOS = 'windows'; $env:GOARCH = $arch; $env:CGO_ENABLED = '0'
        $flags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.date=$Date"
        & $Go build -trimpath -ldflags $flags -o (Join-Path $stage 'msk.exe') ./cmd/msk
        if ($LASTEXITCODE -ne 0) { throw "Go build failed for $arch." }
        Copy-Item -LiteralPath (Join-Path $root 'LICENSE'), (Join-Path $root 'README.md') -Destination $stage -Force
        $asset = "msk_${Version}_windows_${arch}.zip"
        Compress-Archive -Path (Join-Path $stage 'msk.exe'), (Join-Path $stage 'LICENSE'), (Join-Path $stage 'README.md') -DestinationPath (Join-Path $dist $asset) -Force
        $hash = (Get-FileHash -LiteralPath (Join-Path $dist $asset) -Algorithm SHA256).Hash.ToLowerInvariant()
        $checksums += "$hash  $asset"
    }
    [IO.File]::WriteAllLines((Join-Path $dist 'checksums.txt'), $checksums, (New-Object Text.UTF8Encoding($false)))
    Write-Host "Built Windows amd64 and arm64 assets in $dist"
} finally {
    $env:GOOS = $oldOS; $env:GOARCH = $oldArch; $env:CGO_ENABLED = $oldCGO
    Pop-Location
}
