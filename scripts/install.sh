#!/bin/bash
# Install cookie-router on STB Armbian
# Usage: bash install.sh

set -e

INSTALL_DIR="/opt/cookie-router"
BINARY="cookie-router-linux-arm64"
SERVICE_FILE="cookie-router.service"

echo "=== Installing Cookie Router ==="

# Create install directory
mkdir -p "$INSTALL_DIR"

# Copy binary
if [ -f "$BINARY" ]; then
    cp "$BINARY" "$INSTALL_DIR/cookie-router"
    chmod +x "$INSTALL_DIR/cookie-router"
    echo "[OK] Binary installed to $INSTALL_DIR/cookie-router"
else
    echo "[ERROR] Binary '$BINARY' not found. Make sure it's in the same directory."
    exit 1
fi

# Create default config if not exists
if [ ! -f "$INSTALL_DIR/config.json" ]; then
    cat > "$INSTALL_DIR/config.json" << 'EOF'
{
  "port": 8080,
  "authToken": "",
  "accounts": []
}
EOF
    echo "[OK] Default config created at $INSTALL_DIR/config.json"
else
    echo "[OK] Existing config preserved at $INSTALL_DIR/config.json"
fi

# Install systemd service
if [ -f "$SERVICE_FILE" ]; then
    cp "$SERVICE_FILE" /etc/systemd/system/cookie-router.service
    systemctl daemon-reload
    systemctl enable cookie-router
    echo "[OK] Systemd service installed and enabled"
else
    echo "[WARN] Service file not found, skipping systemd setup"
fi

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Commands:"
echo "  Start:   systemctl start cookie-router"
echo "  Stop:    systemctl stop cookie-router"
echo "  Status:  systemctl status cookie-router"
echo "  Logs:    journalctl -u cookie-router -f"
echo ""
echo "Dashboard: http://<STB-IP>:8080/dashboard"
echo "API:       http://<STB-IP>:8080/v1/chat/completions"
echo ""
echo "Or run manually: $INSTALL_DIR/cookie-router -port 8080"
