#!/usr/bin/env bash
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
RELEASE_DIR="$SCRIPT_DIR/build/release"
OUTPUT_DIR="$SCRIPT_DIR/build/output"
BUILD_NAME="webdown-$(date +%Y%m%d-%H%M%S)"

echo "=== WebDown Build Script ==="
echo "Working directory: $SCRIPT_DIR"
echo "Release directory: $RELEASE_DIR"
echo "Output directory: $OUTPUT_DIR"
echo ""

echo "[1/4] Building Go binary..."
cd "$SCRIPT_DIR"
go build -o "$RELEASE_DIR/webdown" ./cmd/webdown
echo "✓ Binary built successfully"

echo "[2/4] Preparing release files..."
mkdir -p "$RELEASE_DIR/config"
mkdir -p "$RELEASE_DIR/web/static"
mkdir -p "$RELEASE_DIR/web/templates"
mkdir -p "$RELEASE_DIR/data/uploads"

if [ ! -f "$RELEASE_DIR/config/config.json" ]; then
    cp "$SCRIPT_DIR/config/config.json" "$RELEASE_DIR/config/config.json"
fi
cp "$SCRIPT_DIR/config/config.json" "$RELEASE_DIR/config/config.default.json"
cp -rf "$SCRIPT_DIR/web/static/"* "$RELEASE_DIR/web/static/"
cp -rf "$SCRIPT_DIR/web/templates/"* "$RELEASE_DIR/web/templates/"
cp "$SCRIPT_DIR/build/run.sh" "$RELEASE_DIR/run.sh" 2>/dev/null || true
cp "$SCRIPT_DIR/build/run.bat" "$RELEASE_DIR/run.bat" 2>/dev/null || true
echo "✓ Release files prepared"

echo "[3/4] Creating archive..."
mkdir -p "$OUTPUT_DIR"
cd "$RELEASE_DIR"
tar -czf "$OUTPUT_DIR/$BUILD_NAME.tar.gz" .
echo "✓ Archive created: $BUILD_NAME.tar.gz"

echo "[4/4] Verifying..."
ls -lh "$OUTPUT_DIR/"
echo ""
echo "=== Build Complete ==="
echo "Release directory: $RELEASE_DIR"
echo "Archive file: $OUTPUT_DIR/$BUILD_NAME.tar.gz"
echo ""
echo "To deploy:"
echo "  1. Copy $OUTPUT_DIR/$BUILD_NAME.tar.gz to your server"
echo "  2. Extract: tar -xzf $BUILD_NAME.tar.gz"
echo "  3. Run: ./run.sh"
echo ""
echo "Remember to update config/config.json with your server's public URL"