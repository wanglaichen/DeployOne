#!/usr/bin/env bash
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
RELEASE_DIR="$SCRIPT_DIR/build/release-linux"
OUTPUT_DIR="$SCRIPT_DIR/build/output"
BUILD_NAME="webdown-linux-$(date +%Y%m%d-%H%M%S)"

echo "=== WebDown Linux Build ==="
echo "Working directory: $SCRIPT_DIR"
echo "Release directory: $RELEASE_DIR"
echo "Output archive : $OUTPUT_DIR/$BUILD_NAME.tar.gz"
echo ""

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed. Please install Go first." >&2
  exit 1
fi

GO_VERSION=$(go version)
echo "Go version: $GO_VERSION"
if [[ "$GO_VERSION" != *"go1.20."* ]]; then
  echo "Warning: For Windows Server 2008 / RHEL 6 compatibility, Go 1.20.x is recommended."
  echo "Current: $GO_VERSION"
fi
echo ""

echo "[1/6] Cleaning release directory..."
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"
echo "OK"

echo "[2/6] Building Linux binary..."
cd "$SCRIPT_DIR"
GOOS=linux GOARCH=amd64 go build -o "$RELEASE_DIR/webdown" ./cmd/webdown
echo "OK (linux/amd64)"

echo "[3/6] Preparing release files..."
mkdir -p "$RELEASE_DIR/config"
mkdir -p "$RELEASE_DIR/web/static"
mkdir -p "$RELEASE_DIR/web/templates"
mkdir -p "$RELEASE_DIR/data/uploads"

cp "$SCRIPT_DIR/config/config.json" "$RELEASE_DIR/config/config.json"
cp "$SCRIPT_DIR/config/config.json" "$RELEASE_DIR/config/config.default.json"
cp -rf "$SCRIPT_DIR/web/static/"* "$RELEASE_DIR/web/static/"
cp -rf "$SCRIPT_DIR/web/templates/"* "$RELEASE_DIR/web/templates/"

cat > "$RELEASE_DIR/run.sh" <<'RUN_EOF'
#!/usr/bin/env bash
set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
exec "$SCRIPT_DIR/webdown" -config "$SCRIPT_DIR/config/config.json"
RUN_EOF

cat > "$RELEASE_DIR/stop.sh" <<'STOP_EOF'
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
STOP_EOF

cat > "$RELEASE_DIR/webdown.service" <<'SVC_EOF'
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
SVC_EOF

cat > "$RELEASE_DIR/install-service.sh" <<'INS_EOF'
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
INS_EOF

chmod +x "$RELEASE_DIR/webdown"
chmod +x "$RELEASE_DIR/run.sh"
chmod +x "$RELEASE_DIR/stop.sh"
chmod +x "$RELEASE_DIR/install-service.sh"

echo "OK"

echo "[4/6] Verifying binary..."
file "$RELEASE_DIR/webdown" || true
echo "OK"

echo "[5/6] Creating archive..."
mkdir -p "$OUTPUT_DIR"
cd "$RELEASE_DIR"
tar -czf "$OUTPUT_DIR/$BUILD_NAME.tar.gz" .
echo "OK: $BUILD_NAME.tar.gz"

echo "[6/6] Summary..."
ls -lh "$RELEASE_DIR" | head -20
echo ""
echo "=== Build complete ==="
echo "Linux binary : $RELEASE_DIR/webdown"
echo "Archive      : $OUTPUT_DIR/$BUILD_NAME.tar.gz"
echo ""
echo "Deploy on Linux server:"
echo "  1. Copy $OUTPUT_DIR/$BUILD_NAME.tar.gz to your server"
echo "  2. tar -xzf $BUILD_NAME.tar.gz -C /opt/webdown"
echo "  3. vim /opt/webdown/config/config.json   # set public_base_url"
echo "  4. cd /opt/webdown && ./run.sh"
echo ""
echo "Or install as systemd service:"
echo "  sudo ./install-service.sh"
