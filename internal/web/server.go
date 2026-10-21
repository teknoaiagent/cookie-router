package web

import (
	"bytes"
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
	"sort"
	"strings"
	"time"

	"cookie-router/internal/config"
	"cookie-router/internal/media"
	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
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
	go s.backfillAccountProfiles()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.authMiddleware(s.mux))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, OpenAI-Beta, Accept, *")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dashboard, static files, and media previews are always accessible
		if strings.HasPrefix(r.URL.Path, "/dashboard") || r.URL.Path == "/" || r.URL.Path == "/favicon.ico" ||
			strings.HasPrefix(r.URL.Path, "/v1/images/") || strings.HasPrefix(r.URL.Path, "/v1/videos/") ||
			strings.HasPrefix(r.URL.Path, "/api/media/") {
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
	s.mux.HandleFunc("/api/accounts/probe", s.handleProbeAccount)
	s.mux.HandleFunc("/api/accounts", s.handleAccounts)
	s.mux.HandleFunc("/api/accounts/", s.handleAccountByID)
	s.mux.HandleFunc("/api/accounts/test/", s.handleTestAccount)
	s.mux.HandleFunc("/api/accounts/models/", s.handleDetectAccountModels)
	s.mux.HandleFunc("/api/providers", s.handleProviders)

	// OpenAI-compatible API
	s.mux.HandleFunc("/v1/models", s.handleModels)
	s.mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/v1/images/generations", s.handleImageGenerations)
	s.mux.HandleFunc("/v1/images/edits", s.handleImageEdits)
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
				"email":           a.Email,
				"plan":            a.Plan,
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
				"email":           a.Email,
				"plan":            a.Plan,
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

		// Auto-detect profile / email if not supplied
		if acc.Email == "" || acc.Plan == "" {
			if p := providers.Get(acc.Provider); p != nil {
				if detector, ok := p.(providers.AccountInfoDetector); ok {
					ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
					if info, err := detector.DetectAccountInfo(ctx, acc.Cookie); err == nil && info != nil {
						if acc.Email == "" {
							acc.Email = info.Email
						}
						if acc.Plan == "" {
							acc.Plan = info.Plan
						}
					}
					cancel()
				}
			}
			if acc.Email == "" {
				acc.Email = utils.ExtractEmailFromRaw(acc.Cookie)
			}
		}

		if acc.Name == "" || acc.Name == acc.Provider {
			if acc.Email != "" {
				acc.Name = acc.Email
			}
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

	// Also detect profile info upon successful test
	if detector, ok := provider.(providers.AccountInfoDetector); ok {
		if info, err := detector.DetectAccountInfo(ctx, target.Cookie); err == nil && info != nil {
			s.cfgMgr.SetAccountProfile(id, info.Email, info.Plan)
			target.Email = info.Email
			target.Plan = info.Plan
		}
	} else if target.Email == "" {
		if em := utils.ExtractEmailFromRaw(target.Cookie); em != "" {
			s.cfgMgr.SetAccountProfile(id, em, "")
			target.Email = em
		}
	}

	s.cfgMgr.SetAccountStatus(id, "active", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "active",
		"models": models,
		"email":  target.Email,
		"plan":   target.Plan,
	})
}

func (s *Server) handleProbeAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Cookie   string `json:"cookie"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if body.Cookie == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cookie is required"})
		return
	}

	normCookie := utils.NormalizeCookie(body.Cookie, body.Provider)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	resp := map[string]interface{}{
		"status":   "ok",
		"provider": body.Provider,
	}

	if p := providers.Get(body.Provider); p != nil {
		if detector, ok := p.(providers.AccountInfoDetector); ok {
			info, err := detector.DetectAccountInfo(ctx, normCookie)
			if err == nil && info != nil {
				resp["email"] = info.Email
				resp["name"] = info.Name
				resp["plan"] = info.Plan
			} else if err != nil {
				resp["error"] = err.Error()
			}
		}
		defModels := p.DefaultModels()
		if len(defModels) > 0 {
			resp["defaultModel"] = defModels[0]
		}
	}

	if resp["email"] == nil || resp["email"] == "" {
		if em := utils.ExtractEmailFromRaw(normCookie); em != "" {
			resp["email"] = em
		} else if em := utils.ExtractEmailFromRaw(body.Cookie); em != "" {
			resp["email"] = em
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) backfillAccountProfiles() {
	time.Sleep(1 * time.Second)
	cfg := s.cfgMgr.GetConfig()
	for _, a := range cfg.Accounts {
		if a.Email == "" && a.Cookie != "" {
			if p := providers.Get(a.Provider); p != nil {
				if detector, ok := p.(providers.AccountInfoDetector); ok {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					if info, err := detector.DetectAccountInfo(ctx, a.Cookie); err == nil && info != nil && info.Email != "" {
						s.cfgMgr.SetAccountProfile(a.ID, info.Email, info.Plan)
					}
					cancel()
				}
			}
			if a.Email == "" {
				if em := utils.ExtractEmailFromRaw(a.Cookie); em != "" {
					s.cfgMgr.SetAccountProfile(a.ID, em, "")
				}
			}
		}
	}
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
			// Cache in Zero-Disk RAM buffer for instant subsequent views & gallery
			media.GetManager().Store(&media.MediaItem{
				ID:          fileID,
				Type:        "image",
				ContentType: "image/png",
				Data:        imgBytes,
				Prompt:      "ChatGPT Generated Image",
				Model:       "gpt-6-thinking",
				Provider:    "chatgpt",
				AccountName: a.Name,
				CreatedAt:   time.Now(),
			})

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
		if strings.Contains(targetModel, "gemini") || strings.Contains(targetModel, "imagen") || strings.Contains(targetModel, "banana") || strings.Contains(targetModel, "argon") {
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
			for _, a := range s.cfgMgr.GetConfig().Accounts {
				if a.Enabled {
					copyA := *a
					candidates = append(candidates, &copyA)
				}
			}
		}
	}

	if len(candidates) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": map[string]string{"message": "no candidate account found for image generation", "type": "no_account_error"},
		})
		return
	}

	// Sort candidates: Plus / Pro accounts FIRST
	sort.SliceStable(candidates, func(i, j int) bool {
		isPlusI := strings.EqualFold(candidates[i].Plan, "plus") || strings.EqualFold(candidates[i].Plan, "pro") || strings.EqualFold(candidates[i].Plan, "team")
		isPlusJ := strings.EqualFold(candidates[j].Plan, "plus") || strings.EqualFold(candidates[j].Plan, "pro") || strings.EqualFold(candidates[j].Plan, "team")
		if isPlusI && !isPlusJ {
			return true
		}
		return false
	})

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

	styleDirective := ""
	if req.Style != "" && req.Style != "none" {
		switch strings.ToLower(req.Style) {
		case "flare":
			styleDirective = " dengan gaya visual Flare, pencahayaan sinematik hangat lembut berkarakter."
		case "sunburst":
			styleDirective = " dengan efek Sunburst sinar matahari dramatis dan radiant warmth."
		case "vibrant":
			styleDirective = " dengan warna cerah kontras tinggi (vibrant cinematic)."
		case "natural":
			styleDirective = " dengan gaya fotorealistik natural detail tajam autentik."
		default:
			styleDirective = fmt.Sprintf(" gaya %s.", req.Style)
		}
	}

	for i, account := range candidates {
		provider := providers.Get(account.Provider)
		if provider == nil {
			continue
		}

		log.Printf("[IMAGE GEN] Account %d/%d: %s (%s) prompt: %q", i+1, len(candidates), account.Name, account.Provider, req.Prompt)

		var imgs []string
		seenImg := make(map[string]bool)
		addImg := func(u string) {
			u = strings.TrimSpace(u)
			if u != "" && !seenImg[u] {
				seenImg[u] = true
				imgs = append(imgs, u)
			}
		}
		if req.Image != "" {
			addImg(req.Image)
		}
		for _, att := range req.Attachments {
			addImg(att)
		}

		cleanPrompt := strings.TrimSpace(req.Prompt)
		lowerPrompt := strings.ToLower(cleanPrompt)

		// Ensure prompt explicitly triggers image creation tool in conversational models (ChatGPT Plus)
		hasExplicitCommand := strings.HasPrefix(lowerPrompt, "buat gambar") ||
			strings.HasPrefix(lowerPrompt, "buatkan gambar") ||
			strings.HasPrefix(lowerPrompt, "buat ") ||
			strings.HasPrefix(lowerPrompt, "buatkan ") ||
			strings.HasPrefix(lowerPrompt, "generate image") ||
			strings.HasPrefix(lowerPrompt, "create image") ||
			strings.HasPrefix(lowerPrompt, "draw ") ||
			strings.HasPrefix(lowerPrompt, "lukis ") ||
			strings.HasPrefix(lowerPrompt, "ubah ") ||
			strings.HasPrefix(lowerPrompt, "jadikan ") ||
			strings.HasPrefix(lowerPrompt, "ganti ")

		finalPrompt := cleanPrompt
		if len(imgs) == 0 && !hasExplicitCommand {
			finalPrompt = "Buatkan gambar: " + cleanPrompt
		}

		// Strip any aspect ratio suffix from user prompt to keep the prompt clean
		// ChatGPT image_gen.text2im reacts to exact wording
		aspectSuffixes := []string{
			"(rasio aspek landscape 16:9)",
			"(rasio aspek portrait tegak 9:16)",
			"(rasio aspek landscape)",
			"(rasio aspek portrait)",
		}
		for _, suf := range aspectSuffixes {
			finalPrompt = strings.TrimSpace(strings.ReplaceAll(finalPrompt, suf, ""))
		}

		// Pass aspect ratio directive explicitly so ChatGPT DALL-E 3 configures orientation
		aspectDirective := ""
		if req.Size == "1024x1792" {
			aspectDirective = " dengan orientasi portrait vertikal (rasio aspek 9:16)."
		} else if req.Size == "1792x1024" {
			aspectDirective = " dengan orientasi landscape mendatar (rasio aspek 16:9)."
		} else if req.Size == "1024x1024" {
			aspectDirective = " dengan orientasi persegi (rasio aspek 1:1)."
		}

		imagePrompt := finalPrompt + styleDirective + aspectDirective

		var userContent interface{} = imagePrompt
		if len(imgs) > 0 {
			var parts []interface{}
			for _, img := range imgs {
				parts = append(parts, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": img,
					},
				})
			}
			parts = append(parts, map[string]interface{}{
				"type": "text",
				"text": imagePrompt,
			})
			userContent = parts
		}

		// Pick the best model for image generation
		modelToUse := req.Model
		if account.Provider == "chatgpt" {
			hasModel := func(m string) bool {
				for _, am := range account.AvailableModels {
					if am == m {
						return true
					}
				}
				return false
			}
			if modelToUse == "" || modelToUse == "auto" || modelToUse == "dall-e-3" || modelToUse == "dall-e-2" || modelToUse == "gpt-image-2" {
				if hasModel("gpt-6-thinking") {
					modelToUse = "gpt-6-thinking"
				} else if hasModel("gpt-5-6-thinking") {
					modelToUse = "gpt-5-6-thinking"
				} else if hasModel("gpt-5-6") {
					modelToUse = "gpt-5-6"
				} else {
					modelToUse = "auto"
				}
			}
		} else if account.Provider == "gemini" {
			modelToUse = "imagen-3"
		}
		if modelToUse == "" {
			modelToUse = "auto"
		}

		chatReq := models.ChatCompletionRequest{
			Model: modelToUse,
			Messages: []models.ChatMessage{
				{Role: "user", Content: userContent},
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
		log.Printf("[IMAGE GEN DEBUG] Candidate %s returned content: %q, err: %v", account.Name, content, err)

		// Case A: ChatGPT Estuary format: /v1/images/file_...png
		reEstuary := regexp.MustCompile(`(?:https?://[^/\s]+)?/v1/images/([a-zA-Z0-9_\-]+)\.png`)
		if matches := reEstuary.FindStringSubmatch(content); len(matches) >= 2 {
			fileID := matches[1]
			imageURL := fmt.Sprintf("%s://%s/v1/images/%s.png", scheme, host, fileID)

			dataItem := models.ImageGenerationData{
				URL:           imageURL,
				RevisedPrompt: req.Prompt,
			}

			// Pre-cache into in-memory Zero-Disk MediaManager for fast display and Recent Media Gallery
			if downloader, ok := provider.(providers.ImageDownloader); ok {
				if req.ResponseFormat == "b64_json" {
					imgBytes, err := downloader.DownloadImage(ctx, account.Cookie, fileID)
					if err == nil && len(imgBytes) > 0 {
						dataItem.B64JSON = base64.StdEncoding.EncodeToString(imgBytes)
						dataItem.URL = ""
						media.GetManager().Store(&media.MediaItem{
							ID:            fileID,
							Type:          "image",
							ContentType:   "image/png",
							Data:          imgBytes,
							Prompt:        req.Prompt,
							RevisedPrompt: req.Prompt,
							Model:         modelToUse,
							Provider:      account.Provider,
							AccountName:   account.Name,
							CreatedAt:     time.Now(),
						})
					}
				} else {
					// Async prefetch into RAM buffer so browser gets 0ms fast response
					go func(fID string, cookie string, pPrompt string, mModel string, aName string) {
						bgCtx, bgCancel := context.WithTimeout(context.Background(), 60*time.Second)
						defer bgCancel()
						imgBytes, err := downloader.DownloadImage(bgCtx, cookie, fID)
						if err == nil && len(imgBytes) > 0 {
							media.GetManager().Store(&media.MediaItem{
								ID:            fID,
								Type:          "image",
								ContentType:   "image/png",
								Data:          imgBytes,
								Prompt:        pPrompt,
								RevisedPrompt: pPrompt,
								Model:         mModel,
								Provider:      "chatgpt",
								AccountName:   aName,
								CreatedAt:     time.Now(),
							})
							log.Printf("[IMAGE GEN] Prefetched and cached %s (%d bytes) in RAM buffer", fID, len(imgBytes))
						}
					}(fileID, account.Cookie, req.Prompt, modelToUse, account.Name)
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
			fullResMatch := imgURLMatch
			if strings.Contains(fullResMatch, "googleusercontent.com") && !strings.Contains(fullResMatch, "=s") && !strings.Contains(fullResMatch, "=w") {
				fullResMatch += "=s0"
			}
			hdrs := make(map[string]string)
			if account.Provider == "gemini" || strings.Contains(imgURLMatch, "googleusercontent.com") {
				hdrs["Cookie"] = account.Cookie
				hdrs["Referer"] = "https://gemini.google.com/"
			}
			// Stream remote image directly into RAM buffer (Zero-Disk)
			imgBytes, cType, err := media.FetchRemoteMediaStreamWithHeaders(ctx, fullResMatch, hdrs)
			log.Printf("[IMAGE GEN DEBUG] Downloaded %s: %d bytes, cType=%s, err=%v", fullResMatch[:35], len(imgBytes), cType, err)
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

		log.Printf("[IMAGE GEN] Candidate %s did not return image estuary/url, trying next candidate...", account.Name)
		lastErr = fmt.Errorf("candidate %s did not produce image output", account.Name)
	}

	if len(candidates) > 0 && lastErr != nil && req.Model != "fallback" {
		log.Printf("[IMAGE GEN] Upstream failed: %v", lastErr)
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"error": map[string]string{
				"message": fmt.Sprintf("Gagal generate gambar dari provider: %v", lastErr),
				"type":    "upstream_error",
			},
		})
		return
	}

	// Case C: If all upstream accounts failed, fallback to zero-disk in RAM
	log.Printf("[IMAGE GEN] All candidates exhausted, invoking dynamic visual fallback...")
	imgBytes, cType, err := media.GenerateDynamicImage(req.Prompt+styleDirective, req.Model, req.Size)
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
			RevisedPrompt: req.Prompt,
			Model:         req.Model,
			Provider:      "fallback",
			AccountName:   "in-memory-flux",
			CreatedAt:     time.Now(),
		})

		imageURL := fmt.Sprintf("%s://%s/v1/images/%s.png", scheme, host, fileID)
		dataItem := models.ImageGenerationData{
			URL:           imageURL,
			RevisedPrompt: req.Prompt,
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

	writeJSON(w, http.StatusBadGateway, map[string]interface{}{
		"error": map[string]string{
			"message": fmt.Sprintf("Failed to generate image: %v", lastErr),
			"type":    "upstream_error",
		},
	})
}

// ── Image Edits (OpenAI Standard POST /v1/images/edits) ─────────────────────

func (s *Server) handleImageEdits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form: " + err.Error()})
			return
		}

		file, header, err := r.FormFile("image")
		var b64Data string
		if err == nil && file != nil {
			defer file.Close()
			bytes, err := io.ReadAll(file)
			if err == nil && len(bytes) > 0 {
				mime := header.Header.Get("Content-Type")
				if mime == "" {
					mime = "image/png"
				}
				b64Data = fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(bytes))
			}
		}

		prompt := r.FormValue("prompt")
		model := r.FormValue("model")
		size := r.FormValue("size")
		style := r.FormValue("style")

		req := models.ImageGenerationRequest{
			Prompt: prompt,
			Model:  model,
			Size:   size,
			Style:  style,
			Image:  b64Data,
		}

		reqBody, _ := json.Marshal(req)
		r.Body = io.NopCloser(bytes.NewReader(reqBody))
		r.Header.Set("Content-Type", "application/json")
	}

	s.handleImageGenerations(w, r)
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

		var vidBytes []byte
		replyText := ""

		// 1. If provider supports native AI video generation (e.g. Google Veo 2 via Gemini)
		if vidGen, ok := prov.(providers.VideoGenerator); ok {
			log.Printf("[VIDEO GEN] Requesting native AI video generation from %s (prompt: %q, aspect: %s)...", account.Name, req.Prompt, req.AspectRatio)
			vBytes, revPrompt, vErr := vidGen.GenerateVideo(ctx, account.Cookie, req.Prompt, req.AspectRatio, req.Duration)
			if vErr == nil && len(vBytes) > 1000 {
				vidBytes = vBytes
				replyText = fmt.Sprintf("🎬 **Google Veo 2 AI Video (Master)**\n\n%s", revPrompt)
				log.Printf("[VIDEO GEN] Upstream %s generated real master AI video: %d bytes (%.2f MB)", account.Name, len(vidBytes), float64(len(vidBytes))/(1024*1024))
			} else {
				log.Printf("[VIDEO GEN] Upstream %s native video generation notice: %v (falling back to dynamic stream)", account.Name, vErr)
				replyText = fmt.Sprintf("⚡ **Video AI Cinematic Motion (Mode Dinamis)**\n\nPrompt: *%s*\n\n*(Catatan: Kuota harian Google Veo akun telah mencapai batas maksimum atau antrean render upstream penuh. Video disintesis dalam resolusi 1280×720 HD)*", req.Prompt)
			}
		}

		// 2. Fallback to zero-disk dynamic video if native generator is unavailable
		if len(vidBytes) == 0 {
			if replyText == "" {
				replyText = fmt.Sprintf("🎬 Video AI Sinematik: %s", req.Prompt)
			}

			// Generate zero-disk streaming MP4 video in memory
			bgCtx, bgCancel := context.WithTimeout(context.Background(), 45*time.Second)
			vBytes, err := media.GenerateDynamicVideo(bgCtx, req.Prompt, req.Model, req.Duration, req.AspectRatio)
			bgCancel()
			if err != nil {
				lastErr = err
				continue
			}
			vidBytes = vBytes
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
