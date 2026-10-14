#!/bin/bash
# Build cookie-router for multiple platforms
# Usage: ./scripts/build.sh

set -e

GO_BIN="${GO_BIN:-go}"
VERSION="1.0.0"
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"
OUT_DIR="dist"

mkdir -p "$OUT_DIR"

echo "=== Building Cookie Router v${VERSION} ==="
echo ""

# 1. Linux ARM64 (primary target: STB Armbian)
echo "[1/3] Building linux/arm64 (STB Armbian)..."
GOOS=linux GOARCH=arm64 "$GO_BIN" build -ldflags "$LDFLAGS" -o "${OUT_DIR}/cookie-router-linux-arm64" ./cmd/server/
echo "      -> ${OUT_DIR}/cookie-router-linux-arm64"

# 2. Linux AMD64 (VPS / x86 server)
echo "[2/3] Building linux/amd64..."
GOOS=linux GOARCH=amd64 "$GO_BIN" build -ldflags "$LDFLAGS" -o "${OUT_DIR}/cookie-router-linux-amd64" ./cmd/server/
echo "      -> ${OUT_DIR}/cookie-router-linux-amd64"

# 3. Windows AMD64 (local dev)
echo "[3/3] Building windows/amd64..."
GOOS=windows GOARCH=amd64 "$GO_BIN" build -ldflags "$LDFLAGS" -o "${OUT_DIR}/cookie-router-windows-amd64.exe" ./cmd/server/
echo "      -> ${OUT_DIR}/cookie-router-windows-amd64.exe"

echo ""
echo "=== All builds complete ==="
ls -lh "$OUT_DIR"/
