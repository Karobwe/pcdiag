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

if (-not (Get-Command wix -ErrorAction SilentlyContinue)) {
    dotnet tool install --global wix --version $WixVersion
    $env:PATH += [IO.Path]::PathSeparator + (Join-Path $HOME '.dotnet\tools')
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null

wix build (Join-Path $PSScriptRoot 'pcdiag.wxs') `
    -arch x64 `
    -d "Version=$Version" `
    -d "ExeFile=$((Resolve-Path $Exe).Path)" `
    -d "IconFile=$(Join-Path $root 'assets\icon.ico')" `
    -d "SourceDir=$root" `
    -o $Output
if ($LASTEXITCODE -ne 0) { throw "wix build failed with exit code $LASTEXITCODE" }

Write-Host "Built $Output"
