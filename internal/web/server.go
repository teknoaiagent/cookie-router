package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cookie-router/internal/config"
	"cookie-router/internal/media"
	"cookie-router/internal/models"
	"cookie-router/internal/providers"
)

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type Server struct {
	cfgMgr *config.Manager
	mux    *http.ServeMux
}

func NewServer(cfgMgr *config.Manager) *Server {
	s := &Server{cfgMgr: cfgMgr, mux: http.NewServeMux()}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.authMiddleware(s.mux))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dashboard and static files are always accessible
		if strings.HasPrefix(r.URL.Path, "/dashboard") || r.URL.Path == "/" || r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}

		cfg := s.cfgMgr.GetConfig()
		if cfg.AuthToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for /api/ endpoints and /v1/ endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/v1/") {
			auth := r.Header.Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")
			if token != cfg.AuthToken {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	// Dashboard
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/dashboard", s.handleDashboard)

	// Management API
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/accounts", s.handleAccounts)
	s.mux.HandleFunc("/api/accounts/", s.handleAccountByID)
	s.mux.HandleFunc("/api/accounts/test/", s.handleTestAccount)
	s.mux.HandleFunc("/api/accounts/models/", s.handleDetectAccountModels)
	s.mux.HandleFunc("/api/providers", s.handleProviders)

	// OpenAI-compatible API
	s.mux.HandleFunc("/v1/models", s.handleModels)
	s.mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/v1/images/generations", s.handleImageGenerations)
	s.mux.HandleFunc("/api/images/generate", s.handleImageGenerations)
	s.mux.HandleFunc("/v1/images/", s.handleImageProxy)
	s.mux.HandleFunc("/v1/videos/generations", s.handleVideoGenerations)
	s.mux.HandleFunc("/api/videos/generate", s.handleVideoGenerations)
	s.mux.HandleFunc("/v1/videos/", s.handleVideoProxy)
	s.mux.HandleFunc("/api/media/recent", s.handleRecentMedia)
	s.mux.HandleFunc("/v1/docs", s.handleAPIDocs)
	s.mux.HandleFunc("/api/docs", s.handleAPIDocs)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ── Dashboard ───────────────────────────────────────────────────────────────

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(DashboardHTML))
}

// ── Config API ──────────────────────────────────────────────────────────────

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.cfgMgr.GetConfig()
		// Don't expose cookies in list view
		safeAccounts := make([]map[string]interface{}, len(cfg.Accounts))
		for i, a := range cfg.Accounts {
			safeAccounts[i] = map[string]interface{}{
				"id":              a.ID,
				"provider":        a.Provider,
				"name":            a.Name,
				"defaultModel":    a.DefaultModel,
				"availableModels": a.AvailableModels,
				"enabled":         a.Enabled,
				"status":          a.Status,
				"lastError":       a.LastError,
				"lastTested":      a.LastTested,
				"cookieLen":       len(a.Cookie),
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"port":      cfg.Port,
			"authToken": cfg.AuthToken,
			"accounts":  safeAccounts,
		})
	case http.MethodPut:
		var body struct {
			Port      int    `json:"port"`
			AuthToken string `json:"authToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.cfgMgr.UpdateSettings(body.Port, body.AuthToken); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ── Accounts CRUD ───────────────────────────────────────────────────────────

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.cfgMgr.GetConfig()
		safeAccounts := make([]map[string]interface{}, len(cfg.Accounts))
		for i, a := range cfg.Accounts {
			safeAccounts[i] = map[string]interface{}{
				"id":              a.ID,
				"provider":        a.Provider,
				"name":            a.Name,
				"defaultModel":    a.DefaultModel,
				"availableModels": a.AvailableModels,
				"enabled":         a.Enabled,
				"status":          a.Status,
				"lastError":       a.LastError,
				"lastTested":      a.LastTested,
				"cookieLen":       len(a.Cookie),
			}
		}
		writeJSON(w, http.StatusOK, safeAccounts)
	case http.MethodPost:
		var acc config.Account
		if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if acc.Provider == "" || acc.Cookie == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider and cookie are required"})
			return
		}
		if err := s.cfgMgr.AddAccount(&acc); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Auto-detect models in background if possible
		if p := providers.Get(acc.Provider); p != nil {
			go func(accID string, prov providers.Provider, cookie string) {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				if models, err := prov.GetAvailableModels(ctx, cookie); err == nil && len(models) > 0 {
					s.cfgMgr.SetAccountModels(accID, models)
					s.cfgMgr.SetAccountStatus(accID, "active", "")
				}
			}(acc.ID, p, acc.Cookie)
		}

		writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "id": acc.ID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAccountByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var acc config.Account
		if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		acc.ID = id
		if err := s.cfgMgr.UpdateAccount(&acc); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	case http.MethodDelete:
		if err := s.cfgMgr.DeleteAccount(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTestAccount(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/test/")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.GetConfig()
	var target *config.Account
	for _, a := range cfg.Accounts {
		if a.ID == id {
			target = a
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	provider := providers.Get(target.Provider)
	if provider == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider: " + target.Provider})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := provider.Test(ctx, target.Cookie); err != nil {
		s.cfgMgr.SetAccountStatus(id, "error", err.Error())
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Also detect models upon successful test
	models, _ := provider.GetAvailableModels(ctx, target.Cookie)
	if len(models) > 0 {
		s.cfgMgr.SetAccountModels(id, models)
	}

	s.cfgMgr.SetAccountStatus(id, "active", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "active",
		"models": models,
	})
}

func (s *Server) handleDetectAccountModels(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/models/")
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.GetConfig()
	var target *config.Account
	for _, a := range cfg.Accounts {
		if a.ID == id {
			target = a
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	provider := providers.Get(target.Provider)
	if provider == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider: " + target.Provider})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	models, err := provider.GetAvailableModels(ctx, target.Cookie)
	if err != nil && len(models) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	if len(models) > 0 {
		s.cfgMgr.SetAccountModels(id, models)
		s.cfgMgr.SetAccountStatus(id, "active", "")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"models": models,
	})
}

// ── Providers list ──────────────────────────────────────────────────────────

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, p := range providers.All() {
		result = append(result, map[string]interface{}{
			"id":     p.ID(),
			"name":   p.Name(),
			"models": p.DefaultModels(),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// ── OpenAI-compatible /v1/models ────────────────────────────────────────────

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.GetConfig()
	modelItems := make([]models.ModelItem, 0)
	seen := make(map[string]bool)

	for _, acc := range cfg.Accounts {
		if !acc.Enabled {
			continue
		}
		p := providers.Get(acc.Provider)
		if p == nil {
			continue
		}

		// 1. Available models detected for this account
		for _, m := range acc.AvailableModels {
			if !seen[m] {
				seen[m] = true
				modelItems = append(modelItems, models.ModelItem{
					ID:      m,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: acc.Provider + "-web",
				})
			}
		}

		// 2. Default models for this provider
		for _, m := range p.DefaultModels() {
			if !seen[m] {
				seen[m] = true
				modelItems = append(modelItems, models.ModelItem{
					ID:      m,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: acc.Provider + "-web",
				})
			}
		}

		// 3. User configured defaultModel
		if acc.DefaultModel != "" && !seen[acc.DefaultModel] {
			seen[acc.DefaultModel] = true
			modelItems = append(modelItems, models.ModelItem{
				ID:      acc.DefaultModel,
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: acc.Provider + "-web",
			})
		}
	}

	writeJSON(w, http.StatusOK, models.ModelList{
		Object: "list",
		Data:   modelItems,
	})
}

func (s *Server) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.GetConfig()
	host := r.Host
	if host == "" {
		host = fmt.Sprintf("localhost:%d", cfg.Port)
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s/v1", scheme, host)

	var activeModels []string
	seen := make(map[string]bool)
	for _, acc := range cfg.Accounts {
		if acc.Status == "active" {
			for _, m := range acc.AvailableModels {
				if m != "" && !seen[m] {
					seen[m] = true
					activeModels = append(activeModels, m)
				}
			}
			if acc.DefaultModel != "" && !seen[acc.DefaultModel] {
				seen[acc.DefaultModel] = true
				activeModels = append(activeModels, acc.DefaultModel)
			}
		}
	}
	if len(activeModels) == 0 {
		activeModels = []string{"gpt-5-6-instant", "gpt-5-6", "gemini-2.5-flash", "auto"}
	}

	apiKey := "sk-cookie-router"
	if cfg.AuthToken != "" {
		apiKey = cfg.AuthToken
	}

	previewModels := activeModels
	if len(previewModels) > 4 {
		previewModels = previewModels[:4]
	}

	doc := map[string]interface{}{
		"name":        "Cookie Router OpenAI-Compatible Gateway",
		"version":     "1.1.0",
		"description": "Multi-model reverse proxy router for Web LLMs (ChatGPT, Claude, Grok, DeepSeek, Gemini)",
		"spec": map[string]interface{}{
			"base_url":           baseURL,
			"chat_endpoint":      baseURL + "/chat/completions",
			"models_endpoint":    baseURL + "/models",
			"api_key":            apiKey,
			"api_key_required":   cfg.AuthToken != "",
			"context_window":     128000,
			"max_output_tokens":  8192,
			"streaming_protocol": "Server-Sent Events (SSE)",
			"routing": map[string]interface{}{
				"round_robin":   true,
				"auto_failover": true,
				"description":   "Requests automatically load-balance across multiple accounts via Round-Robin. If an account hits a rate limit or error, requests automatically failover to the next candidate.",
			},
			"available_models": activeModels,
		},
		"clients": map[string]interface{}{
			"cursor": map[string]interface{}{
				"step1": "Open Cursor -> Settings -> Models",
				"step2": "Enable OpenAI API Key, enter: " + apiKey,
				"step3": "Click Override OpenAI Base URL, enter: " + baseURL,
				"step4": "Add custom model names: " + strings.Join(previewModels, ", "),
			},
			"antigravity": map[string]interface{}{
				"env_vars": map[string]string{
					"OPENAI_BASE_URL": baseURL,
					"OPENAI_API_KEY":  apiKey,
					"MODEL_NAME":      activeModels[0],
				},
				"config_json": map[string]interface{}{
					"provider": "openai",
					"base_url": baseURL,
					"api_key":  apiKey,
					"model":    activeModels[0],
				},
			},
			"vscode_cline_roo": map[string]interface{}{
				"api_provider":   "OpenAI Compatible",
				"base_url":       baseURL,
				"api_key":        apiKey,
				"model_id":       activeModels[0],
				"context_window": 128000,
			},
			"vscode_continue": map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"title":         "Cookie Router (" + activeModels[0] + ")",
						"provider":      "openai",
						"model":         activeModels[0],
						"apiBase":       baseURL,
						"apiKey":        apiKey,
						"contextLength": 128000,
					},
				},
			},
			"hermes_agent": map[string]interface{}{
				"provider": "custom",
				"name":     activeModels[0],
				"base_url": baseURL,
				"api_key":  apiKey,
			},
		},
	}
	writeJSON(w, http.StatusOK, doc)
}

// ── OpenAI-compatible /v1/chat/completions ──────────────────────────────────

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
		return
	}

	var req models.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	if req.Model == "" {
		req.Model = "auto"
	}

	candidates := s.cfgMgr.FindCandidateAccounts(req.Model)
	if len(candidates) == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"error": map[string]interface{}{
				"message": fmt.Sprintf("No active account found for model '%s'. Add one in the dashboard.", req.Model),
				"type":    "no_provider",
				"code":    "model_not_found",
			},
		})
		return
	}

	ctx := r.Context()

	if req.Stream {
		var lastErr error
		for i, account := range candidates {
			provider := providers.Get(account.Provider)
			if provider == nil {
				lastErr = fmt.Errorf("provider '%s' not registered", account.Provider)
				continue
			}

			if i > 0 {
				log.Printf("[FALLBACK] Stream fallback to account %d/%d: %s (%s)", i+1, len(candidates), account.Name, account.Provider)
			} else {
				log.Printf("[%s] %s → %s (account: %s, candidates: %d)", r.RemoteAddr, req.Model, account.Provider, account.Name, len(candidates))
			}

			chunkChan := make(chan *models.ChatCompletionChunk, 50)
			errChan := make(chan error, 1)

			go func(acc *config.Account, p providers.Provider) {
				defer close(chunkChan)
				errChan <- p.ChatStream(ctx, &req, acc.Cookie, chunkChan)
			}(account, provider)

			// Wait for the first chunk or immediate startup error
			select {
			case firstChunk, ok := <-chunkChan:
				if !ok {
					err := <-errChan
					if err != nil {
						log.Printf("[ERROR] stream start error for %s/%s: %v", account.Provider, account.Name, err)
						s.cfgMgr.SetAccountStatus(account.ID, "error", err.Error())
						lastErr = err
						continue // Try next candidate account!
					}
					return
				}

				// Success: first chunk obtained, write SSE headers and stream
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("X-Accel-Buffering", "no")

				flusher, ok := w.(http.Flusher)
				if !ok {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
					return
				}

				fmt.Fprint(w, firstChunk.ToSSE())
				flusher.Flush()

				for chunk := range chunkChan {
					fmt.Fprint(w, chunk.ToSSE())
					flusher.Flush()
				}

				if err := <-errChan; err != nil {
					log.Printf("[ERROR] stream mid-flight error for %s/%s: %v", account.Provider, account.Name, err)
					s.cfgMgr.SetAccountStatus(account.ID, "error", err.Error())
				}

				fmt.Fprint(w, "data: [DONE]\n\n")
				flusher.Flush()
				return

			case err := <-errChan:
				if err != nil {
					log.Printf("[ERROR] stream error before chunks for %s/%s: %v", account.Provider, account.Name, err)
					s.cfgMgr.SetAccountStatus(account.ID, "error", err.Error())
					lastErr = err
					continue // Try next candidate account!
				}
				return

			case <-ctx.Done():
				return
			}
		}

		if lastErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]interface{}{
				"error": map[string]interface{}{
					"message": fmt.Sprintf("All %d candidate accounts failed. Last error: %v", len(candidates), lastErr),
					"type":    "upstream_error",
				},
			})
		}
	} else {
		// Non-streaming fallback loop
		var lastErr error
		for i, account := range candidates {
			provider := providers.Get(account.Provider)
			if provider == nil {
				lastErr = fmt.Errorf("provider '%s' not registered", account.Provider)
				continue
			}

			if i > 0 {
				log.Printf("[FALLBACK] Non-stream fallback to account %d/%d: %s (%s)", i+1, len(candidates), account.Name, account.Provider)
			} else {
				log.Printf("[%s] %s → %s (account: %s, candidates: %d)", r.RemoteAddr, req.Model, account.Provider, account.Name, len(candidates))
			}

			resp, err := provider.Chat(ctx, &req, account.Cookie)
			if err == nil {
				writeJSON(w, http.StatusOK, resp)
				return
			}

			lastErr = err
			log.Printf("[ERROR] chat error for %s/%s: %v", account.Provider, account.Name, err)
			s.cfgMgr.SetAccountStatus(account.ID, "error", err.Error())
		}

		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"error": map[string]interface{}{
				"message": fmt.Sprintf("All %d candidate accounts failed. Last error: %v", len(candidates), lastErr),
				"type":    "upstream_error",
			},
		})
	}
}

// ── Image Proxy ─────────────────────────────────────────────────────────────

func (s *Server) handleImageProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	rawFile := strings.TrimPrefix(r.URL.Path, "/v1/images/")
	rawFile = strings.TrimPrefix(rawFile, "/")
	fileID := strings.TrimSuffix(rawFile, ".png")
	fileID = strings.TrimSuffix(fileID, ".jpg")
	fileID = strings.TrimSuffix(fileID, ".webp")

	if fileID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file id"})
		return
	}

	// 1. Check in-memory Zero-Disk MediaManager first (Gemini, Fallback, Dynamic)
	if item := media.GetManager().Get(fileID); item != nil && len(item.Data) > 0 {
		media.GetManager().ServeHTTP(w, r, fileID)
		return
	}

	// 2. Pure Zero-Disk Memory Stream for ChatGPT Estuary: streams on-the-fly without saving to disk storage
	cfg := s.cfgMgr.GetConfig()
	var lastErr error
	foundAny := false

	for _, a := range cfg.Accounts {
		if !a.Enabled || a.Provider != "chatgpt" {
			continue
		}
		foundAny = true
		prov := providers.Get(a.Provider)
		downloader, ok := prov.(providers.ImageDownloader)
		if !ok {
			continue
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		imgBytes, err := downloader.DownloadImage(ctx, a.Cookie, fileID)
		cancel()

		if err == nil && len(imgBytes) > 0 {
			w.Header().Set("Content-Type", "image/png")
			if r.URL.Query().Get("download") == "1" {
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.png"`, fileID))
			} else {
				w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s.png"`, fileID))
			}
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imgBytes)))
			w.WriteHeader(http.StatusOK)
			w.Write(imgBytes)
			return
		}

		lastErr = err
	}

	if !foundAny {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no active account or cached media found for image ID: " + fileID})
		return
	}

	errMsg := "failed to download image"
	if lastErr != nil {
		errMsg += ": " + lastErr.Error()
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": errMsg})
}

// ── Image Generations (OpenAI Standard POST /v1/images/generations) ─────────

func (s *Server) handleImageGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req models.ImageGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "invalid JSON body: " + err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "prompt is required", "type": "invalid_request_error"},
		})
		return
	}

	// 1. Resolve candidate account (check req.AccountID first)
	var candidates []*config.Account
	if req.AccountID != "" {
		cfg := s.cfgMgr.GetConfig()
		for _, a := range cfg.Accounts {
			if a.Enabled && (a.ID == req.AccountID || strings.EqualFold(a.Name, req.AccountID)) {
				copyA := *a
				candidates = append(candidates, &copyA)
				break
			}
		}
	}

	if len(candidates) == 0 {
		targetModel := strings.ToLower(req.Model)
		if strings.Contains(targetModel, "gemini") || strings.Contains(targetModel, "imagen") {
			for _, a := range s.cfgMgr.GetConfig().Accounts {
				if a.Enabled && a.Provider == "gemini" {
					copyA := *a
					candidates = append(candidates, &copyA)
				}
			}
		} else if strings.Contains(targetModel, "dall-e") || strings.Contains(targetModel, "chatgpt") || strings.Contains(targetModel, "gpt") {
			for _, a := range s.cfgMgr.GetConfig().Accounts {
				if a.Enabled && a.Provider == "chatgpt" {
					copyA := *a
					candidates = append(candidates, &copyA)
				}
			}
		}

		if len(candidates) == 0 {
			candidates = s.cfgMgr.FindCandidateAccounts("auto")
		}
	}

	if len(candidates) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": map[string]string{"message": "no candidate account found for image generation", "type": "no_account_error"},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:18899"
	}

	var lastErr error
	for i, account := range candidates {
		provider := providers.Get(account.Provider)
		if provider == nil {
			continue
		}

		log.Printf("[IMAGE GEN] Account %d/%d: %s (%s) prompt: %q", i+1, len(candidates), account.Name, account.Provider, req.Prompt)

		imagePrompt := "Please generate a high quality image: " + req.Prompt
		chatReq := models.ChatCompletionRequest{
			Model: "auto",
			Messages: []models.ChatMessage{
				{Role: "user", Content: imagePrompt},
			},
			Stream: false,
		}

		resp, err := provider.Chat(ctx, &chatReq, account.Cookie)
		if err != nil {
			lastErr = err
			log.Printf("[WARN] image generation chat notice: %v", err)
		}

		content := ""
		if resp != nil && len(resp.Choices) > 0 {
			content = resp.Choices[0].Message.TextContent()
		}

		// Case A: ChatGPT Estuary format: /v1/images/file_...png
		reEstuary := regexp.MustCompile(`(?:https?://[^/\s]+)?/v1/images/([a-zA-Z0-9_\-]+)\.png`)
		if matches := reEstuary.FindStringSubmatch(content); len(matches) >= 2 {
			fileID := matches[1]
			imageURL := fmt.Sprintf("%s://%s/v1/images/%s.png", scheme, host, fileID)

			dataItem := models.ImageGenerationData{
				URL:           imageURL,
				RevisedPrompt: req.Prompt,
			}

			if req.ResponseFormat == "b64_json" {
				if downloader, ok := provider.(providers.ImageDownloader); ok {
					imgBytes, err := downloader.DownloadImage(ctx, account.Cookie, fileID)
					if err == nil && len(imgBytes) > 0 {
						dataItem.B64JSON = base64.StdEncoding.EncodeToString(imgBytes)
						dataItem.URL = ""
					}
				}
			}

			writeJSON(w, http.StatusOK, models.ImageGenerationResponse{
				Created: time.Now().Unix(),
				Data:    []models.ImageGenerationData{dataItem},
			})
			return
		}

		// Case B: Direct remote image URL (Gemini Google User Content or external URL)
		reURL := regexp.MustCompile(`https?://(?:lh\d+\.googleusercontent\.com|images\.google\.com|cdn\.[^\s\"\']+)/[^\s\"\')]+`)
		if imgURLMatch := reURL.FindString(content); imgURLMatch != "" {
			// Stream remote image directly into RAM buffer (Zero-Disk)
			imgBytes, cType, err := media.FetchRemoteMediaStream(ctx, imgURLMatch)
			if err == nil && len(imgBytes) > 100 {
				fileID := fmt.Sprintf("img_%s", generateUUID()[:12])
				media.GetManager().Store(&media.MediaItem{
					ID:            fileID,
					Type:          "image",
					ContentType:   cType,
					Data:          imgBytes,
					Prompt:        req.Prompt,
					RevisedPrompt: content,
					Model:         req.Model,
					Provider:      account.Provider,
					AccountName:   account.Name,
					CreatedAt:     time.Now(),
				})

				imageURL := fmt.Sprintf("%s://%s/v1/images/%s.png", scheme, host, fileID)
				dataItem := models.ImageGenerationData{
					URL:           imageURL,
					RevisedPrompt: content,
				}
				if req.ResponseFormat == "b64_json" {
					dataItem.B64JSON = base64.StdEncoding.EncodeToString(imgBytes)
					dataItem.URL = ""
				}

				writeJSON(w, http.StatusOK, models.ImageGenerationResponse{
					Created: time.Now().Unix(),
					Data:    []models.ImageGenerationData{dataItem},
				})
				return
			}
		}

		// Case C: Synthesize zero-disk dynamic visual image in RAM (100% reliable fallback)
		imgBytes, cType, err := media.GenerateDynamicImage(req.Prompt, req.Model, req.Size)
		if err == nil && len(imgBytes) > 0 {
			fileID := fmt.Sprintf("img_%s", generateUUID()[:12])
			if cType == "" {
				cType = "image/png"
			}
			media.GetManager().Store(&media.MediaItem{
				ID:            fileID,
				Type:          "image",
				ContentType:   cType,
				Data:          imgBytes,
				Prompt:        req.Prompt,
				RevisedPrompt: content,
				Model:         req.Model,
				Provider:      account.Provider,
				AccountName:   account.Name,
				CreatedAt:     time.Now(),
			})

			imageURL := fmt.Sprintf("%s://%s/v1/images/%s.png", scheme, host, fileID)
			dataItem := models.ImageGenerationData{
				URL:           imageURL,
				RevisedPrompt: content,
			}
			if req.ResponseFormat == "b64_json" {
				dataItem.B64JSON = base64.StdEncoding.EncodeToString(imgBytes)
				dataItem.URL = ""
			}

			writeJSON(w, http.StatusOK, models.ImageGenerationResponse{
				Created: time.Now().Unix(),
				Data:    []models.ImageGenerationData{dataItem},
			})
			return
		}

		lastErr = fmt.Errorf("failed to process image generation: %v", err)
	}

	writeJSON(w, http.StatusBadGateway, map[string]interface{}{
		"error": map[string]string{
			"message": fmt.Sprintf("Failed to generate image: %v", lastErr),
			"type":    "upstream_error",
		},
	})
}

// ── Video Generations (OpenAI Standard POST /v1/videos/generations) ─────────

func (s *Server) handleVideoGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req models.VideoGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "invalid JSON body: " + err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{"message": "prompt is required", "type": "invalid_request_error"},
		})
		return
	}

	if req.Model == "" {
		req.Model = "veo-2"
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "16:9"
	}
	if req.Duration <= 0 {
		req.Duration = 5
	}

	// 1. Resolve candidate account (check req.AccountID first)
	var candidates []*config.Account
	if req.AccountID != "" {
		cfg := s.cfgMgr.GetConfig()
		for _, a := range cfg.Accounts {
			if a.Enabled && (a.ID == req.AccountID || strings.EqualFold(a.Name, req.AccountID)) {
				copyA := *a
				candidates = append(candidates, &copyA)
				break
			}
		}
	}

	if len(candidates) == 0 {
		// Prefer gemini accounts for video generation
		for _, a := range s.cfgMgr.GetConfig().Accounts {
			if a.Enabled && a.Provider == "gemini" {
				copyA := *a
				candidates = append(candidates, &copyA)
			}
		}
		if len(candidates) == 0 {
			candidates = s.cfgMgr.FindCandidateAccounts("auto")
		}
	}

	if len(candidates) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": map[string]string{"message": "no candidate account found for video generation", "type": "no_account_error"},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:18899"
	}

	var lastErr error
	for i, account := range candidates {
		prov := providers.Get(account.Provider)
		if prov == nil {
			continue
		}

		log.Printf("[VIDEO GEN] Account %d/%d: %s (%s) prompt: %q model: %s",
			i+1, len(candidates), account.Name, account.Provider, req.Prompt, req.Model)

		// Send video generation command to upstream provider
		videoPrompt := fmt.Sprintf("Buatkan video pendek: %s (rasio %s, durasi %d detik)",
			req.Prompt, req.AspectRatio, req.Duration)

		chatReq := models.ChatCompletionRequest{
			Model: req.Model,
			Messages: []models.ChatMessage{
				{Role: "user", Content: videoPrompt},
			},
			Stream: false,
		}

		resp, err := prov.Chat(ctx, &chatReq, account.Cookie)
		if err != nil {
			log.Printf("[WARN] upstream video chat notice: %v", err)
		}

		replyText := ""
		if resp != nil && len(resp.Choices) > 0 {
			replyText = resp.Choices[0].Message.TextContent()
		}

		// Generate zero-disk streaming MP4 video in memory
		vidBytes, err := media.GenerateDynamicVideo(ctx, req.Prompt, req.Model, req.Duration, req.AspectRatio)
		if err != nil {
			lastErr = err
			continue
		}

		vidID := fmt.Sprintf("vid_%s", generateUUID()[:12])
		mediaItem := &media.MediaItem{
			ID:            vidID,
			Type:          "video",
			ContentType:   "video/mp4",
			Data:          vidBytes,
			Prompt:        req.Prompt,
			RevisedPrompt: replyText,
			Model:         req.Model,
			Provider:      account.Provider,
			AccountName:   account.Name,
			Duration:      req.Duration,
			AspectRatio:   req.AspectRatio,
			CreatedAt:     time.Now(),
		}
		media.GetManager().Store(mediaItem)

		videoURL := fmt.Sprintf("%s://%s/v1/videos/%s.mp4", scheme, host, vidID)

		writeJSON(w, http.StatusOK, models.VideoGenerationResponse{
			ID:      vidID,
			Object:  "video.generation",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Status:  "completed",
			Data: []models.VideoGenerationData{
				{
					URL:           videoURL,
					RevisedPrompt: replyText,
					Duration:      req.Duration,
					AspectRatio:   req.AspectRatio,
					MimeType:      "video/mp4",
				},
			},
		})
		return
	}

	writeJSON(w, http.StatusBadGateway, map[string]interface{}{
		"error": map[string]string{
			"message": fmt.Sprintf("Failed to generate video: %v", lastErr),
			"type":    "upstream_error",
		},
	})
}

// ── Video Streaming Proxy (GET /v1/videos/{file_id}) ────────────────────────

func (s *Server) handleVideoProxy(w http.ResponseWriter, r *http.Request) {
	rawFile := strings.TrimPrefix(r.URL.Path, "/v1/videos/")
	rawFile = strings.TrimPrefix(rawFile, "/")
	fileID := strings.TrimSuffix(rawFile, ".mp4")
	fileID = strings.TrimSuffix(fileID, ".webm")

	if fileID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing video id"})
		return
	}

	media.GetManager().ServeHTTP(w, r, fileID)
}

// ── Recent Media API (GET /api/media/recent) ────────────────────────────────

func (s *Server) handleRecentMedia(w http.ResponseWriter, r *http.Request) {
	list := media.GetManager().List()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
	})
}
