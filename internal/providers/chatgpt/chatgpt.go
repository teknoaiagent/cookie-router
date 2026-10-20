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
	"log"
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
	userEmail   string
	userName    string
	planType    string
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
		User        struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
		Account struct {
			ID       string `json:"id"`
			PlanType string `json:"planType"`
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
		userEmail:   sessionData.User.Email,
		userName:    sessionData.User.Name,
		planType:    sessionData.Account.PlanType,
		expiresAt:   exp,
	}
	p.tokenCache.Store(cacheKey, cached)

	return cached, nil
}

func (p *Provider) DetectAccountInfo(ctx context.Context, cookie string) (*providers.AccountInfo, error) {
	tok, err := p.getSession(ctx, cookie)
	if err != nil {
		return nil, err
	}
	plan := tok.planType
	if plan == "" {
		if tok.accountID != "" {
			plan = "plus"
		} else {
			plan = "free"
		}
	}
	return &providers.AccountInfo{
		Email: tok.userEmail,
		Name:  tok.userName,
		Plan:  plan,
	}, nil
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

func cleanChatGPTModelSlug(model string) (string, string) {
	m := strings.ToLower(strings.TrimSpace(model))
	effort := ""
	if strings.Contains(m, "instant") || strings.Contains(m, "low") {
		effort = "low"
	} else if strings.Contains(m, "medium") {
		effort = "medium"
	} else if strings.Contains(m, "thinking") || strings.Contains(m, "high") {
		effort = "high"
	}

	slug := "auto"
	if strings.Contains(m, "gpt-6") {
		slug = "gpt-6"
	} else if strings.Contains(m, "gpt-5-6") {
		slug = "gpt-5-6"
	} else if strings.Contains(m, "gpt-4o-mini") {
		slug = "gpt-4o-mini"
	} else if strings.Contains(m, "gpt-4o") {
		slug = "gpt-4o"
	} else if strings.Contains(m, "o1-mini") {
		slug = "o1-mini"
	} else if strings.Contains(m, "o1") {
		slug = "o1"
	} else if strings.Contains(m, "o3-mini") {
		slug = "o3-mini"
	} else if strings.Contains(m, "dall-e-3") || strings.Contains(m, "gpt-image-2") {
		slug = "gpt-image-2"
	} else if m != "" && m != "auto" {
		slug = m
	}
	return slug, effort
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

	// Collect any attached images (multimodal vision / img2img)
	var attachedImageURLs []string
	for _, m := range req.Messages {
		if urls := m.ImageURLs(); len(urls) > 0 {
			attachedImageURLs = append(attachedImageURLs, urls...)
		}
	}

	userUploadedFileIDs := make(map[string]bool)
	var contentParts []interface{}
	for _, imgURL := range attachedImageURLs {
		imgBytes, mimeType, err := utils.ParseImageData(imgURL)
		if err == nil && len(imgBytes) > 0 {
			fileID, err := p.UploadImageFile(ctx, cookie, "uploaded_image.png", imgBytes, mimeType)
			if err == nil && fileID != "" {
				userUploadedFileIDs[fileID] = true
				contentParts = append(contentParts, map[string]interface{}{
					"content_type":  "image_asset_pointer",
					"asset_pointer": "file-service://" + fileID,
					"size_bytes":    len(imgBytes),
					"width":         1024,
					"height":        1024,
				})
				log.Printf("[CHATGPT] Multimodal image uploaded: %s (%d bytes)", fileID, len(imgBytes))
			} else {
				log.Printf("[CHATGPT] Image upload notice: %v", err)
			}
		}
	}

	slug, thinkingEffort := cleanChatGPTModelSlug(req.Model)
	modelSlug := slug

	if thinkingEffort != "" {
		effortDirective := ""
		switch thinkingEffort {
		case "low":
			effortDirective = "[Thinking Effort: Instant / Low - Jawab langsung, ringkas, dan cepat tanpa penalaran panjang]\n\n"
		case "medium":
			effortDirective = "[Thinking Effort: Medium - Penalaran berimbang dan terstruktur]\n\n"
		case "high":
			effortDirective = "[Thinking Effort: High - Gunakan penalaran mendalam, teliti, dan analisis menyeluruh]\n\n"
		}
		combinedUserMessage = effortDirective + combinedUserMessage
	}

	userMsgContent := map[string]interface{}{}
	if len(contentParts) > 0 {
		contentParts = append(contentParts, combinedUserMessage)
		userMsgContent["content_type"] = "multimodal_text"
		userMsgContent["parts"] = contentParts
	} else {
		userMsgContent["content_type"] = "text"
		userMsgContent["parts"] = []string{combinedUserMessage}
	}

	payload := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{
			{
				"id":      randomUUID(),
				"author":  map[string]string{"role": "user"},
				"content": userMsgContent,
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
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

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
			log.Printf("[RAW-SSE] [DONE]")
			break
		}
		if len(data) > 0 {
			log.Printf("[RAW-SSE-FULL] %s", data)
		}

		if len(data) > 0 && !strings.HasPrefix(data, "{") {
			continue
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
			log.Printf("[CHATGPT-SSE-ERR] event error: %v", event.Error)
			return fmt.Errorf("ChatGPT stream event error: %v", event.Error)
		}

		if event.Message.Author.Role != "" {
			log.Printf("[CHATGPT-SSE] role=%s, parts=%d, ctype=%s", event.Message.Author.Role, len(event.Message.Content.Parts), event.Message.Content.ContentType)
		}

		// Detect generated images in multimodal parts (delivered via tool or assistant)
		// Never treat user's uploaded reference image as the generated image output
		if event.Message.Author.Role != "user" && event.Message.Author.Role != "system" {
			for _, part := range event.Message.Content.Parts {
				if partMap, ok := part.(map[string]interface{}); ok {
					cType, _ := partMap["content_type"].(string)
					assetPtr, _ := partMap["asset_pointer"].(string)
					if cType == "image_asset_pointer" || strings.HasPrefix(assetPtr, "sediment://") || strings.HasPrefix(assetPtr, "file-service://") {
						cleanFileID := strings.TrimPrefix(strings.TrimPrefix(assetPtr, "sediment://"), "file-service://")
						if cleanFileID != "" && !seenImages[cleanFileID] && !userUploadedFileIDs[cleanFileID] {
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
	if err := scanner.Err(); err != nil {
		log.Printf("[CHATGPT-SCANNER-ERR] %v", err)
	}

	// If no image was found in the initial SSE stream, poll the conversation endpoint
	// ONLY if this was an image generation request (e.g. DALL-E / Plus async image generator)
	isImageReq := strings.Contains(strings.ToLower(req.Model), "image") ||
		strings.Contains(strings.ToLower(req.Model), "dall-e") ||
		strings.Contains(strings.ToLower(currentPrompt), "buat gambar") ||
		strings.Contains(strings.ToLower(currentPrompt), "buatkan gambar") ||
		strings.Contains(strings.ToLower(currentPrompt), "buat foto") ||
		strings.Contains(strings.ToLower(currentPrompt), "buatkan foto") ||
		strings.Contains(strings.ToLower(currentPrompt), "generate image") ||
		strings.Contains(strings.ToLower(currentPrompt), "create image")

	if isImageReq && len(seenImages) == 0 && convID != "" {
		pollCtx, pollCancel := context.WithTimeout(ctx, 150*time.Second)
		defer pollCancel()

		log.Printf("[IMAGE GEN ASYNC] Polling conversation %s for async image completion (timeout 150s)...", convID)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		pollCount := 0

	pollLoop:
		for {
			select {
			case <-pollCtx.Done():
				log.Printf("[IMAGE GEN ASYNC] Polling timeout reached for conversation %s", convID)
				break pollLoop
			case <-ticker.C:
				pollCount++
				convURL := fmt.Sprintf("%s/backend-api/conversation/%s", ChatGptBase, convID)
				pReq, pErr := http.NewRequestWithContext(pollCtx, "GET", convURL, nil)
				if pErr != nil {
					continue
				}
				pReq.Header.Set("User-Agent", UserAgent)
				pReq.Header.Set("Authorization", "Bearer "+accessToken)
				if accountID != "" {
					pReq.Header.Set("ChatGPT-Account-Id", accountID)
				}
				pReq.Header.Set("Cookie", cookieHeader)
				pReq.Header.Set("Origin", ChatGptBase)
				pReq.Header.Set("Referer", ChatGptBase+"/")

				pResp, pErr := p.client.Do(pReq)
				if pErr != nil || pResp.StatusCode != 200 {
					if pResp != nil {
						if pollCount%5 == 0 {
							log.Printf("[IMAGE GEN ASYNC] Poll %d: status %d", pollCount, pResp.StatusCode)
						}
						pResp.Body.Close()
					}
					continue
				}

				var convData struct {
					Mapping map[string]struct {
						Message *struct {
							ID      string `json:"id"`
							Author  struct {
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
					} `json:"mapping"`
				}

				dErr := json.NewDecoder(pResp.Body).Decode(&convData)
				pResp.Body.Close()
				if dErr != nil {
					continue
				}

				if pollCount%5 == 0 {
					log.Printf("[IMAGE GEN ASYNC] Poll %d: %d nodes in mapping", pollCount, len(convData.Mapping))
				}

				foundImage := false
				for _, node := range convData.Mapping {
					if node.Message == nil || node.Message.Author.Role == "user" || node.Message.Author.Role == "system" {
						continue
					}
					for _, part := range node.Message.Content.Parts {
						if partMap, ok := part.(map[string]interface{}); ok {
							cType, _ := partMap["content_type"].(string)
							assetPtr, _ := partMap["asset_pointer"].(string)
							if cType == "image_asset_pointer" || strings.HasPrefix(assetPtr, "sediment://") || strings.HasPrefix(assetPtr, "file-service://") {
								cleanFileID := strings.TrimPrefix(strings.TrimPrefix(assetPtr, "sediment://"), "file-service://")
								if cleanFileID != "" && !seenImages[cleanFileID] && !userUploadedFileIDs[cleanFileID] {
									seenImages[cleanFileID] = true
									foundImage = true
									title := node.Message.Metadata.ImageGenTitle
									if title == "" {
										title = node.Message.Metadata.Dalle.Prompt
									}
									if title == "" {
										title = "Generated Image"
									}

									log.Printf("[IMAGE GEN ASYNC] Found completed image: %s (title: %s)", cleanFileID, title)
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
				}

				if foundImage {
					break pollLoop
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

// UploadImageFile uploads an image to ChatGPT backend-api files storage and returns the file_id
func (p *Provider) UploadImageFile(ctx context.Context, cookie string, filename string, data []byte, mimeType string) (string, error) {
	normCookie := utils.NormalizeCookie(cookie, "chatgpt")
	tok, err := p.getSession(ctx, normCookie)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	deviceID := extractCookieField(normCookie, "oai-did")
	if deviceID == "" {
		deviceID = randomUUID()
	}

	if filename == "" {
		filename = "image.png"
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	// 1. Initialize file record
	initPayload := map[string]interface{}{
		"file_name": filename,
		"file_size": len(data),
		"use_case":  "multimodal",
	}
	initBytes, err := json.Marshal(initPayload)
	if err != nil {
		return "", err
	}

	req1, err := http.NewRequestWithContext(ctx, "POST", ChatGptBase+"/backend-api/files", bytes.NewReader(initBytes))
	if err != nil {
		return "", err
	}
	req1.Header.Set("User-Agent", UserAgent)
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+tok.accessToken)
	if tok.accountID != "" {
		req1.Header.Set("ChatGPT-Account-Id", tok.accountID)
	}
	req1.Header.Set("Cookie", normCookie)
	req1.Header.Set("OAI-Device-Id", deviceID)
	req1.Header.Set("Origin", ChatGptBase)
	req1.Header.Set("Referer", ChatGptBase+"/")

	resp1, err := p.client.Do(req1)
	if err != nil {
		return "", fmt.Errorf("failed to call files init: %w", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp1.Body)
		return "", fmt.Errorf("file init returned status %d: %s", resp1.StatusCode, string(b))
	}

	var initResp struct {
		Status    string `json:"status"`
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&initResp); err != nil {
		return "", fmt.Errorf("failed to decode file init response: %w", err)
	}

	if initResp.UploadURL == "" || initResp.FileID == "" {
		return "", errors.New("empty upload_url or file_id from ChatGPT")
	}

	// 2. Upload file binary to Azure / storage blob
	req2, err := http.NewRequestWithContext(ctx, "PUT", initResp.UploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req2.Header.Set("x-ms-blob-type", "BlockBlob")
	req2.Header.Set("Content-Type", mimeType)

	resp2, err := p.client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("failed to upload binary blob: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		b, _ := io.ReadAll(resp2.Body)
		return "", fmt.Errorf("blob upload returned status %d: %s", resp2.StatusCode, string(b))
	}

	// 3. Mark file as uploaded
	req3, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/backend-api/files/%s/uploaded", ChatGptBase, initResp.FileID), strings.NewReader("{}"))
	if err != nil {
		return "", err
	}
	req3.Header.Set("User-Agent", UserAgent)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+tok.accessToken)
	if tok.accountID != "" {
		req3.Header.Set("ChatGPT-Account-Id", tok.accountID)
	}
	req3.Header.Set("Cookie", normCookie)
	req3.Header.Set("OAI-Device-Id", deviceID)
	req3.Header.Set("Origin", ChatGptBase)
	req3.Header.Set("Referer", ChatGptBase+"/")

	resp3, err := p.client.Do(req3)
	if err != nil {
		return "", fmt.Errorf("failed to mark file uploaded: %w", err)
	}
	defer resp3.Body.Close()

	return initResp.FileID, nil
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
