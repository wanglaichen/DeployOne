#!/usr/bin/env bash
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
EXE_PATH="$SCRIPT_DIR/webdown"
SOURCE_MARKER="$SCRIPT_DIR/cmd/webdown"

if [ -f "$EXE_PATH" ]; then
  ROOT="$SCRIPT_DIR"
  MODE="release"
elif [ -d "$SOURCE_MARKER" ]; then
  ROOT="$(dirname "$SCRIPT_DIR")"
  MODE="source"
else
  echo "Error: This script must be placed in either:" >&2
  echo "  - The source code root (containing 'build', 'cmd', 'web' folders)" >&2
  echo "  - A release directory (containing 'webdown' binary)" >&2
  exit 1
fi

CONFIG_PATH="$ROOT/config/config.json"

if [ ! -f "$CONFIG_PATH" ]; then
  echo "Error: config file not found: $CONFIG_PATH" >&2
  exit 1
fi

case "$MODE" in
  release)
    exec "$EXE_PATH" -config "$CONFIG_PATH"
    ;;
  source)
    if ! command -v go >/dev/null 2>&1; then
      echo "Error: 'go' command not found in PATH" >&2
      exit 1
    fi
    cd "$ROOT"
    exec go run ./cmd/webdown -config "$CONFIG_PATH"
    ;;
esac