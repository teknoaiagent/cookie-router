package gemini

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
)

type sessionCacheEntry struct {
	at        string
	bl        string
	expiresAt time.Time
}

type Provider struct {
	client       *http.Client
	sessionMu    sync.RWMutex
	sessionCache map[string]*sessionCacheEntry
}

func init() {
	providers.Register(New())
}

func New() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		sessionCache: make(map[string]*sessionCacheEntry),
	}
}

func (p *Provider) ID() string   { return "gemini" }
func (p *Provider) Name() string { return "Gemini Web (Google)" }

func (p *Provider) DefaultModels() []string {
	return []string{
		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-2.0-flash-thinking",
		"gemini-2.0-pro-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-exp-1206",
		"gemini-advanced",
		"gemini-auto",
		"imagen-3",
	}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func normalizeCookie(raw string) string {
	return utils.NormalizeCookie(raw, "gemini")
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	norm := normalizeCookie(cookie)
	if !strings.Contains(norm, "__Secure-1PSID") {
		return errors.New("cookie must contain __Secure-1PSID")
	}
	_, _, err := p.getTokens(ctx, norm)
	return err
}

func (p *Provider) getTokens(ctx context.Context, cookie string) (string, string, error) {
	norm := normalizeCookie(cookie)

	p.sessionMu.RLock()
	cached, ok := p.sessionCache[norm]
	p.sessionMu.RUnlock()

	if ok && cached != nil && time.Now().Before(cached.expiresAt) && cached.at != "" {
		return cached.at, cached.bl, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://gemini.google.com/app", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", norm)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("gemini web request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("gemini app returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	bodyStr := string(bodyBytes)

	var at, bl string

	// Extract from WIZ_global_data
	wizRegex := regexp.MustCompile(`window\.WIZ_global_data\s*=\s*(\{.*?\});`)
	wizMatch := wizRegex.FindStringSubmatch(bodyStr)
	if len(wizMatch) > 1 {
		var wizMap map[string]interface{}
		if err := json.Unmarshal([]byte(wizMatch[1]), &wizMap); err == nil {
			// In modern builds, 'thykhd' or 'SNlM0e' is the anti-CSRF token
			if val, ok := wizMap["thykhd"].(string); ok && val != "" {
				at = val
			} else if val, ok := wizMap["SNlM0e"].(string); ok && val != "" {
				at = val
			}

			if val, ok := wizMap["cfb2h"].(string); ok && val != "" {
				bl = val
			}
		}
	}

	// Fallback regex if WIZ_global_data JSON parse didn't find them
	if at == "" {
		atRegex := regexp.MustCompile(`\"(?:thykhd|SNlM0e)\"\s*:\s*\"([^\"]+)\"`)
		if m := atRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
			at = m[1]
		}
	}
	if at == "" {
		// Look for any AFWL token pattern
		afwlRegex := regexp.MustCompile(`\"(AFWL[a-zA-Z0-9_\-]+)\"`)
		if m := afwlRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
			at = m[1]
		}
	}

	if bl == "" {
		blRegex := regexp.MustCompile(`\"cfb2h\"\s*:\s*\"([^\"]+)\"`)
		if m := blRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
			bl = m[1]
		}
	}
	if bl == "" {
		bl = "boq_gemini-web-uiserver_20261012.19_p1"
	}

	if at == "" {
		return "", "", errors.New("failed to extract Google Gemini anti-CSRF token (at/thykhd) - cookie may be expired or invalid")
	}

	p.sessionMu.Lock()
	p.sessionCache[norm] = &sessionCacheEntry{
		at:        at,
		bl:        bl,
		expiresAt: time.Now().Add(45 * time.Minute),
	}
	p.sessionMu.Unlock()

	return at, bl, nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	norm := normalizeCookie(cookie)

	defaultList := p.DefaultModels()
	modelSet := make(map[string]bool)
	var result []string

	for _, m := range defaultList {
		if !modelSet[m] {
			modelSet[m] = true
			result = append(result, m)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://gemini.google.com/app", nil)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
		req.Header.Set("Cookie", norm)
		resp, err := p.client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyStr := string(bodyBytes)

			// Scan for dynamic gemini models
			mRegex := regexp.MustCompile(`gemini-[a-zA-Z0-9\.\-_]+`)
			matches := mRegex.FindAllString(bodyStr, -1)
			for _, m := range matches {
				mLower := strings.ToLower(m)
				// Filter out web assets/css classes, uiserver, js bundles
				if strings.Contains(mLower, "-bg") ||
					strings.Contains(mLower, "-text") ||
					strings.Contains(mLower, "-panel") ||
					strings.Contains(mLower, "-lesson") ||
					strings.Contains(mLower, "-server") ||
					strings.Contains(mLower, "uiserver") ||
					strings.Contains(mLower, "bardchatui") ||
					strings.Contains(mLower, ".") ||
					strings.Contains(mLower, "-apps") {
					continue
				}
				if !modelSet[mLower] {
					modelSet[mLower] = true
					result = append(result, mLower)
				}
			}
		}
	}

	return result, nil
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

	respID := "chatcmpl-gemini-" + randomUUID()[:8]
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

func buildPrompt(req *models.ChatCompletionRequest) string {
	if len(req.Messages) == 0 {
		return ""
	}
	if len(req.Messages) == 1 {
		return req.Messages[0].TextContent()
	}

	var sb strings.Builder
	for _, m := range req.Messages {
		role := strings.ToLower(m.Role)
		content := m.TextContent()
		switch role {
		case "system":
			sb.WriteString(fmt.Sprintf("[System Instructions: %s]\n\n", content))
		case "user":
			sb.WriteString(fmt.Sprintf("User: %s\n\n", content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("Assistant: %s\n\n", content))
		default:
			sb.WriteString(fmt.Sprintf("%s: %s\n\n", m.Role, content))
		}
	}
	sb.WriteString("Assistant: ")
	return strings.TrimSpace(sb.String())
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	norm := normalizeCookie(cookie)
	if !strings.Contains(norm, "__Secure-1PSID") {
		return errors.New("missing __Secure-1PSID in cookie")
	}

	at, bl, err := p.getTokens(ctx, norm)
	if err != nil {
		return err
	}

	prompt := buildPrompt(req)
	if prompt == "" {
		return errors.New("empty prompt in chat request")
	}

	// Model routing hints if applicable
	modelLower := strings.ToLower(req.Model)
	if strings.Contains(modelLower, "thinking") {
		prompt = "[Mode: Deep Thinking / Reasoning]\n" + prompt
	} else if strings.Contains(modelLower, "imagen") || strings.Contains(modelLower, "image") {
		if !strings.HasPrefix(strings.ToLower(prompt), "generate an image") {
			prompt = "Generate an image: " + prompt
		}
	}

	// Inner payload: [ [ [prompt], null, ["", "", ""] ] ]
	innerPayload := []interface{}{
		[]interface{}{prompt},
		nil,
		[]interface{}{"", "", ""},
	}
	innerBytes, err := json.Marshal(innerPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal inner payload: %w", err)
	}

	outerPayload := []interface{}{
		nil,
		string(innerBytes),
	}
	outerBytes, err := json.Marshal(outerPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal outer payload: %w", err)
	}

	// Generate random _reqid
	nBig, _ := rand.Int(rand.Reader, big.NewInt(900000))
	reqID := 100000 + nBig.Int64()

	genURL := fmt.Sprintf("https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate?bl=%s&_reqid=%d&rt=c",
		url.QueryEscape(bl), reqID)

	formData := url.Values{}
	formData.Set("f.req", string(outerBytes))
	formData.Set("at", at)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", genURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	httpReq.Header.Set("Cookie", norm)
	httpReq.Header.Set("Origin", "https://gemini.google.com")
	httpReq.Header.Set("Referer", "https://gemini.google.com/app")
	httpReq.Header.Set("Accept", "*/*")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gemini StreamGenerate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Invalidate cache if 401 or 403
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			p.sessionMu.Lock()
			delete(p.sessionCache, norm)
			p.sessionMu.Unlock()
		}
		bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("gemini StreamGenerate error HTTP %d: %s", resp.StatusCode, string(bodyPreview))
	}

	chunkID := "chatcmpl-gemini-" + randomUUID()[:8]
	scanner := bufio.NewScanner(resp.Body)
	// Buffer up to 1MB lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	prevLen := 0
	hasEmitted := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "[[\"wrb.fr\"") {
			continue
		}

		var envelope [][]interface{}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil || len(envelope) == 0 {
			continue
		}

		if len(envelope[0]) < 3 {
			continue
		}
		innerStr, ok := envelope[0][2].(string)
		if !ok || innerStr == "" {
			continue
		}

		var innerData []interface{}
		if err := json.Unmarshal([]byte(innerStr), &innerData); err != nil {
			continue
		}

		var textVal string

		// 1. Check Candidates at index 4
		if len(innerData) > 4 && innerData[4] != nil {
			if cands, ok := innerData[4].([]interface{}); ok && len(cands) > 0 {
				if cand0, ok := cands[0].([]interface{}); ok && len(cand0) >= 2 {
					if candParts, ok := cand0[1].([]interface{}); ok && len(candParts) > 0 {
						if tv, ok := candParts[0].(string); ok {
							textVal = tv
						}
					}
				}
			}
		}

		// 2. If candidates text is empty, check tool / video_generation / media objects in innerData
		if textVal == "" {
			for _, item := range innerData {
				if obj, ok := item.(map[string]interface{}); ok {
					if val7, ok := obj["7"].([]interface{}); ok && len(val7) > 0 {
						if msg, ok := val7[0].(string); ok && msg != "" {
							textVal = msg
							break
						}
					}
				}
			}
		}

		if textVal == "" {
			continue
		}

		if len(textVal) > prevLen {
			delta := textVal[prevLen:]
			prevLen = len(textVal)
			hasEmitted = true

			chunkChan <- &models.ChatCompletionChunk{
				ID:      chunkID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
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

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("reading gemini response stream: %w", err)
	}

	if !hasEmitted {
		return errors.New("gemini returned empty response text")
	}

	stop := "stop"
	chunkChan <- &models.ChatCompletionChunk{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChunkChoice{
			{
				Index:        0,
				FinishReason: &stop,
			},
		},
	}

	return nil
}
