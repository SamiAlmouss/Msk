# Offline smoke tests: do not modify the real user PATH or installation.
[CmdletBinding()]
param([string]$Version = 'v1.0.0', [string]$Assets = '')
$ErrorActionPreference = 'Stop'
if (-not $Assets) { $Assets = Join-Path (Split-Path -Parent $PSScriptRoot) 'dist' }
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('msk-installer-test-' + [Guid]::NewGuid().ToString('N'))
$originalLocalAppData = $env:LOCALAPPDATA
$originalSessionPath = $env:Path
$originalUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
try {
    [IO.Directory]::CreateDirectory($testRoot) | Out-Null
    $env:LOCALAPPDATA = Join-Path $testRoot 'local'
    . (Join-Path $PSScriptRoot 'install.ps1') -Version $Version -AssetDirectory $Assets -NoPathUpdate
    $installed = Join-Path $env:LOCALAPPDATA 'msk\bin\msk.exe'
    if (-not (Test-Path -LiteralPath $installed)) { throw 'Install did not create executable.' }
    & $installed --version
    if ($LASTEXITCODE -ne 0) { throw 'Installed executable failed.' }
    $before = (Get-FileHash -LiteralPath $installed).Hash
    . (Join-Path $PSScriptRoot 'install.ps1') -Version $Version -AssetDirectory $Assets -NoPathUpdate
    if ((Get-FileHash -LiteralPath $installed).Hash -ne $before) { throw 'Reinstall changed binary unexpectedly.' }
    $locked = [IO.File]::Open($installed, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
    $failed = $false
    try {
        try { & (Join-Path $PSScriptRoot 'install.ps1') -Version $Version -AssetDirectory $Assets -NoPathUpdate } catch { $failed = $true }
    } finally { $locked.Dispose() }
    if (-not $failed -or (Get-FileHash -LiteralPath $installed).Hash -ne $before) { throw 'Locked installation was not safely rejected.' }
    $bin = Split-Path -Parent $installed
    $samplePath = "C:\Other;;$bin;C:\Last"
    if ((Add-MskPath $samplePath $bin) -cne $samplePath) { throw 'PATH deduplication failed.' }
    if ((Add-MskPath 'C:\Other' $bin) -cne "C:\Other;$bin") { throw 'PATH append failed.' }
    $badAssets = Join-Path $testRoot 'bad-assets'
    [IO.Directory]::CreateDirectory($badAssets) | Out-Null
    Copy-Item -Path (Join-Path $Assets '*.zip') -Destination $badAssets
    [IO.File]::WriteAllText((Join-Path $badAssets 'checksums.txt'), 'invalid')
    $failed = $false
    try { & (Join-Path $PSScriptRoot 'install.ps1') -Version $Version -AssetDirectory $badAssets -NoPathUpdate } catch { $failed = $true }
    if (-not $failed -or (Get-FileHash -LiteralPath $installed).Hash -ne $before) { throw 'Bad checksum did not preserve installation.' }
    $corruptAssets = Join-Path $testRoot 'corrupt-assets'
    [IO.Directory]::CreateDirectory($corruptAssets) | Out-Null
    Copy-Item -Path (Join-Path $Assets '*') -Destination $corruptAssets -Recurse
    foreach ($zip in Get-ChildItem -LiteralPath $corruptAssets -Filter '*.zip') { [IO.File]::AppendAllText($zip.FullName, 'corruption') }
    $failed = $false
    try { & (Join-Path $PSScriptRoot 'install.ps1') -Version $Version -AssetDirectory $corruptAssets -NoPathUpdate } catch { $failed = $true }
    if (-not $failed -or (Get-FileHash -LiteralPath $installed).Hash -ne $before) { throw 'Checksum mismatch did not preserve installation.' }
    . (Join-Path $PSScriptRoot 'uninstall.ps1') -NoPathUpdate
    if ((Remove-MskPath $samplePath) -cne 'C:\Other;;C:\Last') { throw 'PATH removal altered other entries.' }
    if (Test-Path -LiteralPath $installed) { throw 'Uninstall left executable.' }
    if ($env:Path -cne $originalSessionPath -or [Environment]::GetEnvironmentVariable('Path', 'User') -cne $originalUserPath) { throw 'Offline tests modified real PATH.' }
    Write-Host 'Installer tests passed: install, reinstall, locked file, checksum rejection, PATH functions, uninstall.'
} finally {
    $env:LOCALAPPDATA = $originalLocalAppData
    # Unique, absolute test directory allocated beneath system temp by this script.
    if (Test-Path -LiteralPath $testRoot) { Remove-Item -LiteralPath $testRoot -Recurse -Force }
}
