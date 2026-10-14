# 📸 Cookie Router - Development Snapshot & Checkpoint
**Tanggal Snapshot:** 15 Oktober 2026  
**Versi:** v2.0.0 (Milestone 2 - Multi-Provider & Zero-Disk Multimodal Hub)  
**Port Utama:** `18899` (`http://localhost:18899/`)  

---

## 🎯 Ringkasan Proyek & Status Terakhir

`cookie-router` adalah gateway API mandiri berbasis bahasa Go yang mengubah cookie sesi peramban web (tanpa perlu API key berbayar) menjadi API standar yang kompatibel 100% dengan OpenAI (`/v1/chat/completions`, `/v1/models`, `/v1/images/generations`, `/v1/videos/generations`). Dilengkapi dengan Web Dashboard & Chat Playground interaktif serta sistem instalasi 1-klik untuk TV Box STB Armbian Linux ARM64.

---

## ⚡ Capaian & Perubahan Penting Hari Ini

1. **ChatGPT Web Provider & Sentinel PoW Hardening**:
   - Memperbaiki parser token sesi agar menangkap metadata `account.id` dari `/api/auth/session`.
   - Menginjeksi otomatis header `ChatGPT-Account-Id: <id>` pada setiap request obrolan dan request PoW Sentinel requirements (`backend-api/sentinel/chat-requirements`) untuk akun berstatus ChatGPT Plus.
   - Mengalihkan endpoint percakapan ChatGPT ke rute resmi terbaru `/backend-api/f/conversation`.
   - Menambahkan mekanisme fallback slug model agar jika request memanggil slug yang usang atau `auto`, sistem otomatis memetakan ke model aktif (`gpt-4o`).

2. **Multimodal Studio & Zero-Disk Memory Streaming**:
   - Endpoint `/v1/images/generations` dan `/v1/videos/generations` siap digunakan.
   - **Zero-Storage Policy**: Mengalirkan biner gambar dan video secara langsung di memori RAM (*in-memory proxy streaming* via `/v1/images/{file_id}` dan `/v1/videos/{file_id}`) dengan auto-purge TTL. **Tidak menimbun file biner di harddisk/storage lokal.**
   - **Fallback Generator**: Mesin visual fallback in-memory (FLUX/SDXL) yang otomatis aktif jika upstream penyedia gambar sedang mengalami kendala captcha atau rate limit.
   - **Web UI Media**: Dilengkapi HTML5 Video Player, Image Lightbox preview pop-up, dan tombol unduh instan.

3. **Multi-Turn Chat Completion**:
   - Menangani riwayat percakapan (`messages: [...]`) secara utuh pada `ChatStream` dan `Chat` non-streaming.
   - Provider Perplexity aktif dan merespons lancar (model `best` dan `sonar`).
   - Provider Gemini Web mendukung batch RPC `otAQ7b` / `StreamGenerate`.

4. **Kesiapan Distribusi & STB Armbian**:
   - Biner Windows (`cookie-router-windows-amd64.exe`), Linux ARM64 (`cookie-router-linux-arm64`), dan Linux AMD64 (`cookie-router-linux-amd64`) telah dikompilasi dengan optimasi `-ldflags="-s -w"`.
   - Dibuatkan 4 varian paket rilis di folder `dist/`:
     - `cookie-router-v2-windows-portable.zip`
     - `cookie-router-v2-stb-armbian-arm64.zip`
     - `cookie-router-v2-all-binaries-multiplatform.zip`
     - `cookie-router-v2-source-code-clean.zip`
   - Telah dibuat `.gitignore` ketat untuk menjaga keamanan cookie (`config.json` tidak akan bocor).

---

## 📂 Peta Struktur Direktori Codebase

```
cookie-router/
├── cmd/
│   └── server/
│       └── main.go                 # Entrypoint server, CLI flag, graceful shutdown
├── internal/
│   ├── config/                     # Manajemen config.json, akun & provider
│   ├── providers/                  # Implementasi upstream provider web cookies
│   │   ├── chatgpt/                # ChatGPT Web + Internal Sentinel PoW Engine
│   │   ├── perplexity/             # Perplexity Web API
│   │   ├── gemini/                 # Google Gemini Web batch RPC
│   │   ├── claude/                 # Anthropic Claude Web
│   │   ├── grok/                   # xAI Grok Web
│   │   ├── deepseek/               # DeepSeek Web
│   │   ├── poe/                    # Poe Web
│   │   ├── qwen/                   # Qwen Web
│   │   ├── kimi/                   # Moonshot Kimi Web
│   │   ├── custom/                 # Universal custom endpoint
│   │   └── fallback/               # In-Memory AI image fallback generator
│   ├── router/                     # Round-robin account rotation & load balancing
│   ├── server/                     # HTTP router, OpenAI standard handlers, media proxy
│   └── web/                        # Embedded HTML/CSS/JS Dashboard & Playground
├── dist/                           # Paket rilis zip & binary
├── config.example.json             # Template konfigurasi bersih
├── run-windows.bat                 # Skrip 1-klik start Windows
├── README.md                       # Dokumentasi lengkap proyek
├── DEVELOPMENT-SNAPSHOT.md         # Dokumen checkpoint ini
└── .gitignore                      # Proteksi keamanan kredensial
```

---

## 🛠️ Perintah Berguna untuk Melanjutkan Besok

### 1. Menjalankan Server Lokal (Development)
```bash
# Menjalankan langsung via Go:
go run cmd/server/main.go

# Atau menjalankan biner Windows yang sudah ada:
./cookie-router-windows-amd64.exe

# Atau via file batch 1-klik:
run-windows.bat
```

### 2. Melakukan Compile / Build Biner Baru
```bash
# Windows:
go build -ldflags="-s -w" -o cookie-router.exe cmd/server/main.go

# Linux ARM64 (STB Armbian TV Box):
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/cookie-router-linux-arm64 cmd/server/main.go

# Linux AMD64 (VPS / Docker):
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/cookie-router-linux-amd64 cmd/server/main.go
```

### 3. Menguji Endpoint via cURL
```bash
# Cek Daftar Model
curl -s http://localhost:18899/v1/models

# Tes Chat Completion
curl -X POST http://localhost:18899/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"best","messages":[{"role":"user","content":"halo"}],"stream":false}'

# Tes Chat Streaming
curl -N -X POST http://localhost:18899/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"sonar","messages":[{"role":"user","content":"halo"}],"stream":true}'
```

---

## 📋 Catatan & Rencana Pengembangan Selanjutnya (Backlog Besok)

1. **Auto Session Renewal / Token Refresher**:
   - Mengeksplorasi automasi sinkronisasi cookie langsung dari peramban lokal (CDP) ke `config.json` agar saat cookie ChatGPT/Google kedaluwarsa, router bisa otomatis meminta pembaruan tanpa harus copy-paste manual.
2. **Pengujian Fisik di STB Armbian**:
   - Mentransfer `dist/cookie-router-v2-stb-armbian-arm64.zip` ke STB fisik, menjalankan `install.sh`, dan memantau penggunaan RAM secara live (`systemctl status cookie-router`).
3. **Ekspansi Fitur Playground**:
   - Menambahkan opsi upload gambar (Vision / multimodal input) di playground chat UI.
4. **Push ke GitHub**:
   - Jika ingin membackup kode ke remote GitHub, tinggal buat private repository kosong di GitHub lalu hubungkan remote git local ini.
