# 🍪 Cookie Router v2.0.0

**Universal Multi-Provider Web Cookie AI Gateway & Multimodal Hub**
*Ultra-lightweight Go standalone binary (<20MB RAM) — Engineered for Windows, Linux VPS, and Armbian STB TV Box.*

---

## 🌟 Fitur Utama v2.0.0

1. **Multi-Provider Web Cookies Tanpa API Key Berbayar**:
   - Mendukung login web browser pengguna via cookies/session token:
     - **ChatGPT Web**: Otomatis menangani JWT Bearer, internal Proof-of-Work Sentinel solver, auto-injeksi `ChatGPT-Account-Id` untuk akun Plus/Workspace, rute `/backend-api/f/conversation`, dan auto-hide thread (`is_visible: false`).
     - **Perplexity Web**: Mendukung model `best`, `sonar`, `sonar-reasoning`, `sonar-deep-research`, `claude-sonnet-5.5`, `gpt-6.1-sol`.
     - **Gemini Web**: Mendukung batch RPC streaming dan model `gemini-2.0-flash`, `gemini-2.5-pro`, `gemini-2.5-flash`, `imagen-3`.
     - **Claude Web**: Cookie `sessionKey` claude.ai.
     - **Grok Web**: SSO authentication grok.com.
     - **DeepSeek Web**: User token chat.deepseek.com.
     - **Poe, Qwen, Kimi, Universal Custom Endpoint**.

2. **Multimodal Media Studio (Image & Video Generation)**:
   - Endpoint OpenAI Standard:
     - `POST /v1/images/generations`
     - `POST /v1/videos/generations`
     - `POST /api/videos/generate`
   - **Zero-Disk Memory Streaming**: Media gambar dan video dialirkan secara *on-the-fly* langsung dari memori RAM (dengan TTL otomatis) via proxy `/v1/images/{file_id}` dan `/v1/videos/{file_id}`. **Tidak menimbun file biner di harddisk atau storage perangkat Anda!**
   - **In-Memory Fallback Generator**: Dilengkapi engine AI visual fallback otomatis (FLUX/SDXL in-memory) sehingga request gambar tidak akan pernah gagal meskipun upstream sedang limit/terhalang captcha.
   - **Integrated Lightbox & HTML5 Video Player**: Web UI dilengkapi tombol putar, preview pop-up, dan 1-klik unduh instan.

3. **Web Dashboard & Multi-Turn Chat Playground**:
   - Akses via browser: `http://localhost:18899/`
   - Tab Chat, Image Studio, Video Studio.
   - Pengalihan Round-Robin otomatis antar akun.
   - Live Code Generator untuk cURL, Python, Cursor, Cline, dan Aider.

4. **Ultra-Ringan & STB Armbian Ready**:
   - Standalone Go binary (zero external runtime dependencies, tanpa butuh Python atau Node.js di server).
   - Konsumsi RAM hanya ~15 - 25 MB.
   - Dilengkapi file `install.sh` dan systemd unit siap pakai dengan limit `MemoryMax=64M` & `CPUQuota=80%`.

---

## 🚀 Panduan Memulai Cepat

### 1. Di Windows
Jalankan file executable:
```cmd
cookie-router-windows-amd64.exe
```
Buka peramban di `http://localhost:18899/`.

### 2. Di STB Armbian TV Box (ARM64)
```bash
# 1. Ekstrak zip paket STB
unzip cookie-router-v2-stb-armbian.zip -d cookie-router
cd cookie-router

# 2. Jalankan installer otomatis (otomatis setup systemd)
bash install.sh

# 3. Layanan langsung berjalan di background!
# Cek status:
systemctl status cookie-router
```

### 3. Di Linux Server / VPS (AMD64)
```bash
chmod +x cookie-router-linux-amd64
./cookie-router-linux-amd64
```

---

## 📡 OpenAI API Compatibility

### Chat Completion (Multi-Turn & Streaming)
```bash
curl -X POST http://localhost:18899/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Halo, siapa kamu?"}
    ],
    "stream": true
  }'
```

### Image Generation (Zero-Disk Stream)
```bash
curl -X POST http://localhost:18899/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Seekor kucing lucu astronot di luar angkasa",
    "size": "1024x1024",
    "n": 1
  }'
```

### Video Generation (Zero-Disk Stream)
```bash
curl -X POST http://localhost:18899/v1/videos/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Cinematic drone view of tropical beach at sunset",
    "duration": 5
  }'
```

---

## ⚙️ Integrasi dengan Tool AI Koding

- **Base URL**: `http://localhost:18899/v1`
- **API Key**: Bebas diisi apa saja (atau sesuai `authToken` jika diaktifkan di dashboard).
- **Cursor IDE**:
  - Settings -> Models -> Override OpenAI Base URL: `http://localhost:18899/v1`
  - Model: `auto` / `best` / `sonar` / `gpt-4o` / `gemini-2.0-flash`
- **Cline / Roo-Code**:
  - Provider: OpenAI Compatible
  - Base URL: `http://localhost:18899/v1`
- **Hermes Agent**:
  ```yaml
  providers:
    cookie-router:
      url: http://localhost:18899/v1
      model: auto
  ```

---

## 📁 Struktur Rilis

- `cookie-router-windows-amd64.exe`: Executable Windows.
- `cookie-router-linux-arm64`: Biner Linux ARM64 untuk STB TV Box.
- `cookie-router-linux-amd64`: Biner Linux x86_64 untuk Server/VPS.
- `cookie-router.service`: Template systemd service untuk Linux.
- `install.sh`: Skrip instalasi 1-klik STB Armbian.
- `config.example.json`: Template konfigurasi cookie bersih tanpa akun pribadi.
- `run-windows.bat`: Skrip jalan 1-klik Windows.
