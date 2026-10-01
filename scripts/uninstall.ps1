[CmdletBinding()]
param([switch]$NoPathUpdate)
$ErrorActionPreference = 'Stop'
if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) { throw 'This script supports Windows only.' }
if (-not $env:LOCALAPPDATA) { throw 'LOCALAPPDATA is unavailable.' }
$bin = Join-Path $env:LOCALAPPDATA 'msk\bin'
$target = Join-Path $bin 'msk.exe'
function Normalize-MskPath([string]$Value) {
    if (-not $Value) { return '' }
    try { return [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($Value.Trim().Trim('"'))).TrimEnd('\', '/').ToUpperInvariant() } catch { return $Value.Trim().ToUpperInvariant() }
}
function Remove-MskPath([string]$Current) {
    $normal = Normalize-MskPath $bin
    return (($Current -split ';' | Where-Object { (Normalize-MskPath $_) -ne $normal }) -join ';')
}
if (Test-Path -LiteralPath $target) {
    try { Remove-Item -LiteralPath $target -Force } catch { throw "Cannot remove msk.exe; close running msk processes. $($_.Exception.Message)" }
}
# Only remove empty installation directories; never recursively delete user content.
if ((Test-Path -LiteralPath $bin) -and @(Get-ChildItem -LiteralPath $bin -Force).Count -eq 0) { [IO.Directory]::Delete($bin) }
$parent = Split-Path -Parent $bin
if ((Test-Path -LiteralPath $parent) -and @(Get-ChildItem -LiteralPath $parent -Force).Count -eq 0) { [IO.Directory]::Delete($parent) }
if (-not $NoPathUpdate) {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $updated = Remove-MskPath $userPath
    if ($updated -cne $userPath) { [Environment]::SetEnvironmentVariable('Path', $updated, 'User') }
    $env:Path = Remove-MskPath $env:Path
}
Write-Host 'Uninstalled msk. Other files and PATH entries were preserved.'
