param(
  [string]$ConfigPath = "",
  [switch]$NoPause
)

$ErrorActionPreference = "Stop"

function Pause-OnError {
  param([int]$ExitCode)
  if (-not $NoPause) {
    Write-Host ""
    Write-Host "Program exited. Exit code: $ExitCode"
    Read-Host "Press Enter to close this window"
  }
}

$ScriptDir = $PSScriptRoot
$ExePath = Join-Path $ScriptDir "webdown.exe"
if (-not (Test-Path $ExePath)) {
  Write-Host "Error: webdown.exe not found in $ScriptDir" -ForegroundColor Red
  Pause-OnError -ExitCode 1
  exit 1
}

if ([string]::IsNullOrWhiteSpace($ConfigPath)) {
  $ConfigPath = Join-Path $ScriptDir "config\config.json"
} elseif (-not [System.IO.Path]::IsPathRooted($ConfigPath)) {
  $ConfigPath = Join-Path $ScriptDir $ConfigPath
}

if (-not (Test-Path $ConfigPath)) {
  Write-Host "Error: config file not found: $ConfigPath" -ForegroundColor Red
  Pause-OnError -ExitCode 1
  exit 1
}

Push-Location $ScriptDir
try {
  & $ExePath -config $ConfigPath
  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0) { Pause-OnError -ExitCode $exitCode }
  exit $exitCode
} finally {
  Pop-Location
}