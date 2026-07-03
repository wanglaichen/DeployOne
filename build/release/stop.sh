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