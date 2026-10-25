<#
.SYNOPSIS
  Installs, checks and uninstalls an MSI silently (used by CI, needs administrator rights).
#>
[CmdletBinding()]
param([Parameter(Mandatory)] [string] $Msi)

$ErrorActionPreference = 'Stop'
$msiPath = (Resolve-Path $Msi).Path
$dir = Join-Path $env:ProgramFiles 'PC Diag'
$config = Join-Path $env:TEMP 'pcdiag-test.json'
'{ "support": { "name": "CI" } }' | Set-Content -Path $config -Encoding utf8

function Invoke-Msiexec([string[]] $arguments) {
    $p = Start-Process msiexec.exe -ArgumentList $arguments -Wait -PassThru
    if ($p.ExitCode -notin 0, 3010) { throw "msiexec $($arguments -join ' ') exited with $($p.ExitCode)" }
}

Invoke-Msiexec @('/i', "`"$msiPath`"", '/qn', '/norestart', "CONFIGFILE=`"$config`"", 'DESKTOPSHORTCUT=1', '/l*v', 'install.log')
foreach ($f in 'pcdiag.exe', 'pcdiag.json', 'pcdiag.example.json', 'LICENSE.txt') {
    if (-not (Test-Path (Join-Path $dir $f))) { throw "missing $f after install" }
}
if (-not (Test-Path (Join-Path $env:ProgramData 'Microsoft\Windows\Start Menu\Programs\PC Diag.lnk'))) { throw 'missing Start menu shortcut' }
if (-not (Test-Path (Join-Path $env:PUBLIC 'Desktop\PC Diag.lnk'))) { throw 'missing desktop shortcut' }
& (Join-Path $dir 'pcdiag.exe') -version | Out-Host

Invoke-Msiexec @('/x', "`"$msiPath`"", '/qn', '/norestart', '/l*v', 'uninstall.log')
if (Test-Path $dir) { throw "$dir still exists after uninstall" }
Write-Host 'MSI install / uninstall test passed'
