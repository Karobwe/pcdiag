<#
.SYNOPSIS
  Builds the PC Diag MSI package with the WiX Toolset.

.EXAMPLE
  ./packaging/windows/build-msi.ps1 -Exe .\pcdiag.exe -Version 1.2.3 -Output .\dist\pcdiag_1.2.3_windows_amd64.msi

.NOTES
  Requires the .NET SDK. The WiX command-line tool is installed as a .NET global tool if missing.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Exe,
    # Windows Installer only accepts numeric versions (major.minor.build).
    [Parameter(Mandatory)] [ValidatePattern('^\d+\.\d+\.\d+$')] [string] $Version,
    [Parameter(Mandatory)] [string] $Output
)

$ErrorActionPreference = 'Stop'
$WixVersion = '5.0.2'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

# Report errors as GitHub Actions annotations when running in CI.
function Fail([string] $message, [string[]] $details) {
    if ($env:GITHUB_ACTIONS) {
        foreach ($line in $details) { if ($line -match 'error|warning|exception') { Write-Host "::error::$line" } }
    }
    throw $message
}

if (-not (Get-Command wix -ErrorAction SilentlyContinue)) {
    $out = dotnet tool install --global wix --version $WixVersion 2>&1 | ForEach-Object { "$_" }
    $out | Out-Host
    if ($LASTEXITCODE -ne 0) { Fail 'Unable to install the WiX tool' $out }
    $env:PATH += [IO.Path]::PathSeparator + (Join-Path $HOME '.dotnet\tools')
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null
# Build in a temporary folder: WiX also writes a .wixpdb debug file that must not be distributed.
$work = Join-Path ([IO.Path]::GetTempPath()) ("pcdiag-msi-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $work | Out-Null
$msi = Join-Path $work 'pcdiag.msi'

$out = wix build (Join-Path $PSScriptRoot 'pcdiag.wxs') `
    -arch x64 `
    -d "Version=$Version" `
    -d "ExeFile=$((Resolve-Path $Exe).Path)" `
    -d "IconFile=$(Join-Path $root 'assets\icon.ico')" `
    -d "SourceDir=$root" `
    -o $msi 2>&1 | ForEach-Object { "$_" }
$out | Out-Host
if ($LASTEXITCODE -ne 0) { Fail "wix build failed with exit code $LASTEXITCODE" $out }

Move-Item -Force $msi $Output
Remove-Item -Recurse -Force $work
Write-Host "Built $Output"
