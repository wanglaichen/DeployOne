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
    Write-Host "If the port is already in use, the server is already running or an old process must be stopped first."
    Read-Host "Press Enter to close this window"
  }
}

$ScriptDir = $PSScriptRoot
$ExePath = Join-Path $ScriptDir "webdown.exe"
$SourceMarker = Join-Path $ScriptDir "cmd\webdown"

if (Test-Path $ExePath) {
  $Root = $ScriptDir
  $Mode = "release"
}
elseif (Test-Path $SourceMarker) {
  $Root = Split-Path -Parent $ScriptDir
  $Mode = "source"
}
else {
  Write-Host "Error: This script must be placed in either:" -ForegroundColor Red
  Write-Host "  - The source code root (containing 'build', 'cmd', 'web' folders)" -ForegroundColor Red
  Write-Host "  - A release directory (containing 'webdown.exe')" -ForegroundColor Red
  if (-not $NoPause) { Read-Host "Press Enter to close this window" }
  exit 1
}

if ([string]::IsNullOrWhiteSpace($ConfigPath)) {
  $ConfigPath = Join-Path $Root "config\config.json"
}
elseif (-not [System.IO.Path]::IsPathRooted($ConfigPath)) {
  $ConfigPath = Join-Path $Root $ConfigPath
}

if (-not (Test-Path $ConfigPath)) {
  Write-Host "Error: config file not found: $ConfigPath" -ForegroundColor Red
  Pause-OnError -ExitCode 1
  exit 1
}

Push-Location $Root
try {
  switch ($Mode) {
    "release" {
      & $ExePath -config $ConfigPath
      $exitCode = $LASTEXITCODE
    }
    "source" {
      if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Error: 'go' command not found in PATH" -ForegroundColor Red
        Pause-OnError -ExitCode 1
        exit 1
      }
      go run .\cmd\webdown -config $ConfigPath
      $exitCode = $LASTEXITCODE
    }
  }

  if ($exitCode -ne 0) {
    Pause-OnError -ExitCode $exitCode
  }
  exit $exitCode
}
finally {
  Pop-Location
}