---
name: cookie-router
description: "Use when managing, routing LLMs to, or troubleshooting local Cookie Router gateway on port 18899."
version: 1.0.0
author: Tekno Sca, Hermes Agent
license: MIT
platforms: [linux, windows]
metadata:
  hermes:
    tags: [cookie-router, gateway, openai-proxy, gemini, chatgpt, llm-router, zero-disk]
    category: software-development
---

# Cookie Router Gateway

Cookie Router is a lightweight, zero-disk Web Cookie LLM Gateway running locally on STB Armbian as an OpenAI-compatible proxy (`http://127.0.0.1:18899/v1`). It translates OpenAI API calls into native browser session requests (Google Gemini Web, ChatGPT Plus, etc.) with streaming support and zero disk writes.

## Infrastructure & Service Details

- **Service Unit:** `cookie-router.service` (`systemctl status/restart/stop cookie-router`)
- **Binary Path:** `/opt/cookie-router/cookie-router` (static ARM64 binary, CGO=0)
- **Configuration:** `/opt/cookie-router/config.json` (permissions `chmod 600`)
- **Local Port:** `18899` (Base URL: `http://127.0.0.1:18899/v1`, LAN: `http://192.168.1.11:18899/v1`)
- **Dashboard UI:** `http://192.168.1.11:18899/`
- **Memory Ceiling:** 50MB RAM buffer limit for media assets with automatic LRU/FIFO eviction.

## API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/v1/chat/completions` | `POST` | OpenAI-compatible chat completion (SSE streaming supported) |
| `/v1/models` | `GET` | List all available models across active accounts |
| `/v1/images/generations` | `POST` | Multimodal image generation (Imagen 3 / DALL-E) |
| `/v1/videos/generations` | `POST` | Video generation (Veo-2 / dynamic cinematic MP4 stream) |
| `/v1/images/{id}.png` | `GET` | Zero-disk image stream (`?download=1` for master uncompressed PNG) |
| `/v1/videos/{id}.mp4` | `GET` | Zero-disk seekable MP4 stream (HTTP 206 Partial Content) |
| `/api/accounts` | `GET/POST` | List or register accounts |
| `/api/accounts/{id}` | `PUT/DELETE`| Update account cookies or remove account |
| `/api/accounts/models/{id}` | `POST` | Trigger background model probe for an account |

## Supported Models

- **Google Gemini:** `gemini-4-argon`, `gemini-4-argon-high`, `gemini-4-argon-medium`, `gemini-4-argon-low`, `gemini-3.1-pro`, `gemini-3.8-flash`, `gemini-3.5-flash-lite`, `imagen-3`, `nano-banana`, `veo-2`.
- **ChatGPT:** `gpt-6`, `gpt-6-thinking`, `gpt-6-instant`, `gpt-5-6`, `gpt-5-6-thinking`, `gpt-5-6-instant`, `gpt-4o`, `dall-e-3`, `auto`.

## Routing & Integration Examples

### 1. Hermes Agent Provider (`config.yaml`)
To use Cookie Router as a model provider in Hermes Agent on this STB:
```yaml
model: gemini-4-argon
provider: custom
custom_providers:
  cookie_router:
    base_url: http://127.0.0.1:18899/v1
    api_key: sk-local
```

### 2. Python Client (OpenAI SDK)
```python
from openai import OpenAI

client = OpenAI(base_url="http://127.0.0.1:18899/v1", api_key="sk-local")
resp = client.chat.completions.create(
    model="gemini-4-argon",
    messages=[{"role": "user", "content": "Halo!"}],
    stream=True
)
for chunk in resp:
    print(chunk.choices[0].delta.content or "", end="")
```

## Troubleshooting & Maintenance

1. **Error: "Apakah Anda sudah masuk? Saya bisa mencari gambar..." on Gemini Image Gen:**
   - **Root Cause:** Cookie timestamp signature (`__Secure-1PSIDTS`) or anti-CSRF token (`SNlM0e`) expired.
   - **Fix:** Copy fresh cookies from browser tab `gemini.google.com`, update the account via `PUT /api/accounts/{id}` or edit `/opt/cookie-router/config.json`, then restart: `systemctl restart cookie-router`.

2. **Error 401 token_revoked on ChatGPT:**
   - **Root Cause:** Session token expired.
   - **Fix:** Log in to `chatgpt.com`, copy the fresh `__Secure-next-auth.session-token` cookie, update in `/opt/cookie-router/config.json`, and restart.

3. **Service Management Commands:**
   - Status: `systemctl status cookie-router`
   - Realtime logs: `journalctl -u cookie-router -f`
   - Restart: `systemctl restart cookie-router`
