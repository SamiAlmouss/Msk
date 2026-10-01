# Compatible with Windows PowerShell 5.1 and PowerShell 7 on Windows.
[CmdletBinding()]
param(
    [string]$GITHUB_OWNER = 'GITHUB_OWNER',
    [string]$GITHUB_REPOSITORY = 'GITHUB_REPOSITORY',
    [string]$Version = 'latest',
    # Offline distribution tests use the same release assets without networking.
    [string]$AssetDirectory,
    [switch]$NoPathUpdate
)
$ErrorActionPreference = 'Stop'
if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) { throw 'This installer supports Windows only.' }
if (-not $env:LOCALAPPDATA) { throw 'LOCALAPPDATA is unavailable.' }
if ($Version -ne 'latest' -and $Version -notmatch '^v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?$') { throw 'Version must be latest or a tag such as v1.0.0.' }
if (-not $AssetDirectory -and ($GITHUB_OWNER -eq 'GITHUB_OWNER' -or $GITHUB_REPOSITORY -eq 'GITHUB_REPOSITORY')) {
    throw 'Set GITHUB_OWNER and GITHUB_REPOSITORY to the published GitHub repository.'
}
function Normalize-MskPath([string]$Value) {
    if (-not $Value) { return '' }
    try { return [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($Value.Trim().Trim('"'))).TrimEnd('\', '/').ToUpperInvariant() } catch { return $Value.Trim().ToUpperInvariant() }
}
function Add-MskPath([string]$Current, [string]$Directory) {
    $normal = Normalize-MskPath $Directory
    foreach ($entry in ($Current -split ';')) { if ((Normalize-MskPath $entry) -eq $normal) { return $Current } }
    if ([string]::IsNullOrEmpty($Current)) { return $Directory }
    if ($Current.EndsWith(';')) { return $Current + $Directory }
    return $Current + ';' + $Directory
}
$nativeArch = $env:PROCESSOR_ARCHITEW6432
if (-not $nativeArch) { $nativeArch = $env:PROCESSOR_ARCHITECTURE }
switch ($nativeArch.ToUpperInvariant()) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "Unsupported Windows architecture: $nativeArch" }
}
$temp = Join-Path ([IO.Path]::GetTempPath()) ('msk-install-' + [Guid]::NewGuid().ToString('N'))
$pending = $null
$backup = $null
try {
    [IO.Directory]::CreateDirectory($temp) | Out-Null
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    $headers = @{ 'User-Agent' = 'msk-installer'; 'Accept' = 'application/vnd.github+json' }
    if ($Version -eq 'latest') {
        if ($AssetDirectory) { throw 'Offline assets require an explicit -Version.' }
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPOSITORY/releases/latest" -Headers $headers
        $Version = $release.tag_name
        if ($Version -notmatch '^v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?$') { throw 'Latest release has an unsupported version tag.' }
    }
    $asset = "msk_${Version}_windows_${arch}.zip"
    foreach ($name in @($asset, 'checksums.txt')) {
        $destination = Join-Path $temp $name
        if ($AssetDirectory) { Copy-Item -LiteralPath (Join-Path $AssetDirectory $name) -Destination $destination }
        else { Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/$GITHUB_OWNER/$GITHUB_REPOSITORY/releases/download/$Version/$name" -Headers $headers -OutFile $destination }
    }
    $matchesFound = @()
    foreach ($line in [IO.File]::ReadAllLines((Join-Path $temp 'checksums.txt'))) {
        if ($line -match '^([a-fA-F0-9]{64})\s+\*?(.+)$' -and $Matches[2] -ceq $asset) { $matchesFound += $Matches[1] }
    }
    if ($matchesFound.Count -ne 1) { throw "Expected exactly one checksum for $asset." }
    $actual = (Get-FileHash -LiteralPath (Join-Path $temp $asset) -Algorithm SHA256).Hash
    if ($actual -ine $matchesFound[0]) { throw 'SHA-256 mismatch; existing installation has not been changed.' }
    # Checksums verify release consistency, not publisher identity or a trusted signature.
    $extract = Join-Path $temp 'extracted'
    Expand-Archive -LiteralPath (Join-Path $temp $asset) -DestinationPath $extract
    $binary = Join-Path $extract 'msk.exe'
    if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) { throw 'Release archive does not contain msk.exe at its root.' }
    $bin = Join-Path $env:LOCALAPPDATA 'msk\bin'
    [IO.Directory]::CreateDirectory($bin) | Out-Null
    $target = Join-Path $bin 'msk.exe'
    $pending = Join-Path $bin ('.msk-update-' + [Guid]::NewGuid().ToString('N') + '.exe')
    Copy-Item -LiteralPath $binary -Destination $pending
    try {
        if ([IO.File]::Exists($target)) {
            $backup = Join-Path $bin ('.msk-backup-' + [Guid]::NewGuid().ToString('N') + '.exe')
            [IO.File]::Replace($pending, $target, $backup)
        }
        else { [IO.File]::Move($pending, $target) }
        $pending = $null
    } catch { throw "Cannot replace msk.exe. Close running msk processes and check directory permissions. $($_.Exception.Message)" }
    if (-not $NoPathUpdate) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $updated = Add-MskPath $userPath $bin
        if ($updated -cne $userPath) { [Environment]::SetEnvironmentVariable('Path', $updated, 'User') }
        $env:Path = Add-MskPath $env:Path $bin
    }
    Write-Host "Installed msk $Version ($arch) in $bin"
    Write-Host 'SHA-256 verified. Checksums are not a trusted code signature.'
} finally {
    if ($pending -and (Test-Path -LiteralPath $pending)) { Remove-Item -LiteralPath $pending -Force }
    if ($backup -and (Test-Path -LiteralPath $backup)) { Remove-Item -LiteralPath $backup -Force }
    # $temp is a unique directory created by this script, within the system temp root.
    if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Recurse -Force }
}
