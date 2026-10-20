#!/usr/bin/env bash
# ==============================================================================
#  Cookie Router - Linux STB (Armbian / Debian / Ubuntu) 1-Click Installer
# ==============================================================================
set -e

INSTALL_DIR="/opt/cookie-router"
SERVICE_FILE="/etc/systemd/system/cookie-router.service"

echo "========================================================"
echo "  🍪 Installing Cookie Router on Linux STB..."
echo "========================================================"

# 1. Check Root
if [ "$EUID" -ne 0 ]; then
  echo "[-] Mohon jalankan script ini sebagai root: sudo bash deploy-stb.sh"
  exit 1
fi

# 2. Detect Architecture
ARCH=$(uname -m)
BINARY_SOURCE=""
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
  echo "[+] Terdeteksi arsitektur ARM64 / aarch64 (STB Armbian)"
  BINARY_SOURCE="cookie-router-linux-arm64"
elif [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ]; then
  echo "[+] Terdeteksi arsitektur x86_64 / amd64"
  BINARY_SOURCE="cookie-router-linux-amd64"
else
  echo "[-] Arsitektur tidak didukung: $ARCH"
  exit 1
fi

# 3. Create target directory
mkdir -p "$INSTALL_DIR"

# 4. Copy binary
if [ -f "$BINARY_SOURCE" ]; then
  cp "$BINARY_SOURCE" "$INSTALL_DIR/cookie-router"
elif [ -f "dist/$BINARY_SOURCE" ]; then
  cp "dist/$BINARY_SOURCE" "$INSTALL_DIR/cookie-router"
elif [ -f "cookie-router" ]; then
  cp "cookie-router" "$INSTALL_DIR/cookie-router"
else
  echo "[-] File binary $BINARY_SOURCE tidak ditemukan di direktori saat ini."
  exit 1
fi

chmod +x "$INSTALL_DIR/cookie-router"
echo "[+] Binary terpasang di $INSTALL_DIR/cookie-router"

# 5. Setup config.json
if [ ! -f "$INSTALL_DIR/config.json" ]; then
  if [ -f "config.json" ]; then
    cp "config.json" "$INSTALL_DIR/config.json"
    echo "[+] Menyalin config.json yang sudah ada ke $INSTALL_DIR/config.json"
  else
    cat << 'EOF' > "$INSTALL_DIR/config.json"
{
  "port": 18899,
  "authToken": "",
  "accounts": []
}
EOF
    echo "[+] Membuat template config.json baru di $INSTALL_DIR/config.json"
  fi
  chmod 600 "$INSTALL_DIR/config.json"
fi

# 6. Install Systemd Service
cat << EOF > "$SERVICE_FILE"
[Unit]
Description=Cookie Router - Lightweight Web Cookie LLM Gateway
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/cookie-router -config $INSTALL_DIR/config.json
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment=PORT=18899

[Install]
WantedBy=multi-user.target
EOF

# 7. Reload & Enable Systemd
systemctl daemon-reload
systemctl enable cookie-router
systemctl restart cookie-router

# 8. Check Status & Print IP
sleep 2
if systemctl is-active --quiet cookie-router; then
  LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "IP_STB_ANDA")
  echo ""
  echo "========================================================"
  echo "  ✅ Cookie Router BERHASIL AKTIF DI LINUX STB!"
  echo "========================================================"
  echo "  📊 Dashboard: http://${LOCAL_IP}:18899"
  echo "  🔌 API Chat:  http://${LOCAL_IP}:18899/v1/chat/completions"
  echo "  ⚙️ Service:   systemctl status cookie-router"
  echo "  📝 Logs:      journalctl -u cookie-router -f"
  echo "========================================================"
else
  echo "[-] Layanan gagal berjalan. Cek log dengan: journalctl -u cookie-router -xe"
fi
