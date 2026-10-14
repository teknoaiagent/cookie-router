package chatgpt

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
)

const (
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	ChatGptBase = "https://chatgpt.com"
)

type Provider struct {
	client     *http.Client
	tokenCache sync.Map // cookieKey -> *cachedToken
}

type cachedToken struct {
	accessToken string
	accountID   string
	expiresAt   time.Time
}

func init() {
	providers.Register(New())
}

func New() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *Provider) ID() string   { return "chatgpt" }
func (p *Provider) Name() string { return "ChatGPT Web (Plus/Free)" }
func (p *Provider) DefaultModels() []string {
	return []string{"gpt-4o", "gpt-4o-mini", "gpt-image-2", "dall-e-3", "o1", "o3-mini", "auto"}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func normalizeCookie(raw string) string {
	return utils.NormalizeCookie(raw, "chatgpt")
}

func (p *Provider) getSession(ctx context.Context, rawCookie string) (*cachedToken, error) {
	cookieHeader := normalizeCookie(rawCookie)
	cacheKey := fmt.Sprintf("%x", hex.EncodeToString([]byte(cookieHeader)[:min(32, len(cookieHeader))]))

	if val, ok := p.tokenCache.Load(cacheKey); ok {
		tok := val.(*cachedToken)
		if time.Now().Before(tok.expiresAt) {
			return tok, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ChatGptBase+"/api/auth/session", nil)
	if err != nil {
		return nil, err
	}

	deviceID := extractCookieField(cookieHeader, "oai-did")
	if deviceID == "" {
		deviceID = randomUUID()
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", ChatGptBase+"/")
	req.Header.Set("Origin", ChatGptBase)
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("OAI-Language", "en-US")
	req.Header.Set("OAI-Device-Id", deviceID)
	req.Header.Set("Cookie", cookieHeader)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ChatGPT session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var sessionData struct {
		AccessToken string `json:"accessToken"`
		Expires     string `json:"expires"`
		Error       string `json:"error"`
		Account     struct {
			ID string `json:"id"`
		} `json:"account"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionData); err != nil {
		return nil, fmt.Errorf("invalid json in session response: %w", err)
	}

	if sessionData.Error != "" {
		return nil, fmt.Errorf("session error: %s", sessionData.Error)
	}

	if sessionData.AccessToken == "" {
		return nil, errors.New("no accessToken in session response (cookie might be expired)")
	}

	exp := time.Now().Add(5 * time.Minute)
	if sessionData.Expires != "" {
		if t, err := time.Parse(time.RFC3339, sessionData.Expires); err == nil {
			if t.Before(exp) {
				exp = t
			}
		}
	}

	cached := &cachedToken{
		accessToken: sessionData.AccessToken,
		accountID:   sessionData.Account.ID,
		expiresAt:   exp,
	}
	p.tokenCache.Store(cacheKey, cached)

	return cached, nil
}

func (p *Provider) getAccessToken(ctx context.Context, rawCookie string) (string, error) {
	tok, err := p.getSession(ctx, rawCookie)
	if err != nil {
		return "", err
	}
	return tok.accessToken, nil
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	tok, err := p.getAccessToken(ctx, cookie)
	if err != nil {
		return err
	}
	if tok == "" {
		return errors.New("empty accessToken received")
	}
	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	sess, err := p.getSession(ctx, cookie)
	if err != nil {
		return p.DefaultModels(), err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ChatGptBase+"/backend-api/models?history_and_training_disabled=false", nil)
	if err != nil {
		return p.DefaultModels(), nil
	}

	cookieHeader := normalizeCookie(cookie)
	deviceID := extractCookieField(cookieHeader, "oai-did")
	if deviceID == "" {
		deviceID = randomUUID()
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+sess.accessToken)
	if sess.accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", sess.accountID)
	}
	req.Header.Set("Referer", ChatGptBase+"/")
	req.Header.Set("Origin", ChatGptBase)
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("OAI-Language", "en-US")
	req.Header.Set("OAI-Device-Id", deviceID)
	req.Header.Set("Cookie", cookieHeader)

	resp, err := p.client.Do(req)
	if err != nil {
		return p.DefaultModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var data struct {
			Models []struct {
				Slug string `json:"slug"`
			} `json:"models"`
			Categories []struct {
				Category string `json:"category"`
				Models   []struct {
					Slug string `json:"slug"`
				} `json:"models"`
			} `json:"categories"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
			var modelList []string
			seen := make(map[string]bool)
			for _, m := range data.Models {
				if m.Slug != "" && !seen[m.Slug] {
					seen[m.Slug] = true
					modelList = append(modelList, m.Slug)
				}
			}
			for _, cat := range data.Categories {
				for _, m := range cat.Models {
					if m.Slug != "" && !seen[m.Slug] {
						seen[m.Slug] = true
						modelList = append(modelList, m.Slug)
					}
				}
			}
			if len(modelList) > 0 {
				return modelList, nil
			}
		}
	}

	return p.DefaultModels(), nil
}

func extractCookieField(cookieHeader, name string) string {
	parts := strings.Split(cookieHeader, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, name+"=") {
			return strings.TrimPrefix(p, name+"=")
		}
	}
	return ""
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatCompletionRequest, cookie string) (*models.ChatCompletionResponse, error) {
	chunkChan := make(chan *models.ChatCompletionChunk, 20)
	errChan := make(chan error, 1)

	var fullContent strings.Builder
	var lastChunk *models.ChatCompletionChunk

	go func() {
		errChan <- p.ChatStream(ctx, req, cookie, chunkChan)
		close(chunkChan)
	}()

	for chunk := range chunkChan {
		lastChunk = chunk
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	if err := <-errChan; err != nil {
		return nil, err
	}

	respID := "chatcmpl-cgpt-" + randomUUID()[:8]
	if lastChunk != nil && lastChunk.ID != "" {
		respID = lastChunk.ID
	}

	finishReason := "stop"
	return &models.ChatCompletionResponse{
		ID:      respID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: fullContent.String(),
				},
				FinishReason: &finishReason,
			},
		},
		Usage: models.ChatCompletionUsage{
			PromptTokens:     len(req.Messages) * 15,
			CompletionTokens: len(fullContent.String()) / 4,
			TotalTokens:      len(req.Messages)*15 + len(fullContent.String())/4,
		},
	}, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	sess, err := p.getSession(ctx, cookie)
	if err != nil {
		return err
	}
	accessToken := sess.accessToken
	accountID := sess.accountID

	cookieHeader := normalizeCookie(cookie)

	// Build messages
	var systemPrompt string
	var conversationHistory []string
	var currentPrompt string

	for i, m := range req.Messages {
		text := m.TextContent()
		if m.Role == "system" {
			systemPrompt = text
		} else if i == len(req.Messages)-1 && m.Role == "user" {
			currentPrompt = text
		} else {
			roleLabel := "User"
			if m.Role == "assistant" {
				roleLabel = "Assistant"
			}
			conversationHistory = append(conversationHistory, fmt.Sprintf("%s: %s", roleLabel, text))
		}
	}

	var parts []string
	if systemPrompt != "" {
		parts = append(parts, fmt.Sprintf("System Instructions: %s", systemPrompt))
	}
	if len(conversationHistory) > 0 {
		parts = append(parts, fmt.Sprintf("Context:\n%s", strings.Join(conversationHistory, "\n")))
	}
	parts = append(parts, currentPrompt)
	combinedUserMessage := strings.Join(parts, "\n\n")

	modelSlug := strings.TrimSpace(req.Model)
	if modelSlug == "" {
		modelSlug = "auto"
	}

	payload := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{
			{
				"id":     randomUUID(),
				"author": map[string]string{"role": "user"},
				"content": map[string]interface{}{
					"content_type": "text",
					"parts":        []string{combinedUserMessage},
				},
			},
		},
		"model":                         modelSlug,
		"parent_message_id":             randomUUID(),
		"timezone_offset_min":           -420,
		"history_and_training_disabled": false,
		"conversation_mode":             map[string]string{"kind": "primary_assistant"},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ChatGptBase+"/backend-api/f/conversation", bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	if accountID != "" {
		httpReq.Header.Set("ChatGPT-Account-Id", accountID)
	}
	httpReq.Header.Set("Cookie", cookieHeader)
	httpReq.Header.Set("Origin", ChatGptBase)
	httpReq.Header.Set("Referer", ChatGptBase+"/")
	httpReq.Header.Set("Sec-Ch-Ua", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
	httpReq.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	httpReq.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	httpReq.Header.Set("Sec-Fetch-Dest", "empty")
	httpReq.Header.Set("Sec-Fetch-Mode", "cors")
	httpReq.Header.Set("Sec-Fetch-Site", "same-origin")
	httpReq.Header.Set("OAI-Language", "en-US")
	streamDeviceID := extractCookieField(cookieHeader, "oai-did")
	if streamDeviceID == "" {
		streamDeviceID = randomUUID()
	}
	httpReq.Header.Set("OAI-Device-Id", streamDeviceID)

	reqToken, proofToken, sErr := p.getSentinelTokens(ctx, accessToken, accountID, cookieHeader, streamDeviceID)
	if sErr == nil {
		if reqToken != "" {
			httpReq.Header.Set("Openai-Sentinel-Chat-Requirements-Token", reqToken)
		}
		if proofToken != "" {
			httpReq.Header.Set("Openai-Sentinel-Proof-Token", proofToken)
		}
	}

	for k, v := range httpReq.Header {
		fmt.Printf("[DEBUG-HEADER] %s: %v\n", k, v)
	}
	fmt.Printf("[DEBUG-CHATGPT] URL: %s, Payload: %s\n", httpReq.URL.String(), string(jsonBytes))
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ChatGPT conversation request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG-CHATGPT-RESP] Status: %d, URL: %s\n", resp.StatusCode, httpReq.URL.String())

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[DEBUG-RESP-DETAIL] Status: %d, Headers: %v, Body: %s\n", resp.StatusCode, resp.Header, string(body))
		return fmt.Errorf("ChatGPT API error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	scanner := bufio.NewScanner(reader)

	chunkID := "chatcmpl-cgpt-" + randomUUID()[:8]
	var previousText string
	var convID string
	seenImages := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			ConversationID string `json:"conversation_id"`
			Message        struct {
				ID     string `json:"id"`
				Author struct {
					Role string `json:"role"`
					Name string `json:"name"`
				} `json:"author"`
				Content struct {
					ContentType string        `json:"content_type"`
					Parts       []interface{} `json:"parts"`
				} `json:"content"`
				Metadata struct {
					ImageGenTitle string `json:"image_gen_title"`
					Dalle         struct {
						Prompt string `json:"prompt"`
					} `json:"dalle"`
				} `json:"metadata"`
			} `json:"message"`
			Error interface{} `json:"error"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.ConversationID != "" {
			convID = event.ConversationID
		}

		if event.Error != nil {
			return fmt.Errorf("ChatGPT stream event error: %v", event.Error)
		}

		// Detect generated images in multimodal parts (delivered via tool or assistant)
		for _, part := range event.Message.Content.Parts {
			if partMap, ok := part.(map[string]interface{}); ok {
				cType, _ := partMap["content_type"].(string)
				assetPtr, _ := partMap["asset_pointer"].(string)
				if cType == "image_asset_pointer" || strings.HasPrefix(assetPtr, "sediment://") || strings.HasPrefix(assetPtr, "file-service://") {
					cleanFileID := strings.TrimPrefix(strings.TrimPrefix(assetPtr, "sediment://"), "file-service://")
					if cleanFileID != "" && !seenImages[cleanFileID] {
						seenImages[cleanFileID] = true
						title := event.Message.Metadata.ImageGenTitle
						if title == "" {
							title = event.Message.Metadata.Dalle.Prompt
						}
						if title == "" {
							title = "Generated Image"
						}

						imgMarkdown := fmt.Sprintf("\n\n![%s](http://localhost:18899/v1/images/%s.png)\n\n", title, cleanFileID)
						chunkChan <- &models.ChatCompletionChunk{
							ID:      chunkID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   modelSlug,
							Choices: []models.ChunkChoice{
								{
									Index: 0,
									Delta: models.ChunkDelta{
										Content: imgMarkdown,
									},
								},
							},
						}
					}
				}
			}
		}

		if event.Message.Author.Role != "assistant" {
			continue
		}

		if len(event.Message.Content.Parts) > 0 {
			if partStr, ok := event.Message.Content.Parts[0].(string); ok {
				if len(partStr) > len(previousText) {
					delta := partStr[len(previousText):]
					previousText = partStr

					chunkChan <- &models.ChatCompletionChunk{
						ID:      chunkID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   modelSlug,
						Choices: []models.ChunkChoice{
							{
								Index: 0,
								Delta: models.ChunkDelta{
									Content: delta,
								},
							},
						},
					}
				}
			}
		}
	}

	// Auto-clean conversation from sidebar in background
	if convID != "" {
		go func(cID string, tok string, c string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = p.HideConversation(ctx, c, cID)
		}(convID, accessToken, cookie)
	}

	stopReason := "stop"
	chunkChan <- &models.ChatCompletionChunk{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelSlug,
		Choices: []models.ChunkChoice{
			{
				Index:        0,
				FinishReason: &stopReason,
			},
		},
	}

	return scanner.Err()
}

// DownloadImage retrieves the generated image bytes from ChatGPT estuary
func (p *Provider) DownloadImage(ctx context.Context, cookie string, fileID string) ([]byte, error) {
	normCookie := utils.NormalizeCookie(cookie, "chatgpt")
	accessToken, err := p.getAccessToken(ctx, normCookie)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	deviceID := extractCookieField(normCookie, "oai-did")
	if deviceID == "" {
		deviceID = randomUUID()
	}

	downloadInfoURL := fmt.Sprintf("%s/backend-api/files/%s/download", ChatGptBase, fileID)
	req1, err := http.NewRequestWithContext(ctx, "GET", downloadInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req1.Header.Set("User-Agent", UserAgent)
	req1.Header.Set("Authorization", "Bearer "+accessToken)
	req1.Header.Set("Cookie", normCookie)
	req1.Header.Set("OAI-Device-Id", deviceID)

	resp1, err := p.client.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("failed to call file download endpoint: %w", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp1.Body)
		return nil, fmt.Errorf("download endpoint returned status %d: %s", resp1.StatusCode, string(b))
	}

	var dlInfo struct {
		Status      string `json:"status"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&dlInfo); err != nil {
		return nil, fmt.Errorf("failed to decode download info: %w", err)
	}

	if dlInfo.DownloadURL == "" {
		return nil, errors.New("empty download_url from ChatGPT")
	}

	req2, err := http.NewRequestWithContext(ctx, "GET", dlInfo.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("User-Agent", UserAgent)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	req2.Header.Set("Cookie", normCookie)
	req2.Header.Set("OAI-Device-Id", deviceID)

	resp2, err := p.client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image content: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		return nil, fmt.Errorf("image content fetch returned status %d: %s", resp2.StatusCode, string(b))
	}

	return io.ReadAll(resp2.Body)
}

// HideConversation archives/hides the conversation so it does not appear in user sidebar
func (p *Provider) HideConversation(ctx context.Context, cookie string, convID string) error {
	if convID == "" {
		return nil
	}
	normCookie := utils.NormalizeCookie(cookie, "chatgpt")
	accessToken, err := p.getAccessToken(ctx, normCookie)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/backend-api/conversation/%s", ChatGptBase, convID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(`{"is_visible":false}`))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Cookie", normCookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
