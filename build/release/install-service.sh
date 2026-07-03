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