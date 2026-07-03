$ErrorActionPreference = "Stop"

$Root = $PSScriptRoot
$ReleaseDir = Join-Path $Root "build\release"
$OutputFile = Join-Path $ReleaseDir "webdown.exe"

New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null

Push-Location $Root
try {
  $ReleaseWebDir = Join-Path $ReleaseDir "web"
  $ReleaseRunScript = Join-Path $ReleaseDir "run.ps1"
  $ReleaseRunBat = Join-Path $ReleaseDir "run.bat"
  $ReleaseRunSh = Join-Path $ReleaseDir "run.sh"
  $ReleaseStopSh = Join-Path $ReleaseDir "stop.sh"
  $ReleaseService = Join-Path $ReleaseDir "webdown.service"
  $ReleaseInstall = Join-Path $ReleaseDir "install-service.sh"
  $ReleaseConfigDir = Join-Path $ReleaseDir "config"
  $ReleaseConfigFile = Join-Path $ReleaseConfigDir "config.json"
  $DefaultConfigFile = Join-Path $ReleaseConfigDir "config.default.json"

  foreach ($p in @($ReleaseWebDir, $ReleaseRunScript, $ReleaseRunBat, $ReleaseRunSh, $ReleaseStopSh, $ReleaseService, $ReleaseInstall)) {
    if (Test-Path $p) { Remove-Item -Recurse -Force $p }
  }

  Get-ChildItem -Path $ReleaseDir -Filter "*.exe~" -File -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
  Get-ChildItem -Path $ReleaseDir -Filter "*.bak" -File -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue

  go build -o $OutputFile .\cmd\webdown
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }

  New-Item -ItemType Directory -Force -Path $ReleaseConfigDir | Out-Null
  if (-not (Test-Path $ReleaseConfigFile)) {
    Copy-Item -Force (Join-Path $Root "config\config.json") $ReleaseConfigFile
  }
  Copy-Item -Force (Join-Path $Root "config\config.json") $DefaultConfigFile
  Copy-Item -Recurse -Force (Join-Path $Root "web") (Join-Path $ReleaseDir "web")
  New-Item -ItemType Directory -Force -Path (Join-Path $ReleaseDir "data\uploads") | Out-Null

  @'
#!/usr/bin/env bash
set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
exec "$SCRIPT_DIR/webdown" -config "$SCRIPT_DIR/config/config.json"
'@ | Set-Content -NoNewline -Encoding ascii $ReleaseRunSh

  @'
#!/usr/bin/env bash
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PID_FILE="$SCRIPT_DIR/webdown.pid"
if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  if ps -p "$PID" > /dev/null 2>&1; then
    echo "Stopping webdown (PID: $PID)..."
    kill "$PID"
    sleep 2
    if ps -p "$PID" > /dev/null 2>&1; then
      echo "Force killing..."
      kill -9 "$PID"
    fi
    echo "Stopped"
  else
    echo "Process not running"
  fi
  rm -f "$PID_FILE"
else
  pkill -f "$SCRIPT_DIR/webdown" 2>/dev/null || true
fi
echo "Done"
'@ | Set-Content -NoNewline -Encoding ascii $ReleaseStopSh

  @'
[Unit]
Description=WebDown APK Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/webdown
ExecStart=/opt/webdown/webdown -config /opt/webdown/config/config.json
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
'@ | Set-Content -NoNewline -Encoding ascii $ReleaseService

  @'
#!/usr/bin/env bash
set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
SERVICE_FILE="/etc/systemd/system/webdown.service"
echo "Installing systemd service to $SERVICE_FILE"
sudo cp "$SCRIPT_DIR/webdown.service" "$SERVICE_FILE"
sudo systemctl daemon-reload
sudo systemctl enable webdown
sudo systemctl start webdown
echo "Service installed. Use: sudo systemctl status webdown"
'@ | Set-Content -NoNewline -Encoding ascii $ReleaseInstall

  $RunPs1 = @'
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
'@
Set-Content -NoNewline -Encoding ascii $ReleaseRunScript -Value $RunPs1

  $RunBat = @"
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0run.ps1" %*
if errorlevel 1 (
  echo.
  echo Startup failed. Check the message above.
  pause
)
"@
Set-Content -NoNewline -Encoding ascii $ReleaseRunBat -Value $RunBat

  Write-Host ""
  Write-Host "Build success: $ReleaseDir"
  Write-Host "Windows files: webdown.exe, run.ps1, run.bat"
  Write-Host "Linux files  : run.sh, stop.sh, install-service.sh, webdown.service"
  Write-Host "Persistent data kept in: $(Join-Path $ReleaseDir "data")"
  Write-Host "Existing config kept at: $ReleaseConfigFile"
}
finally {
  Pop-Location
}
