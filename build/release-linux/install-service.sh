#!/usr/bin/env bash
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
SERVICE_NAME="webdown"
INSTALL_DIR="/opt/webdown"

if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo ./install-service.sh)"
    exit 1
fi

echo "=== WebDown Service Installer ==="
echo "Install directory: $INSTALL_DIR"
echo "Service name: $SERVICE_NAME"
echo ""

echo "[1/5] Stopping existing service..."
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
echo "✓ Stopped"

echo "[2/5] Copying files to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
cp -f "$SCRIPT_DIR/webdown" "$INSTALL_DIR/webdown"
cp -rf "$SCRIPT_DIR/config" "$INSTALL_DIR/"
cp -rf "$SCRIPT_DIR/web" "$INSTALL_DIR/"
cp -f "$SCRIPT_DIR/run.sh" "$INSTALL_DIR/run.sh"
cp -f "$SCRIPT_DIR/stop.sh" "$INSTALL_DIR/stop.sh"
mkdir -p "$INSTALL_DIR/data/uploads"

chmod +x "$INSTALL_DIR/webdown"
chmod +x "$INSTALL_DIR/run.sh"
chmod +x "$INSTALL_DIR/stop.sh"
echo "✓ Files copied"

echo "[3/5] Setting up log directory..."
mkdir -p /var/log/webdown
echo "✓ Log directory ready"

echo "[4/5] Installing systemd service..."
cp -f "$SCRIPT_DIR/webdown.service" "/etc/systemd/system/$SERVICE_NAME.service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
echo "✓ Service installed"

echo "[5/5] Starting service..."
systemctl start "$SERVICE_NAME"
sleep 2

if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✓ Service started successfully"
    echo ""
    echo "=== Installation Complete ==="
    echo "Service status: systemctl status $SERVICE_NAME"
    echo "View logs:      journalctl -u $SERVICE_NAME -f"
    echo "Stop service:   systemctl stop $SERVICE_NAME"
    echo "Start service:  systemctl start $SERVICE_NAME"
    echo ""
    echo "Access: http://localhost:8080"
else
    echo "✗ Service failed to start. Check: journalctl -u $SERVICE_NAME"
    exit 1
fi