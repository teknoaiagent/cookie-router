package web

const DashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>🍪 Cookie Router</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh}
.container{max-width:980px;margin:0 auto;padding:20px}
header{display:flex;align-items:center;justify-content:space-between;padding:16px 0;border-bottom:1px solid #1e293b;margin-bottom:24px}
h1{font-size:1.5rem;color:#38bdf8}
h1 span{font-size:1.8rem;margin-right:8px}
.badge{background:#1e293b;color:#94a3b8;padding:4px 12px;border-radius:12px;font-size:0.75rem}
.card{background:#1e293b;border-radius:12px;padding:20px;margin-bottom:16px;border:1px solid #334155}
.card h2{font-size:1.1rem;color:#38bdf8;margin-bottom:12px}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}
.form-group{display:flex;flex-direction:column;gap:4px}
.form-group.full{grid-column:1/-1}
label{font-size:0.8rem;color:#94a3b8;font-weight:600;text-transform:uppercase;letter-spacing:0.5px}
input,select,textarea{background:#0f172a;border:1px solid #334155;border-radius:8px;padding:10px 12px;color:#e2e8f0;font-size:0.9rem;outline:none;transition:border 0.2s}
input:focus,select:focus,textarea:focus{border-color:#38bdf8}
textarea{resize:vertical;min-height:90px;font-family:monospace;font-size:0.8rem}
.btn{padding:10px 20px;border-radius:8px;border:none;cursor:pointer;font-size:0.85rem;font-weight:600;transition:all 0.2s}
.btn-primary{background:#0ea5e9;color:#fff}.btn-primary:hover{background:#0284c7}
.btn-danger{background:#ef4444;color:#fff}.btn-danger:hover{background:#dc2626}
.btn-success{background:#22c55e;color:#fff}.btn-success:hover{background:#16a34a}
.btn-detect{background:#6366f1;color:#fff}.btn-detect:hover{background:#4f46e5}
.btn-sm{padding:6px 12px;font-size:0.75rem}
.btn-group{display:flex;gap:8px;margin-top:12px;align-items:center}
.accounts-list{display:flex;flex-direction:column;gap:12px}
.account-item{background:#0f172a;border:1px solid #334155;border-radius:8px;padding:16px;display:flex;flex-direction:column;gap:12px}
@media (min-width: 768px) {
  .account-item-header{display:flex;align-items:center;justify-content:space-between}
}
.account-info{display:flex;align-items:center;gap:12px}
.provider-badge{padding:4px 10px;border-radius:6px;font-size:0.75rem;font-weight:700;text-transform:uppercase}
.provider-chatgpt{background:#10a37f20;color:#10a37f;border:1px solid #10a37f40}
.provider-claude{background:#d9775720;color:#d97757;border:1px solid #d9775740}
.provider-grok{background:#1da1f220;color:#1da1f2;border:1px solid #1da1f240}
.provider-deepseek{background:#4d6bfe20;color:#4d6bfe;border:1px solid #4d6bfe40}
.provider-gemini{background:#4285f420;color:#4285f4;border:1px solid #4285f440}
.status-dot{width:8px;height:8px;border-radius:50%;display:inline-block}
.status-active{background:#22c55e}.status-error{background:#ef4444}.status-untested{background:#94a3b8}
.account-meta{font-size:0.8rem;color:#94a3b8}
.endpoint-box{background:#0f172a;border:1px solid #334155;border-radius:8px;padding:14px;font-family:monospace;font-size:0.85rem;color:#38bdf8;word-break:break-all}
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:16px}
.stat{background:#1e293b;border-radius:8px;padding:16px;text-align:center;border:1px solid #334155}
.stat-value{font-size:1.8rem;font-weight:700;color:#38bdf8}
.stat-label{font-size:0.75rem;color:#94a3b8;margin-top:4px;text-transform:uppercase}
.toast{position:fixed;bottom:20px;right:20px;padding:12px 20px;border-radius:8px;color:#fff;font-size:0.85rem;opacity:0;transition:opacity 0.3s;z-index:999}
.toast.show{opacity:1}
.toast-success{background:#22c55e}.toast-error{background:#ef4444}.toast-info{background:#6366f1}
.badge-model{display:inline-block;background:#0ea5e920;color:#38bdf8;border:1px solid #0ea5e940;border-radius:4px;padding:3px 7px;font-size:0.75rem;font-family:monospace}
.models-box{margin-top:8px;padding-top:8px;border-top:1px dashed #334155;display:flex;flex-wrap:wrap;gap:6px;align-items:center}
.models-title{font-size:0.75rem;color:#94a3b8;font-weight:600;margin-right:4px}
#addForm{display:none}
.tab-nav{display:flex;gap:6px;border-bottom:1px solid #334155;margin:12px 0;overflow-x:auto;padding-bottom:4px}
.tab-btn{background:#0f172a;border:1px solid #334155;color:#94a3b8;padding:6px 12px;border-radius:6px;cursor:pointer;font-size:0.8rem;font-weight:600;white-space:nowrap;transition:all 0.2s}
.tab-btn:hover{background:#1e293b;color:#f1f5f9}
.tab-btn.active{background:#0ea5e920;color:#38bdf8;border-color:#38bdf8}
.tab-pane{display:none}
.tab-pane.active{display:block}
.spec-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:10px;margin:12px 0}
.spec-item{background:#0f172a;border:1px solid #334155;border-radius:6px;padding:10px 12px}
.spec-label{font-size:0.7rem;color:#94a3b8;text-transform:uppercase;margin-bottom:4px;display:flex;justify-content:space-between;align-items:center}
.spec-val{font-family:monospace;font-size:0.82rem;color:#38bdf8;word-break:break-all}
.code-box{position:relative;background:#0f172a;border:1px solid #334155;border-radius:6px;padding:12px;margin:8px 0;font-family:monospace;font-size:0.8rem;color:#93c5fd;white-space:pre-wrap;word-break:break-all;line-height:1.4}
.code-copy-btn{position:absolute;top:6px;right:6px;background:#1e293b;border:1px solid #334155;color:#cbd5e1;padding:3px 8px;border-radius:4px;font-size:0.72rem;cursor:pointer}
.code-copy-btn:hover{background:#334155;color:#fff}
.guide-ol{padding-left:18px;margin:8px 0;font-size:0.82rem;line-height:1.6}
.guide-ol li{margin-bottom:6px}
.clickable-val{background:#1e293b;padding:2px 6px;border-radius:4px;border:1px solid #334155;color:#38bdf8;cursor:pointer;font-family:monospace}
.clickable-val:hover{border-color:#38bdf8;background:#0f172a}

/* Modern Chat Web UI Styles */
.chat-container{display:flex;flex-direction:column;background:#090d16;border:1px solid #1e293b;border-radius:12px;overflow:hidden;box-shadow:0 10px 30px rgba(0,0,0,0.4)}
.chat-topbar{display:flex;align-items:center;justify-content:space-between;padding:10px 16px;background:#0f172a;border-bottom:1px solid #1e293b;flex-wrap:wrap;gap:8px}
.chat-topbar-left{display:flex;align-items:center;gap:10px;flex:1}
.chat-topbar-right{display:flex;align-items:center;gap:8px}
.chat-model-select{font-family:monospace;background:#090d16;border:1px solid #334155;border-radius:6px;padding:6px 12px;color:#38bdf8;font-size:0.85rem;font-weight:600;outline:none}
.chat-model-select:focus{border-color:#38bdf8}
.chat-messages{padding:18px;min-height:380px;max-height:560px;overflow-y:auto;display:flex;flex-direction:column;gap:16px;scroll-behavior:smooth;background:radial-gradient(circle at 50% 0%,#0f172a 0%,#090d16 100%)}
.chat-empty-state{display:flex;flex-direction:column;align-items:center;justify-content:center;padding:50px 20px;color:#64748b;text-align:center}
.chat-empty-icon{font-size:2.4rem;margin-bottom:10px;opacity:0.8}
.chat-empty-title{color:#94a3b8;font-weight:600;font-size:1rem;margin-bottom:6px}
.chat-empty-desc{font-size:0.8rem;max-width:420px;line-height:1.5}
.chat-message-row{display:flex;gap:12px;align-items:flex-start;animation:fadeInMsg 0.25s ease-out forwards}
@keyframes fadeInMsg{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}
.chat-message-row.user{flex-direction:row-reverse}
.chat-avatar{width:34px;height:34px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:0.9rem;font-weight:700;flex-shrink:0;box-shadow:0 2px 8px rgba(0,0,0,0.3)}
.chat-avatar.user{background:linear-gradient(135deg,#0284c7,#38bdf8);color:#fff}
.chat-avatar.assistant{background:linear-gradient(135deg,#6366f1,#a855f7);color:#fff}
.chat-bubble-container{max-width:82%;display:flex;flex-direction:column}
.chat-message-row.user .chat-bubble-container{align-items:flex-end}
.chat-bubble-meta{display:flex;align-items:center;gap:8px;margin-bottom:4px;font-size:0.72rem;color:#94a3b8}
.chat-meta-model{background:#1e293b;border:1px solid #334155;color:#38bdf8;padding:1px 6px;border-radius:4px;font-family:monospace}
.chat-meta-time{color:#64748b}
.chat-bubble{padding:12px 16px;border-radius:12px;font-size:0.88rem;line-height:1.6;word-break:break-word;position:relative}
.chat-message-row.user .chat-bubble{background:#1e293b;border:1px solid #334155;color:#f1f5f9;border-top-right-radius:2px}
.chat-message-row.assistant .chat-bubble{background:#0f172a;border:1px solid #1e293b;color:#cbd5e1;border-top-left-radius:2px}
.chat-bubble-actions{display:flex;align-items:center;gap:6px;margin-top:6px}
.chat-action-btn{background:transparent;border:1px solid #334155;color:#94a3b8;font-size:0.7rem;padding:2px 8px;border-radius:4px;cursor:pointer;transition:all 0.2s}
.chat-action-btn:hover{background:#1e293b;color:#fff;border-color:#475569}

/* Dedicated Media Showcase Container */
.media-showcase-card{margin:12px 0;background:#050811;border:1px solid #1e3a8a;border-radius:10px;overflow:hidden;box-shadow:0 8px 24px rgba(0,0,0,0.5)}
.media-card-header{display:flex;align-items:center;justify-content:space-between;padding:10px 14px;background:#0a1124;border-bottom:1px solid #1e293b;flex-wrap:wrap;gap:8px}
.media-badge-zero{font-size:0.72rem;background:#064e3b;color:#34d399;border:1px solid #059669;padding:2px 8px;border-radius:12px;font-weight:600}
.media-card-viewport{position:relative;display:flex;align-items:center;justify-content:center;background:#02040a;min-height:240px;max-height:520px;padding:12px;cursor:zoom-in;overflow:hidden}
.media-image-preview{max-width:100%;max-height:500px;width:auto;height:auto;object-fit:contain;border-radius:6px;transition:transform 0.25s ease}
.media-card-viewport:hover .media-image-preview{transform:scale(1.015)}
.media-zoom-overlay{position:absolute;bottom:12px;right:12px;background:rgba(15,23,42,0.85);backdrop-filter:blur(4px);color:#e2e8f0;border:1px solid #334155;border-radius:6px;padding:4px 10px;font-size:0.75rem;pointer-events:none;opacity:0.8;transition:opacity 0.2s}
.media-card-viewport:hover .media-zoom-overlay{opacity:1}
.media-card-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding:10px 14px;background:#0a1124;border-top:1px solid #1e293b;flex-wrap:wrap}
.btn-media{padding:6px 12px;border-radius:6px;font-size:0.78rem;font-weight:600;text-decoration:none;border:1px solid transparent;cursor:pointer;display:inline-flex;align-items:center;gap:4px;transition:all 0.2s}
.btn-media-zoom{background:#1e293b;color:#38bdf8;border-color:#334155}
.btn-media-zoom:hover{background:#334155;color:#fff}
.btn-media-download{background:#2563eb;color:#fff}
.btn-media-download:hover{background:#1d4ed8}
.btn-media-copy{background:#1e293b;color:#94a3b8;border-color:#334155}
.btn-media-copy:hover{background:#334155;color:#fff}

/* Google Gemini Model Switcher styles */
.gemini-switcher{position:relative;display:inline-block;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}
.gemini-switcher-btn{background:#090d16;border:1px solid #334155;border-radius:8px;padding:6px 14px;color:#f1f5f9;font-size:0.85rem;font-weight:600;display:inline-flex;align-items:center;gap:6px;cursor:pointer;transition:all 0.2s;outline:none}
.gemini-switcher-btn:hover{background:#1e293b;border-color:#38bdf8}
.gemini-switcher-btn .g-sparkle{color:#38bdf8;font-size:0.9rem}
.gemini-switcher-btn .g-chevron{color:#94a3b8;font-size:0.7rem;margin-left:2px}
.gemini-menu{display:none;position:absolute;top:calc(100% + 6px);left:0;background:#111520;border:1px solid #283347;border-radius:12px;box-shadow:0 14px 40px rgba(0,0,0,0.65), 0 2px 10px rgba(0,0,0,0.5);min-width:240px;padding:6px 0;z-index:9999}
.gemini-menu.show{display:block}
.gemini-item{display:flex;align-items:center;gap:10px;padding:8px 14px;cursor:pointer;transition:background 0.15s;position:relative;border-radius:6px;margin:2px 4px}
.gemini-item:hover{background:#1e2638}
.gemini-check{width:16px;font-size:0.9rem;font-weight:700;color:#38bdf8;display:inline-block;text-align:center}
.gemini-text-block{flex:1;display:flex;flex-direction:column;gap:1px}
.gemini-title{font-size:0.86rem;font-weight:600;color:#f8fafc}
.gemini-sub{font-size:0.75rem;color:#94a3b8}
.gemini-divider{height:1px;background:#242d3d;margin:4px 0}
.gemini-arrow{color:#94a3b8;font-size:0.8rem}
.gemini-submenu{display:none;position:absolute;top:0;left:100%;margin-left:4px;background:#111520;border:1px solid #283347;border-radius:12px;box-shadow:0 14px 40px rgba(0,0,0,0.65), 0 2px 10px rgba(0,0,0,0.5);min-width:200px;padding:6px 0;z-index:10000}
.gemini-parent-item:hover .gemini-submenu, .gemini-submenu.show{display:block}

/* ChatGPT Thinking Effort UI */
.chatgpt-effort-switcher{position:relative;display:inline-block;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}
.chatgpt-effort-btn{background:#090d16;border:1px solid #334155;border-radius:20px;padding:6px 14px;color:#cbd5e1;font-size:0.82rem;font-weight:500;display:inline-flex;align-items:center;gap:6px;cursor:pointer;transition:all 0.2s;outline:none}
.chatgpt-effort-btn:hover{background:#1e293b;border-color:#38bdf8;color:#fff}
.cgpt-chevron{font-size:0.75rem;color:#94a3b8}
.chatgpt-effort-menu{display:none;position:absolute;top:calc(100% + 6px);left:0;background:#181b22;border:1px solid #2d3444;border-radius:14px;box-shadow:0 14px 40px rgba(0,0,0,0.7);padding:14px 16px;min-width:220px;z-index:9999}
.chatgpt-effort-menu.show{display:block}
.cgpt-effort-header{display:flex;align-items:center;justify-content:space-between;cursor:pointer;padding-bottom:10px;font-size:0.92rem;border-bottom:1px solid #283040}
.cgpt-slider-box{margin-top:12px;position:relative}
.cgpt-slider-box input[type=range]{width:100%;-webkit-appearance:none;height:6px;border-radius:3px;background:#334155;outline:none;cursor:pointer}
.cgpt-slider-box input[type=range]::-webkit-slider-thumb{-webkit-appearance:none;width:16px;height:16px;border-radius:50%;background:#ffffff;box-shadow:0 2px 6px rgba(0,0,0,0.5);cursor:pointer}
.cgpt-slider-dots{display:flex;justify-content:space-between;position:relative;top:-11px;pointer-events:none;padding:0 2px}
.cgpt-dot{width:6px;height:6px;border-radius:50%;background:#64748b}
.cgpt-slider-labels{display:flex;justify-content:space-between;font-size:0.72rem;color:#94a3b8;margin-top:2px}

/* Code block wrapper */
.code-block-wrapper{margin:10px 0;background:#02040a;border:1px solid #1e293b;border-radius:8px;overflow:hidden}
.code-block-header{display:flex;align-items:center;justify-content:space-between;padding:6px 12px;background:#0d1526;border-bottom:1px solid #1e293b;font-size:0.72rem;color:#94a3b8;text-transform:uppercase}
.code-block-body{padding:12px;margin:0;font-family:monospace;font-size:0.8rem;color:#93c5fd;overflow-x:auto;line-height:1.5}

/* Chat Input Bar */
.chat-input-bar{display:flex;gap:10px;padding:12px 16px;background:#0f172a;border-top:1px solid #1e293b;align-items:flex-end}
.chat-textarea{flex:1;background:#090d16;border:1px solid #334155;border-radius:8px;padding:10px 14px;color:#f1f5f9;font-size:0.9rem;line-height:1.5;min-height:44px;max-height:180px;resize:none;outline:none;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;transition:border-color 0.2s}
.chat-textarea:focus{border-color:#38bdf8;box-shadow:0 0 0 2px rgba(56,189,248,0.15)}
.chat-input-hints{display:flex;justify-content:space-between;padding:4px 16px 8px;background:#0f172a;font-size:0.7rem;color:#64748b}

/* Lightbox Modal */
.lightbox-modal{display:none;position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.92);z-index:10000;align-items:center;justify-content:center;padding:20px;backdrop-filter:blur(8px)}
.lightbox-modal.active{display:flex}
.lightbox-content{position:relative;max-width:95vw;max-height:95vh;display:flex;flex-direction:column;align-items:center}
.lightbox-image{max-width:100%;max-height:85vh;object-fit:contain;border-radius:8px;box-shadow:0 20px 50px rgba(0,0,0,0.8)}
.lightbox-close{position:absolute;top:-40px;right:0;background:transparent;border:none;color:#f1f5f9;font-size:1.8rem;cursor:pointer}
.lightbox-caption{margin-top:10px;color:#cbd5e1;font-size:0.85rem;text-align:center}

/* Typing / Thinking Animation */
.typing-dots{display:inline-flex;align-items:center;gap:4px}
.typing-dots span{width:6px;height:6px;background:#38bdf8;border-radius:50%;animation:typingBounce 1.4s infinite ease-in-out both}
.typing-dots span:nth-child(1){animation-delay:-0.32s}
.typing-dots span:nth-child(2){animation-delay:-0.16s}
@keyframes typingBounce{0%,80%,100%{transform:scale(0);opacity:0.3}40%{transform:scale(1);opacity:1}}

/* Media mode selector & Video Showcase */
.mode-toggle-group{display:inline-flex;gap:4px;background:#090d16;padding:3px;border-radius:8px;border:1px solid #334155}
.mode-toggle-btn{background:transparent;border:none;color:#94a3b8;padding:5px 12px;border-radius:6px;font-size:0.8rem;font-weight:600;cursor:pointer;display:inline-flex;align-items:center;gap:6px;transition:all 0.2s}
.mode-toggle-btn:hover{color:#e2e8f0;background:#1e293b}
.mode-toggle-btn.active{background:#0ea5e9;color:#fff;box-shadow:0 2px 8px rgba(14,165,233,0.3)}

.media-video-player{width:100%;max-height:480px;border-radius:6px;background:#000;outline:none;display:block}
.media-gallery-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(210px,1fr));gap:12px;margin-top:12px}
.media-gallery-item{background:#090d16;border:1px solid #334155;border-radius:8px;overflow:hidden;transition:all 0.2s;display:flex;flex-direction:column}
.media-gallery-item:hover{transform:translateY(-2px);border-color:#38bdf8;box-shadow:0 6px 16px rgba(0,0,0,0.4)}
.media-gallery-thumb{width:100%;height:130px;object-fit:cover;background:#02040a;display:block}
.media-gallery-meta{padding:8px 10px;font-size:0.75rem;flex:1;display:flex;flex-direction:column;justify-content:space-between}
</style>
</head>
<body>
<div class="container">
<header>
 <h1><span>🍪</span> Cookie Router</h1>
 <span class="badge" id="version">v1.1.0 · Multi-Model Router</span>
</header>

<div class="stats">
 <div class="stat"><div class="stat-value" id="totalAccounts">0</div><div class="stat-label">Accounts</div></div>
 <div class="stat"><div class="stat-value" id="activeAccounts">0</div><div class="stat-label">Active</div></div>
 <div class="stat"><div class="stat-value" id="totalProviders">0</div><div class="stat-label">Providers</div></div>
 <div class="stat"><div class="stat-value" id="totalModels">0</div><div class="stat-label">Available Models</div></div>
</div>

<div class="card">
 <h2>🔌 Agent, CLI & IDE Integration Guides (API Docs)</h2>
 <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:12px">
  Connect Cursor, VS Code (Cline / Roo / Continue), Antigravity CLI, Hermes Agent, or any OpenAI-compatible app directly to your active Web LLM accounts.
 </p>

 <div class="spec-grid">
  <div class="spec-item">
   <div class="spec-label">OpenAI Base URL <button class="code-copy-btn" style="position:static" onclick="copySnippet(document.getElementById('specBaseUrl').textContent, this)">Copy</button></div>
   <div class="spec-val" id="specBaseUrl">http://localhost:18899/v1</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">API Key <button class="code-copy-btn" style="position:static" onclick="copySnippet(document.getElementById('specApiKey').textContent, this)">Copy</button></div>
   <div class="spec-val" id="specApiKey">sk-cookie-router</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">Chat Endpoint <button class="code-copy-btn" style="position:static" onclick="copySnippet(document.getElementById('specChatUrl').textContent, this)">Copy</button></div>
   <div class="spec-val" id="specChatUrl">http://localhost:18899/v1/chat/completions</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">Models List <button class="code-copy-btn" style="position:static" onclick="copySnippet(document.getElementById('specModelsUrl').textContent, this)">Copy</button></div>
   <div class="spec-val" id="specModelsUrl">http://localhost:18899/v1/models</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">Context Window / Max Output</div>
   <div class="spec-val" style="color:#22c55e">128,000 / 8,192 tokens</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">Routing &amp; Resilience</div>
   <div class="spec-val" style="color:#c084fc">Round-Robin &bull; Auto-Failover</div>
  </div>
  <div class="spec-item">
   <div class="spec-label">JSON Documentation</div>
   <div class="spec-val"><a href="/api/docs" target="_blank" style="color:#38bdf8;text-decoration:none">View /api/docs ↗</a></div>
  </div>
 </div>

 <div class="tab-nav">
  <button class="tab-btn active" data-tab="cursor" onclick="switchGuideTab('cursor')">Cursor IDE</button>
  <button class="tab-btn" data-tab="antigravity" onclick="switchGuideTab('antigravity')">Antigravity CLI</button>
  <button class="tab-btn" data-tab="vscode" onclick="switchGuideTab('vscode')">VS Code (Cline / Roo / Continue)</button>
  <button class="tab-btn" data-tab="hermes" onclick="switchGuideTab('hermes')">Hermes Agent</button>
  <button class="tab-btn" data-tab="python" onclick="switchGuideTab('python')">Python & cURL</button>
  <button class="tab-btn" data-tab="aider" onclick="switchGuideTab('aider')">Aider / CLI</button>
  <button class="tab-btn" data-tab="media" onclick="switchGuideTab('media')">🎨 Image & 🎬 Video API</button>
 </div>

 <!-- Pane: Cursor -->
 <div id="tab-cursor" class="tab-pane active">
  <ol class="guide-ol">
   <li>Open <strong>Cursor Settings</strong> (gear icon at top right) &rarr; Select <strong>Models</strong>.</li>
   <li>Toggle <strong>OpenAI API Key</strong> to <code>ON</code>. Enter API key: <span class="clickable-val" onclick="copySnippet(document.getElementById('cursorApiKey').textContent, this)" id="cursorApiKey">sk-cookie-router</span></li>
   <li>Click <strong>Override OpenAI Base URL</strong> and enter: <span class="clickable-val" onclick="copySnippet(document.getElementById('cursorBaseUrl').textContent, this)" id="cursorBaseUrl">http://localhost:18899/v1</span></li>
   <li>Under <strong>Model List</strong>, click <strong>+ Add Model</strong> and type: <span class="clickable-val" onclick="copySnippet(document.getElementById('cursorModel').textContent, this)" id="cursorModel">gpt-5-6-instant</span> (or <code>gpt-5-6</code>, <code>auto</code>).</li>
   <li>Save and start coding! Cursor will route requests directly through your browser cookie session.</li>
  </ol>
 </div>

 <!-- Pane: Antigravity -->
 <div id="tab-antigravity" class="tab-pane">
  <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:6px">Option 1: Set environment variables in your terminal:</p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('antigravityBash').textContent, this)">Copy</button>
   <pre id="antigravityBash"></pre>
  </div>
  <p style="font-size:0.8rem;color:#94a3b8;margin:8px 0 6px">Option 2: Place in your <code>antigravity.json</code> / project config:</p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('antigravityJson').textContent, this)">Copy</button>
   <pre id="antigravityJson"></pre>
  </div>
 </div>

 <!-- Pane: VS Code -->
 <div id="tab-vscode" class="tab-pane">
  <div style="margin-bottom:12px">
   <strong style="color:#38bdf8;font-size:0.85rem">A. Roo Code / Cline Extension Settings:</strong>
   <ol class="guide-ol" style="margin-top:4px">
    <li>API Provider: <strong>OpenAI Compatible</strong></li>
    <li>Base URL: <span class="clickable-val" onclick="copySnippet(document.getElementById('clineBaseUrl').textContent, this)" id="clineBaseUrl">http://localhost:18899/v1</span></li>
    <li>API Key: <span class="clickable-val" onclick="copySnippet(document.getElementById('clineApiKey').textContent, this)" id="clineApiKey">sk-cookie-router</span></li>
    <li>Model ID: <span class="clickable-val" onclick="copySnippet(document.getElementById('clineModel').textContent, this)" id="clineModel">gpt-5-6-instant</span></li>
    <li>Context Window: <code>128000</code> | Max Output: <code>8192</code></li>
   </ol>
  </div>
  <div>
   <strong style="color:#38bdf8;font-size:0.85rem">B. Continue.dev (<code>~/.continue/config.json</code>):</strong>
   <div class="code-box">
    <button class="code-copy-btn" onclick="copySnippet(document.getElementById('continueJson').textContent, this)">Copy</button>
    <pre id="continueJson"></pre>
   </div>
  </div>
 </div>

 <!-- Pane: Hermes Agent -->
 <div id="tab-hermes" class="tab-pane">
  <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:6px">In your Hermes configuration (<code>config.yaml</code> or active profile):</p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('hermesYaml').textContent, this)">Copy</button>
   <pre id="hermesYaml"></pre>
  </div>
 </div>

 <!-- Pane: Python & cURL -->
 <div id="tab-python" class="tab-pane">
  <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:6px"><strong>Python (Official OpenAI SDK):</strong></p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('pythonCode').textContent, this)">Copy</button>
   <pre id="pythonCode"></pre>
  </div>
  <p style="font-size:0.8rem;color:#94a3b8;margin:8px 0 6px"><strong>cURL (Terminal / Bash):</strong></p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('curlCmd').textContent, this)">Copy</button>
   <pre id="curlCmd"></pre>
  </div>
 </div>

 <!-- Pane: Aider -->
 <div id="tab-aider" class="tab-pane">
  <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:6px">Run Aider CLI directly using your Cookie Router gateway:</p>
  <div class="code-box">
   <button class="code-copy-btn" onclick="copySnippet(document.getElementById('aiderCmd').textContent, this)">Copy</button>
   <pre id="aiderCmd"></pre>
  </div>
 </div>

 <!-- Pane: Media API -->
 <div id="tab-media" class="tab-pane">
  <div style="margin-bottom:12px">
   <strong style="color:#38bdf8;font-size:0.85rem">1. Generate Image (POST /v1/images/generations):</strong>
   <p style="font-size:0.8rem;color:#94a3b8;margin:4px 0 6px">Mendukung Gemini Imagen 3 dan ChatGPT DALL-E dengan zero-disk memory streaming:</p>
   <div class="code-box">
    <button class="code-copy-btn" onclick="copySnippet(document.getElementById('curlImgCmd').textContent, this)">Copy</button>
    <pre id="curlImgCmd">curl -X POST http://localhost:18899/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Futuristic cyberpunk cat in neo Tokyo",
    "model": "imagen-3",
    "size": "1024x1024"
  }'</pre>
   </div>
  </div>
  <div>
   <strong style="color:#38bdf8;font-size:0.85rem">2. Generate Video (POST /v1/videos/generations):</strong>
   <p style="font-size:0.8rem;color:#94a3b8;margin:4px 0 6px">Mendukung Gemini Video / Veo-2 dengan zero-disk memory streaming MP4 seekable:</p>
   <div class="code-box">
    <button class="code-copy-btn" onclick="copySnippet(document.getElementById('curlVidCmd').textContent, this)">Copy</button>
    <pre id="curlVidCmd">curl -X POST http://localhost:18899/v1/videos/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A cute cat dancing on the beach at sunset",
    "model": "veo-2",
    "aspect_ratio": "16:9",
    "duration": 5
  }'</pre>
   </div>
  </div>
 </div>
</div>

<div class="card">
 <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
  <h2 style="margin-bottom:0">📋 Accounts</h2>
  <span style="font-size:0.75rem;padding:3px 8px;border-radius:12px;background:#3b0764;color:#d8b4fe;border:1px solid #7e22ce">⚡ Round-Robin &amp; Failover Active</span>
 </div>
 <div class="btn-group" style="margin:12px 0">
  <button class="btn btn-primary btn-sm" onclick="toggleAddForm()">+ Add Account</button>
  <button class="btn btn-sm" style="background:#334155;color:#e2e8f0" onclick="refreshAccounts()">↻ Refresh</button>
 </div>

 <div id="addForm" class="card" style="display:none;background:#0f172a;margin-bottom:12px">
  <div class="form-grid">
   <div class="form-group">
    <label>Provider</label>
    <select id="newProvider" onchange="updateHint()">
     <optgroup label="Web Cookie Providers">
      <option value="chatgpt">ChatGPT Web</option>
      <option value="claude">Claude Web</option>
      <option value="perplexity">Perplexity Web</option>
      <option value="grok">Grok Web</option>
      <option value="deepseek">DeepSeek Web</option>
      <option value="gemini">Gemini Web</option>
      <option value="poe">Poe Web</option>
      <option value="kimi">Kimi Moonshot Web</option>
      <option value="qwen">Qwen Web</option>
     </optgroup>
     <optgroup label="Custom / External API">
      <option value="custom">Custom OpenAI / Local LLM (Ollama, Groq, OpenRouter, etc.)</option>
     </optgroup>
    </select>
   </div>
   <div class="form-group">
    <label>Name (Label) <span style="font-size:0.75rem;color:#94a3b8;font-weight:normal">(Auto-detects from cookie if empty)</span></label>
    <input id="newName" placeholder="e.g. My Account or email@gmail.com">
   </div>
   <div class="form-group">
    <label>Default Model</label>
    <input id="newModel" placeholder="e.g. gpt-4o">
   </div>
   <div class="form-group full">
    <label>Cookie / Token / Cookie-Editor Export <span style="font-size:0.75rem;color:#38bdf8;font-weight:normal">⚡ Auto-detects Email &amp; Plan on paste</span></label>
    <textarea id="newCookie" placeholder="Paste session cookie, header string, or Cookie-Editor export (TSV/JSON)..." oninput="onCookieInput()" onpaste="setTimeout(onCookieInput, 50)"></textarea>
    <div id="cookieDetectStatus" style="display:none;margin-top:6px;font-size:0.8rem;padding:6px 12px;border-radius:6px"></div>
    <p id="cookieHint" style="font-size:0.75rem;color:#38bdf8;margin-top:4px"></p>
   </div>
  </div>
  <div class="btn-group">
   <button class="btn btn-success btn-sm" onclick="addAccount()">Save</button>
   <button class="btn btn-sm" style="background:#334155;color:#e2e8f0" onclick="toggleAddForm(false)">Cancel</button>
  </div>
 </div>

 <div class="accounts-list" id="accountsList"></div>
</div>

<div class="card" style="padding:0;overflow:hidden">
 <div class="chat-container">
  <div class="chat-topbar">
   <div class="chat-topbar-left" style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
    <div class="mode-toggle-group">
     <button class="mode-toggle-btn active" id="modeChatBtn" onclick="switchPlaygroundMode('chat')">💬 Chat</button>
     <button class="mode-toggle-btn" id="modeImageBtn" onclick="switchPlaygroundMode('image')">🎨 Image</button>
     <button class="mode-toggle-btn" id="modeVideoBtn" onclick="switchPlaygroundMode('video')">🎬 Video</button>
    </div>
    <select id="chatAccount" class="chat-model-select" title="Pilih Akun Eksekutor" onchange="onPlaygroundAccountChanged()">
     <option value="">🎯 Akun: Auto Round-Robin</option>
    </select>
    <select id="chatModel" class="chat-model-select" title="Pilih Model"></select>
    <div id="geminiSwitcher" class="gemini-switcher" style="display:none">
     <button type="button" id="geminiSwitcherBtn" class="gemini-switcher-btn" onclick="toggleGeminiMenu(event)">
      <span class="g-sparkle">✨</span>
      <span id="geminiActiveModelLabel">4 Argon</span>
      <span class="g-chevron">▾</span>
     </button>
     <div id="geminiMenu" class="gemini-menu">
      <div class="gemini-item" onclick="selectGeminiModel('gemini-3.5-flash-lite', '3.5 Flash-Lite')">
       <span class="gemini-check" id="check_35"></span>
       <div class="gemini-text-block">
        <div class="gemini-title">3.5 Flash-Lite</div>
        <div class="gemini-sub">Jawaban tercepat</div>
       </div>
      </div>
      <div class="gemini-item" onclick="selectGeminiModel('gemini-3.8-flash', '3.8 Flash')">
       <span class="gemini-check" id="check_38"></span>
       <div class="gemini-text-block">
        <div class="gemini-title">3.8 Flash</div>
        <div class="gemini-sub">Bantuan serbaguna</div>
       </div>
      </div>
      <div class="gemini-item" onclick="selectGeminiModel('gemini-3.1-pro', '3.1 Pro')">
       <span class="gemini-check" id="check_31"></span>
       <div class="gemini-text-block">
        <div class="gemini-title">3.1 Pro</div>
        <div class="gemini-sub">Penalaran yang canggih</div>
       </div>
      </div>
      <div class="gemini-item" onclick="selectGeminiModel('gemini-4-argon', '4 Argon')">
       <span class="gemini-check" id="check_4">✓</span>
       <div class="gemini-text-block">
        <div class="gemini-title">4 Argon</div>
        <div class="gemini-sub">Tugas kompleks</div>
       </div>
      </div>
      <div class="gemini-divider"></div>
      <div class="gemini-item gemini-parent-item" id="reasoningItem">
       <span class="gemini-check" style="visibility:hidden">✓</span>
       <div class="gemini-text-block">
        <div class="gemini-title" style="display:flex;align-items:center;justify-content:space-between">
         <span>Tingkat penalaran</span>
         <span class="gemini-arrow">▸</span>
        </div>
       </div>
       <div id="geminiSubmenu" class="gemini-submenu">
        <div class="gemini-item" onclick="selectGeminiModel('gemini-4-argon-low', '4 Argon (Rendah)', 'low')">
         <span class="gemini-check" id="check_r_low"></span>
         <div class="gemini-text-block">
          <div class="gemini-title">Rendah</div>
          <div class="gemini-sub">Cepat dan efisien</div>
         </div>
        </div>
        <div class="gemini-item" onclick="selectGeminiModel('gemini-4-argon-medium', '4 Argon (Sedang)', 'medium')">
         <span class="gemini-check" id="check_r_med"></span>
         <div class="gemini-text-block">
          <div class="gemini-title">Sedang</div>
          <div class="gemini-sub">Kedalaman seimbang</div>
         </div>
        </div>
        <div class="gemini-item" onclick="selectGeminiModel('gemini-4-argon-high', '4 Argon (Tinggi)', 'high')">
         <span class="gemini-check" id="check_r_high">✓</span>
         <div class="gemini-text-block">
          <div class="gemini-title">Tinggi</div>
          <div class="gemini-sub">Sangat menyeluruh</div>
         </div>
        </div>
       </div>
      </div>
     </div>
    </div>
    <div id="chatgptEffortSwitcher" class="chatgpt-effort-switcher" style="display:none">
     <button type="button" id="chatgptEffortBtn" class="chatgpt-effort-btn" onclick="toggleChatGPTEffortMenu(event)">
      <span>Thinking effort</span>
      <span class="cgpt-chevron">˅</span>
     </button>
     <div id="chatgptEffortMenu" class="chatgpt-effort-menu">
      <div class="cgpt-effort-header" onclick="cycleChatGPTEffort()">
       <span id="cgptEffortLabel" style="font-weight:700;color:#fff">High</span>
       <span style="color:#94a3b8;font-size:0.8rem">›</span>
      </div>
      <div class="cgpt-slider-box">
       <input type="range" id="cgptEffortRange" min="0" max="2" step="1" value="2" oninput="onChatGPTEffortSlide(this.value)">
       <div class="cgpt-slider-dots">
        <span class="cgpt-dot" onclick="onChatGPTEffortSlide(0)" title="Instant"></span>
        <span class="cgpt-dot" onclick="onChatGPTEffortSlide(1)" title="Medium"></span>
        <span class="cgpt-dot" onclick="onChatGPTEffortSlide(2)" title="High"></span>
       </div>
       <div class="cgpt-slider-labels">
        <span onclick="onChatGPTEffortSlide(0)" style="cursor:pointer">Instant</span>
        <span onclick="onChatGPTEffortSlide(1)" style="cursor:pointer">Medium</span>
        <span onclick="onChatGPTEffortSlide(2)" style="cursor:pointer">High</span>
       </div>
      </div>
     </div>
    </div>
    <div id="mediaOptionsRow" style="display:none;align-items:center;gap:6px">
     <select id="mediaAspect" class="chat-model-select" title="Rasio Aspek">
      <option value="16:9">16:9 Landscape</option>
      <option value="9:16">9:16 Portrait</option>
      <option value="1:1">1:1 Square</option>
      <option value="4:3">4:3 Standard</option>
     </select>
     <select id="mediaStyle" class="chat-model-select" title="Preset Gaya Visual" style="display:none">
      <option value="none" selected>📷 Asli / Realistis Bawaan (Rekomendasi)</option>
      <option value="flare">✨ Flare (Artistik Hangat)</option>
      <option value="sunburst">☀️ Sunburst (Sinar Keemasan)</option>
      <option value="vibrant">🎨 Vibrant / Sinematik</option>
      <option value="natural">🌿 Natural Murni</option>
     </select>
     <select id="mediaDuration" class="chat-model-select" title="Durasi Video" style="display:none">
      <option value="5">5 Detik</option>
      <option value="10">10 Detik</option>
     </select>
    </div>
   </div>
   <div class="chat-topbar-right">
    <span id="chatTurnsCount" class="badge" style="font-size:0.7rem;background:#090d16;border:1px solid #334155">0 pesan</span>
    <button class="btn btn-sm" style="background:#1e293b;color:#cbd5e1;border:1px solid #334155" onclick="clearTestChat()" title="Reset dan mulai sesi obrolan baru">🧹 Obrolan Baru</button>
   </div>
  </div>

  <div class="chat-messages" id="chatMessages">
   <div class="chat-empty-state" id="chatEmptyState">
    <div class="chat-empty-icon">🤖</div>
    <div class="chat-empty-title">Siap Menguji Model AI</div>
    <div class="chat-empty-desc">Ketik pertanyaan atau prompt untuk menguji teks chat, pembuatan gambar (Imagen 3 / DALL-E 3), atau video (Veo 2 / Gemini Video) secara interaktif dengan akun terpilih.</div>
   </div>
  </div>

  <div class="chat-input-bar" style="flex-direction:column;align-items:stretch;gap:6px">
   <div id="imageAttachmentPreviewBar" style="display:none;padding:6px 10px;background:#090d16;border:1px solid #334155;border-radius:8px;align-items:center;gap:10px;flex-wrap:wrap"></div>
   <div style="display:flex;align-items:center;gap:8px;width:100%">
    <label for="chatFileInput" class="btn btn-sm" style="background:#1e293b;color:#38bdf8;border:1px solid #334155;height:44px;display:flex;align-items:center;justify-content:center;padding:0 14px;cursor:pointer;border-radius:8px;flex-shrink:0" title="Attach Gambar / Foto (Screenshot / PNG / JPG / WebP)">
     <span style="font-size:1.15rem">📎</span>
    </label>
    <input type="file" id="chatFileInput" accept="image/*" multiple style="display:none" onchange="onFilesSelected(event)">
    <textarea id="chatInput" class="chat-textarea" rows="1" spellcheck="false" placeholder="Ketik pesan atau paste (Ctrl+V) screenshot gambar ke sini..." onkeydown="handleChatInputKey(event)" oninput="autoGrowTextarea(this)"></textarea>
    <button class="btn btn-primary btn-sm" id="btnSendChat" onclick="sendTestChat()" style="height:44px;padding:0 18px;display:flex;align-items:center;gap:6px;flex-shrink:0">
     <span>Kirim</span>
     <span>🚀</span>
    </button>
   </div>
  </div>
  <div class="chat-input-hints">
   <span>💡 <strong>Enter</strong> kirim / generate &bull; <strong>Ctrl+V</strong> paste gambar clipboard &bull; <strong>📎</strong> attach &bull; Zero-Disk RAM Stream</span>
   <span id="chatStatusText" style="color:#38bdf8"></span>
  </div>
 </div>
</div>

<div class="card" id="recentMediaCard">
 <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px;flex-wrap:wrap;gap:8px">
  <div style="display:flex;align-items:center;gap:8px">
   <h2 style="margin-bottom:0">🎨 Recent Media Showcase</h2>
   <span class="badge" style="background:#064e3b;color:#6ee7b7;border:1px solid #059669;font-size:0.7rem">Zero-Disk RAM Stream</span>
  </div>
  <button class="btn btn-sm" style="background:#1e293b;color:#38bdf8;border:1px solid #334155" onclick="loadRecentMedia()">🔄 Refresh</button>
 </div>
 <p style="font-size:0.8rem;color:#94a3b8;margin-bottom:12px">
  Media streaming langsung dari memori tanpa memenuhi disk komputer (Zero-Disk Memory Streaming). Klik untuk memutar video atau zoom gambar resolusi penuh.
 </p>
 <div class="media-gallery-grid" id="recentMediaGrid">
  <div style="grid-column:1/-1;text-align:center;padding:24px;color:#64748b;font-size:0.85rem">Belum ada media yang di-generate dalam sesi ini. Coba buat gambar atau video di atas!</div>
 </div>
</div>

<div class="card">
 <h2>⚙️ Settings</h2>
 <div class="form-grid">
  <div class="form-group">
   <label>Port</label>
   <input id="settingsPort" type="number" placeholder="18899">
  </div>
  <div class="form-group">
   <label>Auth Token (optional)</label>
   <input id="settingsAuth" type="password" placeholder="Bearer token for API access">
  </div>
 </div>
 <div class="btn-group">
  <button class="btn btn-primary btn-sm" onclick="saveSettings()">Save Settings</button>
 </div>
</div>
</div>

<div class="toast" id="toast"></div>

<!-- Image / Media Lightbox Modal -->
<div class="lightbox-modal" id="mediaLightbox" onclick="closeLightbox(event)">
 <div class="lightbox-content">
  <button class="lightbox-close" onclick="closeLightbox(event)">&times;</button>
  <img id="lightboxImg" src="" alt="" class="lightbox-image">
  <div id="lightboxCaption" class="lightbox-caption"></div>
 </div>
</div>

<script>
const API = '';

const HINTS = {
	chatgpt: 'Support direct paste from Cookie-Editor (TSV table, JSON, or header string). Auto-merges session-token.0 and .1!',
	claude: 'Paste sessionKey cookie or full cookie header from claude.ai',
	perplexity: 'Paste __Secure-next-auth.session-token or full cookie header from perplexity.ai',
	grok: 'Paste sso=...; sso-rw=... or Cookie-Editor export from grok.com',
	deepseek: 'Paste userToken from chat.deepseek.com Local Storage or Cookie',
	gemini: 'Paste __Secure-1PSID (optional: __Secure-1PSIDTS) or Cookie-Editor export from gemini.google.com',
	poe: 'Paste p-b cookie from poe.com',
	kimi: 'Paste access_token (from Local Storage) or cookie from kimi.ai',
	qwen: 'Paste token or cookie from chat.qwenlm.ai',
	custom: 'Format: BASE_URL|API_KEY (e.g. https://api.groq.com/openai/v1|gsk_... or http://localhost:11434/v1 for Ollama)',
};

const MODELS = {
	chatgpt: 'gpt-4o',
	claude: 'claude-3-5-sonnet',
	perplexity: 'best',
	grok: 'grok-2',
	deepseek: 'deepseek-chat',
	gemini: 'gemini-2.0-flash',
	poe: 'Claude-3.5-Sonnet',
	kimi: 'kimi-k1.5',
	qwen: 'qwen-max',
	custom: 'default',
};

function updateHint() {
 const p = document.getElementById('newProvider').value;
 document.getElementById('cookieHint').textContent = HINTS[p] || '';
 document.getElementById('newModel').value = MODELS[p] || '';
}

function toast(msg, type) {
 const t = document.getElementById('toast');
 t.textContent = msg;
 t.className = 'toast toast-' + (type || 'info') + ' show';
 setTimeout(() => t.className = 'toast', 3500);
}

function toggleAddForm(force) {
 const f = document.getElementById('addForm');
 const isHidden = f.style.display === 'none' || window.getComputedStyle(f).display === 'none';
 const shouldShow = typeof force === 'boolean' ? force : isHidden;
 f.style.display = shouldShow ? 'block' : 'none';
 if (shouldShow) {
  updateHint();
  const nameInput = document.getElementById('newName');
  if (nameInput) nameInput.focus();
 }
}

async function refreshAccounts() {
 try {
  const r = await fetch(API + '/api/accounts');
  const accounts = await r.json();
  window.allAccounts = accounts;
  populatePlaygroundAccounts();

  document.getElementById('totalAccounts').textContent = accounts.length;
  document.getElementById('activeAccounts').textContent = accounts.filter(a => a.status === 'active').length;

  const providers = new Set(accounts.map(a => a.provider));
  document.getElementById('totalProviders').textContent = providers.size;

  // Calculate unique models
  const uniqueModels = new Set();
  accounts.forEach(a => {
    if (a.defaultModel) uniqueModels.add(a.defaultModel);
    if (Array.isArray(a.availableModels)) {
      a.availableModels.forEach(m => uniqueModels.add(m));
    }
  });
  document.getElementById('totalModels').textContent = uniqueModels.size;

  const list = document.getElementById('accountsList');
  if (accounts.length === 0) {
   list.innerHTML = '<p style="color:#94a3b8;text-align:center;padding:24px">No accounts yet. Click "+ Add Account" to get started.</p>';
   return;
  }

  list.innerHTML = accounts.map(a => {
    let emailHtml = '';
    if (a.email) {
      const isPlus = a.plan && (a.plan.toLowerCase() === 'plus' || a.plan.toLowerCase() === 'pro' || a.plan.toLowerCase() === 'team');
      const planBadge = a.plan ?
        '<span style="font-size:0.7rem;padding:2px 8px;border-radius:12px;font-weight:700;' + (isPlus ? 'background:#065f46;color:#6ee7b7;border:1px solid #059669' : 'background:#1e293b;color:#94a3b8;border:1px solid #334155') + '">' + a.plan.toUpperCase() + '</span>' : '';
      emailHtml = '<div style="display:flex;align-items:center;gap:6px;margin:3px 0 5px 0">' +
        '<span style="color:#38bdf8;font-size:0.85rem;font-weight:600">✉️ ' + a.email + '</span>' +
        planBadge +
        '</div>';
    }

    let modelsBadgesHtml = '';
    if (Array.isArray(a.availableModels) && a.availableModels.length > 0) {
      modelsBadgesHtml = '<div class="models-box">' +
        '<span class="models-title">Available Models (' + a.availableModels.length + '):</span>' +
        a.availableModels.map(m => '<span class="badge-model">' + m + '</span>').join('') +
        '</div>';
    } else {
      modelsBadgesHtml = '<div class="models-box">' +
        '<span class="models-title" style="color:#64748b">No detected models yet. Click "⚡ Detect Models" to probe.</span>' +
        '</div>';
    }

    return '<div class="account-item">' +
     '<div class="account-item-header">' +
      '<div class="account-info">' +
       '<span class="status-dot status-' + a.status + '"></span>' +
       '<span class="provider-badge provider-' + a.provider + '">' + a.provider + '</span>' +
       '<div><div style="font-size:1.05rem;font-weight:bold;color:#f8fafc">' + (a.name || a.email || a.provider) + '</div>' +
       emailHtml +
       '<div class="account-meta">Default: <code>' + (a.defaultModel || 'auto') + '</code> · Cookie: ' + a.cookieLen + ' chars · Status: ' + a.status + (a.lastError ? ' (' + a.lastError + ')' : '') + '</div></div>' +
      '</div>' +
      '<div class="btn-group">' +
       '<button class="btn btn-detect btn-sm" onclick="detectModels(\'' + a.id + '\')">⚡ Detect Models</button>' +
       '<button class="btn btn-sm" style="background:#475569;color:#fff" onclick="editModelsPrompt(\'' + a.id + '\')">✏️ Edit</button>' +
       '<button class="btn btn-success btn-sm" onclick="testAccount(\'' + a.id + '\')">Test</button>' +
       '<button class="btn btn-danger btn-sm" onclick="deleteAccount(\'' + a.id + '\')">Delete</button>' +
      '</div>' +
     '</div>' +
     modelsBadgesHtml +
    '</div>';
  }).join('');
 } catch (e) {
  toast('Failed to load accounts: ' + e.message, 'error');
 }
}

let detectedAccountInfo = null;
let cookieProbeTimer = null;

function onCookieInput() {
 clearTimeout(cookieProbeTimer);
 const cookie = document.getElementById('newCookie').value.trim();
 const statusDiv = document.getElementById('cookieDetectStatus');
 if (!cookie || cookie.length < 20) {
  if (statusDiv) statusDiv.style.display = 'none';
  detectedAccountInfo = null;
  return;
 }

 if (statusDiv) {
  statusDiv.style.display = 'block';
  statusDiv.style.background = '#1e293b';
  statusDiv.style.border = '1px solid #334155';
  statusDiv.style.color = '#94a3b8';
  statusDiv.innerHTML = '<span class="spin">⏳</span> Mendeteksi email &amp; status akun dari cookie...';
 }

 cookieProbeTimer = setTimeout(async () => {
  const provider = document.getElementById('newProvider').value;
  try {
   const res = await fetch(API + '/api/accounts/probe', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ provider, cookie })
   });
   const data = await res.json();
   if (data.email) {
    detectedAccountInfo = data;
    const nameInput = document.getElementById('newName');
    if (!nameInput.value || nameInput.dataset.autofilled === 'true') {
     nameInput.value = data.email;
     nameInput.dataset.autofilled = 'true';
    }
    const modelInput = document.getElementById('newModel');
    if (!modelInput.value && data.defaultModel) {
     modelInput.value = data.defaultModel;
    }
    if (statusDiv) {
     const planBadge = data.plan ? '<span style="background:#022c22;color:#6ee7b7;border:1px solid #059669;padding:2px 6px;border-radius:4px;font-weight:bold;margin-left:6px">' + data.plan.toUpperCase() + '</span>' : '';
     statusDiv.style.background = '#064e3b';
     statusDiv.style.border = '1px solid #059669';
     statusDiv.style.color = '#a7f3d0';
     statusDiv.innerHTML = '✅ <strong>Akun Terdeteksi:</strong> <span style="color:#38bdf8">' + data.email + '</span> ' + planBadge;
    }
    toast('Email terdeteksi: ' + data.email, 'success');
   } else if (data.error) {
    if (statusDiv) {
     statusDiv.style.background = '#450a0a';
     statusDiv.style.border = '1px solid #dc2626';
     statusDiv.style.color = '#fca5a5';
     statusDiv.innerHTML = '⚠️ Gagal membaca sesi: ' + data.error;
    }
   } else {
    if (statusDiv) {
     statusDiv.style.background = '#1e293b';
     statusDiv.style.border = '1px solid #334155';
     statusDiv.style.color = '#cbd5e1';
     statusDiv.innerHTML = 'ℹ️ Cookie terformat (' + cookie.length + ' chars)';
    }
   }
  } catch(e) {
   if (statusDiv) {
    statusDiv.style.background = '#450a0a';
    statusDiv.style.border = '1px solid #dc2626';
    statusDiv.style.color = '#fca5a5';
    statusDiv.innerHTML = '⚠️ Error probe: ' + e.message;
   }
  }
 }, 350);
}

async function addAccount() {
 const provider = document.getElementById('newProvider').value;
 const name = document.getElementById('newName').value;
 const model = document.getElementById('newModel').value;
 const cookie = document.getElementById('newCookie').value.trim();
 if (!cookie) { toast('Cookie / Token is required', 'error'); return; }

 toast('Saving account...', 'info');
 try {
  const payload = {
   provider,
   name: name || (detectedAccountInfo && detectedAccountInfo.email) || provider,
   defaultModel: model,
   cookie,
   email: (detectedAccountInfo && detectedAccountInfo.email) || '',
   plan: (detectedAccountInfo && detectedAccountInfo.plan) || ''
  };
  const r = await fetch(API + '/api/accounts', {
   method: 'POST',
   headers: {'Content-Type': 'application/json'},
   body: JSON.stringify(payload)
  });
  if (r.ok) {
   toast('Account added! Probing models in background...', 'success');
   toggleAddForm(false);
   document.getElementById('newCookie').value = '';
   document.getElementById('newName').value = '';
   document.getElementById('newName').dataset.autofilled = 'false';
   const st = document.getElementById('cookieDetectStatus');
   if (st) st.style.display = 'none';
   detectedAccountInfo = null;
   setTimeout(refreshAccounts, 1500);
  } else {
   const err = await r.json();
   toast(err.error || 'Failed to add', 'error');
  }
 } catch (e) { toast(e.message, 'error'); }
}

async function detectModels(id) {
 toast('Probing available models...', 'info');
 try {
  const r = await fetch(API + '/api/accounts/models/' + id, {method: 'POST'});
  const data = await r.json();
  if (data.status === 'success' && data.models && data.models.length > 0) {
   toast('Detected ' + data.models.length + ' models: ' + data.models.slice(0, 4).join(', ') + (data.models.length > 4 ? '...' : ''), 'success');
  } else if (data.status === 'error') {
   toast('Detection error: ' + (data.error || 'could not fetch models'), 'error');
  } else {
   toast('Finished probing models.', 'info');
  }
  refreshAccounts();
 } catch (e) { toast('Detect failed: ' + e.message, 'error'); }
}

async function editModelsPrompt(id) {
 try {
  const r = await fetch(API + '/api/accounts');
  const accounts = await r.json();
  const acc = accounts.find(a => a.id === id);
  if (!acc) return;
  const current = (acc.availableModels || []).join(', ');
  const input = prompt('Edit model list for ' + (acc.name || acc.provider) + ' (comma-separated):', current);
  if (input === null) return;
  const models = input.split(',').map(m => m.trim()).filter(Boolean);
  acc.availableModels = models;
  const putRes = await fetch(API + '/api/accounts/' + id, {
    method: 'PUT',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(acc)
  });
  if (putRes.ok) {
    toast('Models updated (' + models.length + ' models)', 'success');
    refreshAccounts();
  } else {
    toast('Failed to update models', 'error');
  }
 } catch(e) { toast('Error: ' + e.message, 'error'); }
}

async function testAccount(id) {
 toast('Testing connection...', 'info');
 try {
  const r = await fetch(API + '/api/accounts/test/' + id, {method: 'POST'});
  const data = await r.json();
  if (data.status === 'active') {
   const mCount = (data.models && data.models.length) ? ' (' + data.models.length + ' models detected)' : '';
   toast('Connection OK!' + mCount, 'success');
  } else {
   toast('Test failed: ' + (data.error || 'unknown'), 'error');
  }
  refreshAccounts();
 } catch (e) { toast(e.message, 'error'); }
}

async function deleteAccount(id) {
 if (!confirm('Delete this account?')) return;
 try {
  await fetch(API + '/api/accounts/' + id, {method: 'DELETE'});
  toast('Account deleted', 'success');
  refreshAccounts();
 } catch (e) { toast(e.message, 'error'); }
}

async function loadSettings() {
 try {
  const r = await fetch(API + '/api/config');
  const cfg = await r.json();
  window.cachedConfig = cfg;
  document.getElementById('settingsPort').value = cfg.port || 18899;
  document.getElementById('settingsAuth').value = cfg.authToken || '';
  updateGuideSnippets();
 } catch (e) {}
}

async function saveSettings() {
 const port = parseInt(document.getElementById('settingsPort').value) || 18899;
 const authToken = document.getElementById('settingsAuth').value;
 try {
  const r = await fetch(API + '/api/config', {
   method: 'PUT',
   headers: {'Content-Type': 'application/json'},
   body: JSON.stringify({port, authToken})
  });
  if (r.ok) {
   window.cachedConfig = {port, authToken};
   updateGuideSnippets();
   toast('Settings saved! Restart to apply port change.', 'success');
  } else toast('Failed to save settings', 'error');
 } catch (e) { toast(e.message, 'error'); }
}

async function populateChatModels() {
 try {
  const r = await fetch(API + '/v1/models');
  const data = await r.json();
  if (data && Array.isArray(data.data) && data.data.length > 0) {
   window.cachedModels = data.data.map(m => m.id);
   updateGuideSnippets(window.cachedModels);
  } else {
   window.cachedModels = ['gpt-5-6-instant', 'gpt-5-6', 'gemini-2.5-flash', 'claude-3-7-sonnet', 'auto'];
  }
  updatePlaygroundModelsForMode();
 } catch (e) {
  window.cachedModels = ['gpt-5-6-instant', 'gpt-5-6', 'gemini-2.5-flash', 'claude-3-7-sonnet', 'auto'];
  updatePlaygroundModelsForMode();
 }
}

let playgroundMode = 'chat';

function populatePlaygroundAccounts() {
 const sel = document.getElementById('chatAccount');
 if (!sel) return;
 const savedVal = localStorage.getItem('selectedPlaygroundAccount');
 const prevVal = sel.value || savedVal || '';

 sel.innerHTML = '<option value="">🎯 Akun: Auto Round-Robin</option>';
 const accs = window.allAccounts || [];
 accs.filter(a => a.enabled !== false).forEach(a => {
  const opt = document.createElement('option');
  opt.value = a.id;
  const emailPart = a.email ? ' · ' + a.email : '';
  const planPart = a.plan ? ' [' + a.plan.toUpperCase() + ']' : '';
  opt.textContent = (a.status === 'active' ? '🟢 ' : '⚪ ') + (a.name || a.id) + emailPart + planPart + ' (' + (a.provider || '').toUpperCase() + ')';
  sel.appendChild(opt);
 });
 if (prevVal && Array.from(sel.options).some(o => String(o.value) === String(prevVal))) {
  sel.value = prevVal;
 }
 updatePlaygroundModelsForMode();
}

function switchPlaygroundMode(mode) {
 playgroundMode = mode;
 const btnChat = document.getElementById('modeChatBtn');
 const btnImg = document.getElementById('modeImageBtn');
 const btnVid = document.getElementById('modeVideoBtn');
 if (btnChat) btnChat.classList.toggle('active', mode === 'chat');
 if (btnImg) btnImg.classList.toggle('active', mode === 'image');
 if (btnVid) btnVid.classList.toggle('active', mode === 'video');

 const mediaRow = document.getElementById('mediaOptionsRow');
 const durSel = document.getElementById('mediaDuration');
 const styleSel = document.getElementById('mediaStyle');
 const chatInput = document.getElementById('chatInput');
 const sendBtn = document.getElementById('btnSendChat');

 if (mode === 'chat') {
  if (mediaRow) mediaRow.style.display = 'none';
  if (durSel) durSel.style.display = 'none';
  if (styleSel) styleSel.style.display = 'none';
  if (chatInput) chatInput.placeholder = 'Ketik pesan untuk AI... (Enter untuk kirim, Shift+Enter untuk baris baru)';
  if (sendBtn) sendBtn.innerHTML = '<span>Kirim</span> <span>🚀</span>';
 } else if (mode === 'image') {
  if (mediaRow) mediaRow.style.display = 'flex';
  if (durSel) durSel.style.display = 'none';
  if (styleSel) styleSel.style.display = 'inline-block';
  if (chatInput) chatInput.placeholder = 'Deskripsikan gambar yang ingin dibuat... (contoh: Kucing cyberpunk memakai kacamata neon di Tokyo)';
  if (sendBtn) sendBtn.innerHTML = '<span>Generate Gambar</span> <span>🎨</span>';
 } else if (mode === 'video') {
  if (mediaRow) mediaRow.style.display = 'flex';
  if (styleSel) styleSel.style.display = 'none';
  if (durSel) durSel.style.display = 'inline-block';
  if (chatInput) chatInput.placeholder = 'Deskripsikan video yang ingin dibuat... (contoh: Seekor kucing lucu sedang menari di pantai saat sunset)';
  if (sendBtn) sendBtn.innerHTML = '<span>Generate Video</span> <span>🎬</span>';
 }

 updatePlaygroundModelsForMode();
}

function updatePlaygroundModelsForMode() {
 const sel = document.getElementById('chatModel');
 if (!sel) return;
 const prevVal = sel.value;
 const currentAccId = document.getElementById('chatAccount') ? document.getElementById('chatAccount').value : '';
 const currentAcc = (window.allAccounts || []).find(a => String(a.id) === String(currentAccId));

 sel.innerHTML = '';
 if (playgroundMode === 'chat') {
  if (currentAcc) {
   let models = [];
   const prov = (currentAcc.provider || '').toLowerCase();
   if (prov === 'gemini') {
    const raw = (currentAcc.availableModels && currentAcc.availableModels.length > 0)
      ? currentAcc.availableModels
      : ['gemini-4-argon', 'gemini-4-argon-high', 'gemini-4-argon-medium', 'gemini-4-argon-low', 'gemini-3.1-pro', 'gemini-3.8-flash', 'gemini-3.5-flash-lite', 'imagen-3', 'nano-banana', 'gemini-auto'];
    models = raw.filter(m => {
     const ml = m.toLowerCase();
     return ml.includes('gemini') || ml.includes('argon') || ml.includes('pro') || ml.includes('flash') || ml.includes('imagen') || ml.includes('banana');
    });
   } else if (prov === 'chatgpt') {
    const raw = (currentAcc.availableModels && currentAcc.availableModels.length > 0)
      ? currentAcc.availableModels
      : ['gpt-6-thinking', 'gpt-6', 'gpt-6-instant', 'gpt-5-6-thinking', 'gpt-5-6', 'gpt-4o', 'auto'];
    models = raw.filter(m => {
     const ml = m.toLowerCase();
     return !ml.includes('gemini') && !ml.includes('argon') && !ml.includes('banana');
    });
   } else {
    models = currentAcc.availableModels || ['auto'];
   }

   models.forEach(m => {
    const opt = document.createElement('option');
    opt.value = m;
    opt.textContent = m;
    sel.appendChild(opt);
   });

   if (prevVal && Array.from(sel.options).some(o => o.value === prevVal)) {
    sel.value = prevVal;
   } else if (prov === 'gemini') {
    sel.value = models.includes('gemini-4-argon') ? 'gemini-4-argon' : (models.includes('gemini-3.1-pro') ? 'gemini-3.1-pro' : models[0]);
   } else if (prov === 'chatgpt') {
    sel.value = models.includes('gpt-6-thinking') ? 'gpt-6-thinking' : models[0];
   }
  } else {
   // Auto Round-Robin: Group models cleanly by provider
   const grpChatGPT = document.createElement('optgroup');
   grpChatGPT.label = '🤖 ChatGPT / OpenAI Models';
   const grpGemini = document.createElement('optgroup');
   grpGemini.label = '✨ Google Gemini Models';

   const all = (window.cachedModels && window.cachedModels.length > 0)
     ? window.cachedModels
     : ['gpt-6-thinking', 'gpt-6', 'gpt-5-6', 'gemini-4-argon', 'gemini-3.1-pro', 'auto'];

   all.forEach(m => {
    const opt = document.createElement('option');
    opt.value = m;
    opt.textContent = m;
    const ml = m.toLowerCase();
    if (ml.includes('gemini') || ml.includes('argon') || ml.includes('banana') || ml.includes('imagen')) {
     grpGemini.appendChild(opt);
    } else {
     grpChatGPT.appendChild(opt);
    }
   });

   if (grpChatGPT.children.length > 0) sel.appendChild(grpChatGPT);
   if (grpGemini.children.length > 0) sel.appendChild(grpGemini);

   if (prevVal && Array.from(sel.options).some(o => o.value === prevVal)) {
    sel.value = prevVal;
   } else {
    sel.value = 'gpt-6-thinking';
   }
  }
 } else if (playgroundMode === 'image') {
  let imgModels = [];
  if (currentAcc && currentAcc.provider === 'chatgpt') {
   imgModels = [
    { id: 'gpt-6-thinking', name: 'gpt-6-thinking (Flagship Realistis & Cerdas ⭐)' },
    { id: 'gpt-5-6-thinking', name: 'gpt-5-6-thinking (GPT-5.6 Sol Thinking)' },
    { id: 'gpt-5-6', name: 'gpt-5-6 (GPT-5.6 Sol Standard)' },
    { id: 'dall-e-3', name: 'dall-e-3 (DALL-E 3 Direct)' },
    { id: 'auto', name: 'auto (Pilih Otomatis)' }
   ];
  } else if (currentAcc && currentAcc.provider === 'gemini') {
   imgModels = [
    { id: 'imagen-3', name: 'imagen-3 (Imagen 3 - Gemini HD ⭐)' },
    { id: 'nano-banana', name: 'nano-banana (Nano Banana - Visual Image Pipeline)' },
    { id: 'gemini-4-argon', name: 'gemini-4-argon (4 Argon - Multimodal Visual)' },
    { id: 'imagen-3.0-generate-002', name: 'imagen-3.0-generate-002 (Gemini HQ Direct)' },
    { id: 'auto', name: 'auto (Pilih Otomatis)' }
   ];
  } else {
   // Auto Round-Robin Image: cleanly grouped by provider
   const grpChatGPT = document.createElement('optgroup');
   grpChatGPT.label = '🤖 OpenAI / ChatGPT Image Models';
   const grpGemini = document.createElement('optgroup');
   grpGemini.label = '✨ Google Gemini Image Models';

   [
    { id: 'gpt-6-thinking', name: 'gpt-6-thinking (ChatGPT Flagship High-Res ⭐)' },
    { id: 'dall-e-3', name: 'dall-e-3 (DALL-E 3 Direct)' }
   ].forEach(m => {
    const opt = document.createElement('option');
    opt.value = m.id;
    opt.textContent = m.name;
    grpChatGPT.appendChild(opt);
   });

   [
    { id: 'imagen-3', name: 'imagen-3 (Imagen 3 - Gemini HD ⭐)' },
    { id: 'nano-banana', name: 'nano-banana (Nano Banana - Visual Image Pipeline)' },
    { id: 'gemini-4-argon', name: 'gemini-4-argon (4 Argon Visual)' }
   ].forEach(m => {
    const opt = document.createElement('option');
    opt.value = m.id;
    opt.textContent = m.name;
    grpGemini.appendChild(opt);
   });

   sel.appendChild(grpChatGPT);
   sel.appendChild(grpGemini);
   const optAuto = document.createElement('option');
   optAuto.value = 'auto';
   optAuto.textContent = 'auto (Pilih Otomatis)';
   sel.appendChild(optAuto);

   if (prevVal && Array.from(sel.options).some(o => o.value === prevVal)) {
    sel.value = prevVal;
   } else {
    sel.value = 'gpt-6-thinking';
   }
   return;
  }

  imgModels.forEach(m => {
   const opt = document.createElement('option');
   opt.value = m.id;
   opt.textContent = m.name;
   sel.appendChild(opt);
  });
  if (currentAcc && currentAcc.provider === 'chatgpt') {
   sel.value = 'gpt-6-thinking';
  } else if (currentAcc && currentAcc.provider === 'gemini') {
   sel.value = 'imagen-3';
  } else if (prevVal && Array.from(sel.options).some(o => o.value === prevVal)) {
   sel.value = prevVal;
  }
 } else if (playgroundMode === 'video') {
  let vidModels = [];
  if (currentAcc && currentAcc.provider === 'gemini') {
   vidModels = [
    { id: 'veo-2', name: 'veo-2 (Google Veo 2 / Gemini Cinematic ⭐)' },
    { id: 'gemini-video', name: 'gemini-video (Gemini Cinematic)' },
    { id: 'auto', name: 'auto (Pilih Otomatis)' }
   ];
  } else if (currentAcc && currentAcc.provider === 'chatgpt') {
   vidModels = [
    { id: 'sora', name: 'sora (OpenAI Sora Video ⭐)' },
    { id: 'auto', name: 'auto (Pilih Otomatis)' }
   ];
  } else {
   vidModels = [
    { id: 'veo-2', name: 'veo-2 (Google Veo / Gemini Video)' },
    { id: 'sora', name: 'sora (OpenAI Video)' },
    { id: 'runway-gen-3', name: 'runway-gen-3 (Runway Alpha)' },
    { id: 'kling-v1.5', name: 'kling-v1.5 (Kling AI)' },
    { id: 'luma-ray-2', name: 'luma-ray-2 (Dream Machine)' },
    { id: 'auto', name: 'auto (Pilih Otomatis)' }
   ];
  }
  vidModels.forEach(m => {
   const opt = document.createElement('option');
   opt.value = m.id;
   opt.textContent = m.name;
   sel.appendChild(opt);
  });
  if (currentAcc && currentAcc.provider === 'gemini') {
   sel.value = 'veo-2';
  } else if (currentAcc && currentAcc.provider === 'chatgpt') {
   sel.value = 'sora';
  } else if (prevVal && Array.from(sel.options).some(o => o.value === prevVal)) {
   sel.value = prevVal;
  }
 }

 const switcher = document.getElementById('geminiSwitcher');
 const cgptSwitcher = document.getElementById('chatgptEffortSwitcher');
 const activeProv = currentAcc ? (currentAcc.provider || '').toLowerCase() : '';

 if (playgroundMode === 'chat' && activeProv === 'gemini') {
  if (switcher) switcher.style.display = 'inline-block';
  if (cgptSwitcher) cgptSwitcher.style.display = 'none';
  sel.style.display = 'none';
  syncGeminiMenuState();
 } else if (playgroundMode === 'chat' && activeProv === 'chatgpt') {
  if (switcher) switcher.style.display = 'none';
  if (cgptSwitcher) cgptSwitcher.style.display = 'inline-block';
  sel.style.display = 'inline-block';
  syncChatGPTEffortState();
 } else {
  if (switcher) switcher.style.display = 'none';
  if (cgptSwitcher) cgptSwitcher.style.display = 'none';
  sel.style.display = 'inline-block';
 }
}

function syncGeminiMenuState() {
 const sel = document.getElementById('chatModel');
 if (!sel) return;
 const val = (sel.value || 'gemini-4-argon').toLowerCase();

 ['check_35', 'check_38', 'check_31', 'check_4', 'check_r_low', 'check_r_med', 'check_r_high'].forEach(id => {
  const el = document.getElementById(id);
  if (el) el.textContent = '';
 });

 const labelEl = document.getElementById('geminiActiveModelLabel');

 if (val.includes('3.5-flash-lite') || val.includes('flash-lite')) {
  if (document.getElementById('check_35')) document.getElementById('check_35').textContent = '✓';
  if (labelEl) labelEl.textContent = '3.5 Flash-Lite';
 } else if (val.includes('3.8-flash')) {
  if (document.getElementById('check_38')) document.getElementById('check_38').textContent = '✓';
  if (labelEl) labelEl.textContent = '3.8 Flash';
 } else if (val.includes('3.1-pro')) {
  if (document.getElementById('check_31')) document.getElementById('check_31').textContent = '✓';
  if (labelEl) labelEl.textContent = '3.1 Pro';
 } else {
  if (document.getElementById('check_4')) document.getElementById('check_4').textContent = '✓';
  if (val.includes('low') || val.includes('rendah')) {
   if (document.getElementById('check_r_low')) document.getElementById('check_r_low').textContent = '✓';
   if (labelEl) labelEl.textContent = '4 Argon (Rendah)';
  } else if (val.includes('medium') || val.includes('sedang')) {
   if (document.getElementById('check_r_med')) document.getElementById('check_r_med').textContent = '✓';
   if (labelEl) labelEl.textContent = '4 Argon (Sedang)';
  } else if (val.includes('high') || val.includes('tinggi')) {
   if (document.getElementById('check_r_high')) document.getElementById('check_r_high').textContent = '✓';
   if (labelEl) labelEl.textContent = '4 Argon (Tinggi)';
  } else {
   if (document.getElementById('check_r_high')) document.getElementById('check_r_high').textContent = '✓';
   if (labelEl) labelEl.textContent = '4 Argon';
  }
 }
}

function selectGeminiModel(modelId, label, reasoning) {
 const sel = document.getElementById('chatModel');
 if (sel) {
  let exists = false;
  for (let i = 0; i < sel.options.length; i++) {
   if (sel.options[i].value === modelId) {
    exists = true;
    break;
   }
  }
  if (!exists) {
   const opt = document.createElement('option');
   opt.value = modelId;
   opt.textContent = modelId;
   sel.appendChild(opt);
  }
  sel.value = modelId;
 }
 syncGeminiMenuState();
 closeGeminiMenu();
}

function toggleGeminiMenu(e) {
 if (e) e.stopPropagation();
 const menu = document.getElementById('geminiMenu');
 if (!menu) return;
 menu.classList.toggle('show');
}

function closeGeminiMenu() {
 const menu = document.getElementById('geminiMenu');
 if (menu) menu.classList.remove('show');
}

function syncChatGPTEffortState() {
 const sel = document.getElementById('chatModel');
 if (!sel) return;
 const val = (sel.value || '').toLowerCase();
 const range = document.getElementById('cgptEffortRange');
 const label = document.getElementById('cgptEffortLabel');

 if (val.includes('instant') || val.includes('low')) {
  if (range) range.value = 0;
  if (label) label.textContent = 'Instant (Low)';
 } else if (val.includes('thinking') || val.includes('high')) {
  if (range) range.value = 2;
  if (label) label.textContent = 'High';
 } else {
  if (range) range.value = 1;
  if (label) label.textContent = 'Medium';
 }
}

function onChatGPTEffortSlide(step) {
 step = parseInt(step, 10);
 const range = document.getElementById('cgptEffortRange');
 if (range) range.value = step;
 const label = document.getElementById('cgptEffortLabel');
 const sel = document.getElementById('chatModel');
 if (!sel) return;

 const currentVal = (sel.value || 'gpt-6-thinking').toLowerCase();
 let base = 'gpt-6';
 if (currentVal.includes('5-6')) {
  base = 'gpt-5-6';
 }

 let target = base;
 if (step === 0) {
  target = base + '-instant';
  if (label) label.textContent = 'Instant (Low)';
 } else if (step === 1) {
  target = base;
  if (label) label.textContent = 'Medium';
 } else {
  target = base + '-thinking';
  if (label) label.textContent = 'High';
 }

 let exists = false;
 for (let i = 0; i < sel.options.length; i++) {
  if (sel.options[i].value === target) {
   exists = true;
   break;
  }
 }
 if (!exists) {
  const opt = document.createElement('option');
  opt.value = target;
  opt.textContent = target;
  sel.appendChild(opt);
 }
 sel.value = target;
}

function cycleChatGPTEffort() {
 const range = document.getElementById('cgptEffortRange');
 if (!range) return;
 const cur = parseInt(range.value, 10);
 const next = (cur + 1) % 3;
 onChatGPTEffortSlide(next);
}

function toggleChatGPTEffortMenu(e) {
 if (e) e.stopPropagation();
 const menu = document.getElementById('chatgptEffortMenu');
 if (!menu) return;
 menu.classList.toggle('show');
}

function closeChatGPTEffortMenu() {
 const menu = document.getElementById('chatgptEffortMenu');
 if (menu) menu.classList.remove('show');
}

document.addEventListener('click', function(e) {
 const wrap = document.getElementById('geminiSwitcher');
 if (wrap && !wrap.contains(e.target)) {
  closeGeminiMenu();
 }
 const cgptWrap = document.getElementById('chatgptEffortSwitcher');
 if (cgptWrap && !cgptWrap.contains(e.target)) {
  closeChatGPTEffortMenu();
 }
});

function onPlaygroundAccountChanged() {
 const sel = document.getElementById('chatAccount');
 if (sel) {
  try {
   localStorage.setItem('selectedPlaygroundAccount', sel.value);
  } catch(e){}
 }
 updatePlaygroundModelsForMode();
}

async function loadRecentMedia() {
 const grid = document.getElementById('recentMediaGrid');
 if (!grid) return;
 try {
  const r = await fetch(API + '/api/media/recent');
  const data = await r.json();
  if (data && Array.isArray(data.items) && data.items.length > 0) {
   grid.innerHTML = data.items.map(item => {
    const isVideo = item.type === 'video';
    const fileUrl = isVideo ? (API + '/v1/videos/' + item.id + '.mp4') : (API + '/v1/images/' + item.id + '.png');
    const badge = isVideo ? ('🎬 Video (' + (item.duration || 5) + 's)') : '🎨 Image';

    let previewHtml = '';
    if (isVideo) {
     previewHtml = '<video class="media-gallery-thumb" controls loop playsinline preload="metadata" src="' + fileUrl + '"></video>';
    } else {
     previewHtml = '<img class="media-gallery-thumb" src="' + fileUrl + '" onclick="openLightbox(\'' + fileUrl + '\', \'' + escapeHtml(item.prompt) + '\')" style="cursor:zoom-in" />';
    }

    return '<div class="media-gallery-item">' +
     previewHtml +
     '<div class="media-gallery-meta">' +
      '<div>' +
       '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:4px">' +
        '<span class="badge" style="background:#1e293b;color:#38bdf8;font-size:0.68rem">' + badge + '</span>' +
        '<span style="color:#64748b;font-size:0.68rem">' + escapeHtml(item.accountName || item.provider || 'AI') + '</span>' +
       '</div>' +
       '<div style="font-size:0.75rem;color:#cbd5e1;line-height:1.3;max-height:36px;overflow:hidden;text-overflow:ellipsis" title="' + escapeHtml(item.prompt) + '">' + escapeHtml(item.prompt) + '</div>' +
      '</div>' +
      '<div style="margin-top:8px;display:flex;gap:6px">' +
       '<a href="' + fileUrl + '?download=1" download="' + item.id + (isVideo ? '.mp4' : '.png') + '" class="btn btn-sm" style="flex:1;text-align:center;text-decoration:none;background:#2563eb;color:#fff;padding:4px 8px;font-size:0.72rem">⬇ Unduh</a>' +
       '<button class="btn btn-sm" style="background:#1e293b;color:#94a3b8;border:1px solid #334155;padding:4px 8px;font-size:0.72rem" onclick="copySnippet(\'' + fileUrl + '\', this)">📋</button>' +
      '</div>' +
     '</div>' +
    '</div>';
   }).join('');
  } else {
   grid.innerHTML = '<div style="grid-column:1/-1;text-align:center;padding:24px;color:#64748b;font-size:0.85rem">Belum ada media yang di-generate dalam sesi ini. Coba buat gambar atau video di atas!</div>';
  }
 } catch (e) {}
}

function escapeHtml(str) {
 return (str || '').replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#039;");
}

let chatHistory = [];

function autoGrowTextarea(el) {
 el.style.height = 'auto';
 el.style.height = Math.min(el.scrollHeight, 180) + 'px';
}

function handleChatInputKey(e) {
 if (e.key === 'Enter' && !e.shiftKey) {
  e.preventDefault();
  sendTestChat();
 }
}

function updateChatTurnsBadge() {
 const countEl = document.getElementById('chatTurnsCount');
 if (!countEl) return;
 const total = chatHistory.length;
 countEl.textContent = total + ' pesan' + (total > 0 ? ' (' + Math.ceil(total / 2) + ' putaran)' : '');
}

function clearTestChat() {
 chatHistory = [];
 window.attachedImages = [];
 renderAttachmentBar();
 const msgs = document.getElementById('chatMessages');
 if (msgs) {
  msgs.innerHTML = '<div class="chat-empty-state" id="chatEmptyState">' +
   '<div class="chat-empty-icon">🤖</div>' +
   '<div class="chat-empty-title">Siap Menguji Model AI</div>' +
   '<div class="chat-empty-desc">Ketik pertanyaan atau prompt untuk menguji endpoint /v1/chat/completions secara interaktif dengan riwayat percakapan bersambung &amp; pratinjau media generate.</div>' +
  '</div>';
 }
 const inp = document.getElementById('chatInput');
 if (inp) {
  inp.value = '';
  inp.style.height = '44px';
  inp.focus();
 }
 updateChatTurnsBadge();
 toast('Obrolan dibersihkan (sesi baru)', 'info');
}

function copyCodeBlock(btn) {
 const pre = btn.closest('.code-block-wrapper').querySelector('pre');
 if (!pre) return;
 copySnippet(pre.textContent, btn);
}

function openLightbox(url, caption) {
 const modal = document.getElementById('mediaLightbox');
 const img = document.getElementById('lightboxImg');
 const cap = document.getElementById('lightboxCaption');
 if (!modal || !img) return;
 img.src = url;
 img.alt = caption || 'Generated Media';
 if (cap) cap.textContent = caption || '';
 modal.classList.add('active');
}

function closeLightbox(e) {
 if (e && e.target && e.target.classList.contains('lightbox-image')) {
  return;
 }
 const modal = document.getElementById('mediaLightbox');
 if (modal) modal.classList.remove('active');
}

function renderMarkdown(text) {
 if (!text) return '';

 text = text.replace(/https?:\/\/googleusercontent\.com\/(?:image_generation_content|lmdx_content)\/[^\s]+/g, '').trim();

 const codeBlocks = [];
 let processed = text.replace(new RegExp('\\x60{3}([a-zA-Z0-9_-]*)\\n([\\s\\S]*?)\\x60{3}', 'g'), function(match, lang, code) {
  const id = '___CODEBLOCK_' + codeBlocks.length + '___';
  codeBlocks.push({lang: lang || 'code', code: code.trim()});
  return id;
 });

 const imageCards = [];
 processed = processed.replace(/!\[(.*?)\]\((https?:\/\/[^\s\)]+|\/[^\s\)]+)\)/g, function(match, alt, url) {
  const id = '___IMAGECARD_' + imageCards.length + '___';
  imageCards.push({alt: alt || 'Generated Image', url: url});
  return id;
 });

 processed = escapeHtml(processed);

 processed = processed.replace(/^### (.*$)/gim, '<h3 style="color:#38bdf8;margin:10px 0 6px;font-size:1.05rem">$1</h3>');
 processed = processed.replace(/^## (.*$)/gim, '<h2 style="color:#38bdf8;margin:12px 0 8px;font-size:1.15rem">$1</h2>');
 processed = processed.replace(/^# (.*$)/gim, '<h1 style="color:#38bdf8;margin:14px 0 10px;font-size:1.3rem">$1</h1>');

 processed = processed.replace(/^\> (.*$)/gim, '<blockquote style="border-left:3px solid #38bdf8;padding-left:12px;margin:8px 0;color:#94a3b8;font-style:italic">$1</blockquote>');

 processed = processed.replace(/\*\*\*(.*?)\*\*\*/g, '<strong><em>$1</em></strong>');
 processed = processed.replace(/\*\*(.*?)\*\*/g, '<strong style="color:#f1f5f9">$1</strong>');
 processed = processed.replace(/\*(.*?)\*/g, '<em>$1</em>');

 processed = processed.replace(new RegExp('\\x60([^\\x60]+)\\x60', 'g'), '<code style="background:#090d16;color:#38bdf8;padding:2px 6px;border-radius:4px;font-size:0.85em;border:1px solid #334155">$1</code>');

 processed = processed.replace(/\[(.*?)\]\((https?:\/\/[^\s\)]+)\)/g, '<a href="$2" target="_blank" style="color:#38bdf8;text-decoration:underline">$1</a>');

 processed = processed.replace(/^\s*[-*]\s+(.*$)/gim, '<li style="margin-left:18px;margin-bottom:4px">$1</li>');
 processed = processed.replace(/(<li.*<\/li>)/gim, '<ul style="margin:6px 0;padding-left:6px">$1</ul>');

 processed = processed.replace(/\n\n/g, '<div style="margin-bottom:10px"></div>');
 processed = processed.replace(/\n/g, '<br>');

 imageCards.forEach((img, i) => {
  const cleanName = (img.alt ? img.alt.replace(/[^a-zA-Z0-9_-]/g, '_') : 'generated-media') + '.png';
  const cardHtml = '<div class="media-showcase-card">' +
   '<div class="media-card-header">' +
    '<div style="display:flex;align-items:center;gap:8px">' +
     '<span style="font-size:1rem">🎨</span>' +
     '<span style="font-weight:600;color:#e2e8f0;font-size:0.85rem">' + escapeHtml(img.alt) + '</span>' +
    '</div>' +
    '<span class="media-badge-zero">⚡ Zero-Disk Streaming</span>' +
   '</div>' +
   '<div class="media-card-viewport" onclick="openLightbox(\'' + img.url + '\', \'' + escapeHtml(img.alt) + '\')">' +
    '<img src="' + img.url + '" alt="' + escapeHtml(img.alt) + '" loading="lazy" class="media-image-preview">' +
    '<div class="media-zoom-overlay">🔍 Klik untuk Perbesar Penuh</div>' +
   '</div>' +
   '<div class="media-card-actions">' +
    '<button class="btn-media btn-media-zoom" onclick="openLightbox(\'' + img.url + '\', \'' + escapeHtml(img.alt) + '\')">🔍 Layar Penuh</button>' +
    '<a href="' + img.url + '?download=1" download="' + cleanName + '" class="btn-media btn-media-download">⬇️ Unduh PNG Asli</a>' +
    '<button class="btn-media btn-media-copy" onclick="copySnippet(\'' + img.url + '\', this)">📋 Salin URL</button>' +
   '</div>' +
  '</div>';
  processed = processed.replace('___IMAGECARD_' + i + '___', cardHtml);
 });

 codeBlocks.forEach((cb, i) => {
  const codeHtml = '<div class="code-block-wrapper">' +
   '<div class="code-block-header">' +
    '<span class="code-block-lang">' + escapeHtml(cb.lang) + '</span>' +
    '<button class="code-copy-btn" onclick="copyCodeBlock(this)">Copy Code</button>' +
   '</div>' +
   '<pre class="code-block-body"><code>' + escapeHtml(cb.code) + '</code></pre>' +
  '</div>';
  processed = processed.replace('___CODEBLOCK_' + i + '___', codeHtml);
 });

 return processed;
}

window.attachedImages = [];

function onFilesSelected(event) {
  const files = event.target.files;
  if (!files || files.length === 0) return;
  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file.type.startsWith('image/')) continue;
    const reader = new FileReader();
    reader.onload = function(e) {
      addImageAttachment(e.target.result, file.name);
    };
    reader.readAsDataURL(file);
  }
  event.target.value = '';
}

function addImageAttachment(dataUrl, name) {
  window.attachedImages.push({ dataUrl, name: name || 'image.png' });
  renderAttachmentBar();
}

function removeAttachment(index) {
  window.attachedImages.splice(index, 1);
  renderAttachmentBar();
}

function renderAttachmentBar() {
  const bar = document.getElementById('imageAttachmentPreviewBar');
  if (!bar) return;
  if (window.attachedImages.length === 0) {
    bar.style.display = 'none';
    bar.innerHTML = '';
    return;
  }
  bar.style.display = 'flex';
  bar.innerHTML = window.attachedImages.map(function(att, i) {
    return '<div style="display:flex;align-items:center;gap:8px;background:#1e293b;border:1px solid #475569;border-radius:6px;padding:4px 8px;font-size:0.78rem;color:#e2e8f0">' +
      '<img src="' + att.dataUrl + '" style="width:34px;height:34px;object-fit:cover;border-radius:4px;border:1px solid #334155" />' +
      '<span style="max-width:130px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">' + escapeHtml(att.name) + '</span>' +
      '<button onclick="removeAttachment(' + i + ')" style="background:none;border:none;color:#f87171;font-weight:bold;cursor:pointer;padding:0 4px;font-size:1rem" title="Hapus gambar">✕</button>' +
    '</div>';
  }).join('') + '<span style="font-size:0.75rem;color:#38bdf8;margin-left:auto">' + window.attachedImages.length + ' gambar (' + (playgroundMode === 'image' ? '🖼️ Img2Img' : '👁️ Vision') + ')</span>';
}

// Global paste listener for screenshot clipboard
window.addEventListener('paste', function(e) {
  const items = (e.clipboardData || e.originalEvent?.clipboardData)?.items;
  if (!items) return;
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const blob = item.getAsFile();
      const reader = new FileReader();
      reader.onload = function(event) {
        addImageAttachment(event.target.result, 'screenshot_' + Date.now().toString().slice(-4) + '.png');
        toast('Screenshot clipboard terlampir 📎', 'success');
      };
      reader.readAsDataURL(blob);
    }
  }
});

async function sendTestChat() {
 const input = document.getElementById('chatInput');
 const text = input.value.trim();
 const model = document.getElementById('chatModel').value;
 const accId = document.getElementById('chatAccount') ? document.getElementById('chatAccount').value : '';
 if (!text && window.attachedImages.length === 0) { toast('Ketik pesan atau lampirkan gambar terlebih dahulu', 'error'); return; }

 const currentAttachments = [...window.attachedImages];
 window.attachedImages = [];
 renderAttachmentBar();

 const emptyState = document.getElementById('chatEmptyState');
 if (emptyState) emptyState.remove();

 const messagesContainer = document.getElementById('chatMessages');
 const btn = document.getElementById('btnSendChat');
 const statusText = document.getElementById('chatStatusText');
 const timeStr = new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});

 let attachedImgsHtml = '';
 if (currentAttachments.length > 0) {
   attachedImgsHtml = '<div style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:8px">' +
     currentAttachments.map(att => '<img src="' + att.dataUrl + '" style="max-width:180px;max-height:180px;border-radius:6px;border:1px solid #334155;object-fit:cover" />').join('') +
     '</div>';
 }

 const userAvatar = playgroundMode === 'video' ? '🎬' : (playgroundMode === 'image' ? '🎨' : 'U');
 const userRow = document.createElement('div');
 userRow.className = 'chat-message-row user';
 userRow.innerHTML = '<div class="chat-avatar user">' + userAvatar + '</div>' +
  '<div class="chat-bubble-container">' +
   '<div class="chat-bubble-meta">' +
    '<span>Anda (' + (playgroundMode === 'video' ? 'Video Prompt' : (playgroundMode === 'image' ? (currentAttachments.length > 0 ? 'Img2Img Edit' : 'Image Prompt') : (currentAttachments.length > 0 ? 'Vision Chat' : 'Chat'))) + ')</span>' +
    '<span class="chat-meta-time">' + timeStr + '</span>' +
   '</div>' +
   '<div class="chat-bubble">' + attachedImgsHtml + renderMarkdown(text || '(Gambar Terlampir)') + '</div>' +
  '</div>';
 messagesContainer.appendChild(userRow);

 input.value = '';
 input.style.height = '44px';
 messagesContainer.scrollTop = messagesContainer.scrollHeight;

 const loadingId = 'loading_' + Date.now();
 const assistAvatar = playgroundMode === 'video' ? '🎬' : (playgroundMode === 'image' ? '🎨' : '✨');
 const loadingText = playgroundMode === 'video' ? 'Sedang merender &amp; streaming video MP4 (Zero-Disk RAM)...' :
                     (playgroundMode === 'image' ? (currentAttachments.length > 0 ? 'Sedang mengedit gambar Image-to-Image (Zero-Disk RAM Stream)...' : 'Sedang membuat gambar (Zero-Disk Memory Stream)...') : 'Sedang berpikir &amp; menyusun respon...');

 const loadingRow = document.createElement('div');
 loadingRow.className = 'chat-message-row assistant';
 loadingRow.id = loadingId;
 loadingRow.innerHTML = '<div class="chat-avatar assistant">' + assistAvatar + '</div>' +
  '<div class="chat-bubble-container">' +
   '<div class="chat-bubble-meta">' +
    '<span class="chat-meta-model">' + escapeHtml(model) + '</span>' +
    '<span class="chat-meta-time">Memproses...</span>' +
   '</div>' +
   '<div class="chat-bubble">' +
    '<span class="typing-dots"><span></span><span></span><span></span></span>' +
    '<span style="margin-left:8px;font-size:0.8rem;color:#94a3b8">' + loadingText + '</span>' +
   '</div>' +
  '</div>';
 messagesContainer.appendChild(loadingRow);
 messagesContainer.scrollTop = messagesContainer.scrollHeight;

 btn.disabled = true;
 const t0 = Date.now();

 try {
  if (playgroundMode === 'image') {
   if (statusText) statusText.textContent = '🎨 Mengirim ke /v1/images/generations (' + model + ')...';
   const aspect = document.getElementById('mediaAspect') ? document.getElementById('mediaAspect').value : '1:1';
   let size = "1024x1024";
   if (aspect === "16:9") size = "1792x1024";
   else if (aspect === "9:16") size = "1024x1792";
   const style = document.getElementById('mediaStyle') ? document.getElementById('mediaStyle').value : 'none';

   const r = await fetch(API + '/v1/images/generations', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
     prompt: text || 'Enhance and recreate this image with high aesthetic fidelity',
     model: model,
     account_id: accId || undefined,
     size: size,
     style: style,
     attachments: currentAttachments.map(a => a.dataUrl),
     image: currentAttachments.length > 0 ? currentAttachments[0].dataUrl : undefined,
     response_format: 'url'
    })
   });
   const dur = ((Date.now() - t0) / 1000).toFixed(2);
   const data = await r.json();

   const loadingEl = document.getElementById(loadingId);
   if (loadingEl) loadingEl.remove();

   if (r.ok && data.data && data.data.length > 0) {
    const imgItem = data.data[0];
    const imgUrl = imgItem.url;
    const revised = imgItem.revised_prompt || text;

    const assistantRow = document.createElement('div');
    assistantRow.className = 'chat-message-row assistant';
    assistantRow.innerHTML = '<div class="chat-avatar assistant">🎨</div>' +
     '<div class="chat-bubble-container">' +
      '<div class="chat-bubble-meta">' +
       '<span class="chat-meta-model">' + escapeHtml(model) + '</span>' +
       '<span class="chat-meta-time">' + timeStr + ' &bull; ' + dur + 's</span>' +
       '<span class="media-badge-zero">⚡ Zero-Disk RAM Stream</span>' +
      '</div>' +
      '<div class="chat-bubble">' +
       '<div style="margin-bottom:8px">' + renderMarkdown(revised) + '</div>' +
       '<div class="media-showcase-card">' +
        '<div class="media-card-header">' +
         '<span style="font-size:0.75rem;color:#94a3b8">Format: PNG • Memory Stream</span>' +
         '<span class="media-badge-zero">Public URL Ready</span>' +
        '</div>' +
        '<div class="media-card-viewport" onclick="openLightbox(\'' + imgUrl + '\', \'' + escapeHtml(text) + '\')">' +
         '<img src="' + imgUrl + '" alt="' + escapeHtml(text) + '" class="media-image-preview" />' +
         '<div class="media-zoom-overlay">🔍 Klik untuk Zoom Layar Penuh</div>' +
        '</div>' +
        '<div class="media-card-actions">' +
         '<button class="btn-media btn-media-zoom" onclick="openLightbox(\'' + imgUrl + '\', \'' + escapeHtml(text) + '\')">🔍 Layar Penuh</button>' +
         '<a href="' + imgUrl + '?download=1" download="image.png" class="btn-media btn-media-download">⬇️ Unduh PNG</a>' +
         '<button class="btn-media btn-media-copy" onclick="copySnippet(\'' + imgUrl + '\', this)">📋 Salin URL</button>' +
        '</div>' +
       '</div>' +
      '</div>' +
     '</div>';
    messagesContainer.appendChild(assistantRow);
    toast('Gambar selesai dalam ' + dur + ' detik!', 'success');
    loadRecentMedia();
   } else {
    const err = (data.error && data.error.message) || JSON.stringify(data);
    renderErrorRow(model, r.status, dur, err);
   }

  } else if (playgroundMode === 'video') {
   if (statusText) statusText.textContent = '🎬 Mengirim ke /v1/videos/generations (' + model + ')...';
   const aspect = document.getElementById('mediaAspect') ? document.getElementById('mediaAspect').value : '16:9';
   const duration = parseInt(document.getElementById('mediaDuration') ? document.getElementById('mediaDuration').value : '5') || 5;

   const r = await fetch(API + '/v1/videos/generations', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
     prompt: text,
     model: model,
     account_id: accId || undefined,
     aspect_ratio: aspect,
     duration: duration
    })
   });
   const dur = ((Date.now() - t0) / 1000).toFixed(2);
   const data = await r.json();

   const loadingEl = document.getElementById(loadingId);
   if (loadingEl) loadingEl.remove();

   if (r.ok && data.data && data.data.length > 0) {
    const vidItem = data.data[0];
    const vidUrl = vidItem.url;
    const revised = vidItem.revised_prompt || text;

    const assistantRow = document.createElement('div');
    assistantRow.className = 'chat-message-row assistant';
    assistantRow.innerHTML = '<div class="chat-avatar assistant">🎬</div>' +
     '<div class="chat-bubble-container">' +
      '<div class="chat-bubble-meta">' +
       '<span class="chat-meta-model">' + escapeHtml(model) + '</span>' +
       '<span class="chat-meta-time">' + timeStr + ' &bull; ' + dur + 's</span>' +
       '<span class="media-badge-zero" style="background:#1e3a8a;color:#93c5fd;border-color:#3b82f6">🎬 MP4 Stream (' + duration + 's)</span>' +
      '</div>' +
      '<div class="chat-bubble">' +
       '<div style="margin-bottom:8px">' + renderMarkdown(revised) + '</div>' +
       '<div class="media-showcase-card">' +
        '<div class="media-card-header">' +
         '<span style="font-size:0.75rem;color:#94a3b8">Video MP4 • Rasio: ' + escapeHtml(aspect) + ' • ' + duration + 's</span>' +
         '<span class="media-badge-zero">HTTP 206 Seekable</span>' +
        '</div>' +
        '<div style="background:#02040a;padding:8px;border-radius:6px;display:flex;justify-content:center">' +
         '<video class="media-video-player" controls autoplay loop playsinline src="' + vidUrl + '"></video>' +
        '</div>' +
        '<div class="media-card-actions">' +
         '<a href="' + vidUrl + '?download=1" download="video.mp4" class="btn-media btn-media-download">⬇️ Unduh Video (.mp4)</a>' +
         '<button class="btn-media btn-media-copy" onclick="copySnippet(\'' + vidUrl + '\', this)">📋 Salin URL Video</button>' +
        '</div>' +
       '</div>' +
      '</div>' +
     '</div>';
    messagesContainer.appendChild(assistantRow);
    toast('Video selesai dalam ' + dur + ' detik!', 'success');
    loadRecentMedia();
   } else {
    const err = (data.error && data.error.message) || JSON.stringify(data);
    renderErrorRow(model, r.status, dur, err);
   }

  } else {
   // Normal Chat Mode (with Vision Multimodal Support)
   let userMessageContent;
   if (currentAttachments.length > 0) {
     userMessageContent = currentAttachments.map(function(a) {
       return { type: 'image_url', image_url: { url: a.dataUrl } };
     });
     userMessageContent.push({ type: 'text', text: text || 'Please inspect and analyze this attached image in detail.' });
   } else {
     userMessageContent = text;
   }
   chatHistory.push({role: 'user', content: userMessageContent});
   updateChatTurnsBadge();

   if (statusText) statusText.textContent = '⏳ Mengirim ke /v1/chat/completions (' + model + ')...';
   const r = await fetch(API + '/v1/chat/completions', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
     model: model,
     messages: chatHistory,
     stream: false
    })
   });
   const dur = ((Date.now() - t0) / 1000).toFixed(2);
   const data = await r.json();

   const loadingEl = document.getElementById(loadingId);
   if (loadingEl) loadingEl.remove();

   if (r.ok && data.choices && data.choices.length > 0) {
    const reply = data.choices[0].message.content || '';
    chatHistory.push({role: 'assistant', content: reply});
    updateChatTurnsBadge();

    const assistantRow = document.createElement('div');
    assistantRow.className = 'chat-message-row assistant';
    assistantRow.innerHTML = '<div class="chat-avatar assistant">✨</div>' +
     '<div class="chat-bubble-container">' +
      '<div class="chat-bubble-meta">' +
       '<span class="chat-meta-model">' + escapeHtml(model) + '</span>' +
       '<span class="chat-meta-time">' + timeStr + ' &bull; ' + dur + 's</span>' +
      '</div>' +
      '<div class="chat-bubble">' + renderMarkdown(reply) + '</div>' +
      '<div class="chat-bubble-actions">' +
       '<button class="chat-action-btn" onclick="copySnippet(this.closest(\'.chat-bubble-container\').querySelector(\'.chat-bubble\').innerText, this)">📋 Salin Teks</button>' +
      '</div>' +
     '</div>';
    messagesContainer.appendChild(assistantRow);
    toast('Diterima dalam ' + dur + ' detik', 'success');
   } else {
    const err = (data.error && data.error.message) || JSON.stringify(data);
    renderErrorRow(model, r.status, dur, err);
   }
  }
 } catch (e) {
  const loadingEl = document.getElementById(loadingId);
  if (loadingEl) loadingEl.remove();

  const errRow = document.createElement('div');
  errRow.className = 'chat-message-row assistant';
  errRow.innerHTML = '<div class="chat-avatar assistant" style="background:#ef4444">⚠️</div>' +
   '<div class="chat-bubble-container">' +
    '<div class="chat-bubble-meta"><span class="chat-meta-model" style="color:#ef4444">Network Error</span></div>' +
    '<div class="chat-bubble" style="background:#450a0a;border-color:#7f1d1d;color:#fca5a5">' + escapeHtml(e.message) + '</div>' +
   '</div>';
  messagesContainer.appendChild(errRow);
  toast(e.message, 'error');
 } finally {
  btn.disabled = false;
  if (statusText) statusText.textContent = '';
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
  input.focus();
 }
}

function renderErrorRow(model, status, dur, err) {
 const messagesContainer = document.getElementById('chatMessages');
 const errRow = document.createElement('div');
 errRow.className = 'chat-message-row assistant';
 errRow.innerHTML = '<div class="chat-avatar assistant" style="background:#ef4444">⚠️</div>' +
  '<div class="chat-bubble-container">' +
   '<div class="chat-bubble-meta">' +
    '<span class="chat-meta-model" style="border-color:#ef4444;color:#ef4444">' + escapeHtml(model) + ' (Error ' + status + ')</span>' +
    '<span class="chat-meta-time">' + dur + 's</span>' +
   '</div>' +
   '<div class="chat-bubble" style="background:#450a0a;border-color:#7f1d1d;color:#fca5a5">' + escapeHtml(err) + '</div>' +
  '</div>';
 messagesContainer.appendChild(errRow);
 toast('Error ' + status + ': ' + err, 'error');
}

function switchGuideTab(tabName) {
 document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === tabName));
 document.querySelectorAll('.tab-pane').forEach(p => p.classList.toggle('active', p.id === 'tab-' + tabName));
}

function copySnippet(text, btn) {
 if (!text) return;
 navigator.clipboard.writeText(text).then(() => {
  if (btn) {
   const orig = btn.textContent;
   btn.textContent = '✓ Copied';
   setTimeout(() => btn.textContent = orig, 1600);
  }
  toast('Copied to clipboard!', 'success');
 }).catch(() => {
  toast('Failed to copy', 'error');
 });
}

function updateGuideSnippets(modelsList) {
 const origin = window.location.origin;
 const baseUrl = origin + '/v1';
 const chatUrl = origin + '/v1/chat/completions';
 const modelsUrl = origin + '/v1/models';
 const apiKey = (window.cachedConfig && window.cachedConfig.authToken) ? window.cachedConfig.authToken : 'sk-cookie-router';
 
 if (modelsList && modelsList.length > 0) {
  window.cachedModels = modelsList;
 }
 const models = window.cachedModels || ['gpt-5-6-instant', 'gpt-5-6', 'auto'];
 const defaultModel = models[0] || 'gpt-5-6-instant';

 const setEl = (id, text) => { const el = document.getElementById(id); if (el) el.textContent = text; };
 setEl('specBaseUrl', baseUrl);
 setEl('specApiKey', apiKey);
 setEl('specChatUrl', chatUrl);
 setEl('specModelsUrl', modelsUrl);

 setEl('cursorBaseUrl', baseUrl);
 setEl('cursorApiKey', apiKey);
 setEl('cursorModel', defaultModel);

 setEl('antigravityBash', 'export OPENAI_BASE_URL="' + baseUrl + '"\nexport OPENAI_API_KEY="' + apiKey + '"\nexport MODEL_NAME="' + defaultModel + '"');
 setEl('antigravityJson', JSON.stringify({
  "provider": "openai",
  "base_url": baseUrl,
  "api_key": apiKey,
  "model": defaultModel
 }, null, 2));

 setEl('clineBaseUrl', baseUrl);
 setEl('clineApiKey', apiKey);
 setEl('clineModel', defaultModel);

 setEl('continueJson', JSON.stringify({
  "models": [
   {
    "title": "Cookie Router (" + defaultModel + ")",
    "provider": "openai",
    "model": defaultModel,
    "apiBase": baseUrl,
    "apiKey": apiKey,
    "contextLength": 128000
   }
  ]
 }, null, 2));

 setEl('hermesYaml', 'model:\n  provider: custom\n  name: ' + defaultModel + '\n  base_url: ' + baseUrl + '\n  api_key: ' + apiKey);

 setEl('pythonCode', 'from openai import OpenAI\n\nclient = OpenAI(\n    base_url="' + baseUrl + '",\n    api_key="' + apiKey + '"\n)\n\nresponse = client.chat.completions.create(\n    model="' + defaultModel + '",\n    messages=[{"role": "user", "content": "Halo!"}],\n    stream=True\n)\nfor chunk in response:\n    print(chunk.choices[0].delta.content or "", end="")');

 setEl('curlCmd', 'curl -s -X POST ' + chatUrl + ' \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ' + apiKey + '" \\\n  -d \'{\n    "model": "' + defaultModel + '",\n    "messages": [{"role": "user", "content": "Halo!"}]\n  }\'');

 setEl('aiderCmd', 'aider --openai-api-base ' + baseUrl + ' --openai-api-key ' + apiKey + ' --model openai/' + defaultModel);
}

// Init
loadSettings();
refreshAccounts().then(populateChatModels);
updateGuideSnippets();
loadRecentMedia();
</script>
</body>
</html>`
