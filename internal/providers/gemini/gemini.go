package gemini

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cookie-router/internal/media"
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
		"gemini-4-argon",
		"gemini-4-argon-high",
		"gemini-4-argon-medium",
		"gemini-4-argon-low",
		"gemini-3.1-pro",
		"gemini-3.8-flash",
		"gemini-3.5-flash-lite",
		"imagen-3",
		"nano-banana",
		"gemini-auto",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
		"gemini-2.0-flash-thinking",
		"gemini-advanced",
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

	var cookieSNlM0e string
	for _, part := range strings.Split(norm, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "SNlM0e=") {
			cookieSNlM0e = strings.TrimPrefix(part, "SNlM0e=")
			break
		}
	}

	p.sessionMu.RLock()
	cached, ok := p.sessionCache[norm]
	p.sessionMu.RUnlock()

	if ok && cached != nil && time.Now().Before(cached.expiresAt) && cached.at != "" {
		return cached.at, cached.bl, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://gemini.google.com/app", nil)
	if err != nil {
		if cookieSNlM0e != "" {
			return cookieSNlM0e, "boq_gemini-web-uiserver_20261016.00_p0", nil
		}
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", stripCustomTokens(norm))
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		if cookieSNlM0e != "" {
			return cookieSNlM0e, "boq_gemini-web-uiserver_20261016.00_p0", nil
		}
		return "", "", fmt.Errorf("gemini web request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if cookieSNlM0e != "" {
			return cookieSNlM0e, "boq_gemini-web-uiserver_20261016.00_p0", nil
		}
		return "", "", fmt.Errorf("gemini app returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if cookieSNlM0e != "" {
			return cookieSNlM0e, "boq_gemini-web-uiserver_20261016.00_p0", nil
		}
		return "", "", err
	}
	bodyStr := string(bodyBytes)

	var at, bl string

	if cookieSNlM0e != "" {
		at = cookieSNlM0e
	}

	// Extract from WIZ_global_data
	wizRegex := regexp.MustCompile(`window\.WIZ_global_data\s*=\s*(\{.*?\});`)
	wizMatch := wizRegex.FindStringSubmatch(bodyStr)
	if len(wizMatch) > 1 {
		var wizMap map[string]interface{}
		if err := json.Unmarshal([]byte(wizMatch[1]), &wizMap); err == nil {
			// In modern builds, 'SNlM0e' is the primary anti-CSRF token for StreamGenerate
			if at == "" {
				if val, ok := wizMap["SNlM0e"].(string); ok && val != "" {
					at = val
				} else if val, ok := wizMap["thykhd"].(string); ok && val != "" {
					at = val
				}
			}

			if val, ok := wizMap["cfb2h"].(string); ok && val != "" {
				bl = val
			}
		}
	}

	// Fallback regex if WIZ_global_data JSON parse didn't find them
	if at == "" {
		atRegex := regexp.MustCompile(`\"SNlM0e\"\s*:\s*\"([^\"]+)\"`)
		if m := atRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
			at = m[1]
		}
	}
	if at == "" && cookieSNlM0e != "" {
		at = cookieSNlM0e
	}
	if at == "" {
		atRegex := regexp.MustCompile(`\"thykhd\"\s*:\s*\"([^\"]+)\"`)
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
		bl = "boq_gemini-web-uiserver_20261016.00_p0"
	}

	if at == "" {
		return "", "", errors.New("failed to extract Google Gemini anti-CSRF token (SNlM0e) - cookie may be expired or invalid")
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

func stripCustomTokens(cookie string) string {
	var clean []string
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "SNlM0e=") && !strings.HasPrefix(part, "thykhd=") {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, "; ")
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
					strings.Contains(mLower, "-preview") ||
					strings.Contains(mLower, "preview") ||
					strings.Contains(mLower, "2025") ||
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

func getModeID(model string) string {
	m := strings.ToLower(model)
	if strings.Contains(m, "3.5-flash-lite") || strings.Contains(m, "flash-lite") {
		return "8c46e95b1a07cecc"
	} else if strings.Contains(m, "3.8-flash") {
		return "56fdd199312815e2"
	} else if strings.Contains(m, "3.1-pro") {
		return "e6fa609c3fa255c0"
	} else if strings.Contains(m, "argon") {
		return "95a4c0abcba24364"
	} else if strings.Contains(m, "pro") {
		return "e6fa609c3fa255c0"
	} else if strings.Contains(m, "flash") {
		return "56fdd199312815e2"
	}
	return "95a4c0abcba24364"
}

func isCDPPortOpen() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:9242", 30*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func syncLiveGeminiCookie() string {
	if !isCDPPortOpen() {
		return ""
	}

	var pyCmd string
	for _, candidate := range []string{"python", "python3"} {
		if path, err := exec.LookPath(candidate); err == nil && !strings.Contains(strings.ToLower(path), "windowsapps") {
			pyCmd = candidate
			break
		}
	}
	if pyCmd == "" {
		return ""
	}

	cmd := exec.Command(pyCmd, "-c", `
import urllib.request, json, asyncio, websockets
async def run():
    try:
        tabs = json.loads(urllib.request.urlopen("http://127.0.0.1:9242/json/list", timeout=1.5).read())
        t1 = [t for t in tabs if "gemini.google.com" in t.get("url", "")][0]
        async with websockets.connect(t1["webSocketDebuggerUrl"]) as ws:
            await ws.send(json.dumps({"id": 1, "method": "Network.getCookies", "params": {"urls": ["https://gemini.google.com"]}}))
            cookies = json.loads(await ws.recv())["result"]["cookies"]
            await ws.send(json.dumps({"id": 2, "method": "Runtime.evaluate", "params": {"expression": "window.WIZ_global_data?.SNlM0e", "returnByValue": True}}))
            sn = json.loads(await ws.recv())["result"]["result"]["value"]
            parts = ["SNlM0e=" + str(sn)] + [c["name"] + "=" + c["value"] for c in cookies]
            cookie_str = "; ".join(parts)
            with open("config.json", "r", encoding="utf-8") as f:
                cfg = json.load(f)
            for a in cfg.get("accounts", []):
                if a.get("provider") == "gemini":
                    a["cookie"] = cookie_str
            with open("config.json", "w", encoding="utf-8") as f:
                json.dump(cfg, f, indent=2)
            print(cookie_str)
    except Exception:
        pass
asyncio.run(run())
`)
	out, err := cmd.Output()
	if err == nil && len(out) > 50 {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	norm := normalizeCookie(cookie)
	if liveCookie := syncLiveGeminiCookie(); liveCookie != "" {
		norm = normalizeCookie(liveCookie)
	}
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

	// Extract latest user message so image requests are never contaminated with previous chat transcripts
	var lastUserMsg string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(req.Messages[i].Role, "user") {
			lastUserMsg = strings.TrimSpace(req.Messages[i].TextContent())
			break
		}
	}
	if lastUserMsg == "" {
		lastUserMsg = prompt
	}

	// Check if this is an image generation request
	modelLower := strings.ToLower(req.Model)
	lowerPrompt := strings.ToLower(lastUserMsg)
	isImageRequest := strings.Contains(modelLower, "imagen") ||
		strings.Contains(modelLower, "image") ||
		strings.Contains(modelLower, "nano-banana") ||
		strings.Contains(modelLower, "banana") ||
		strings.HasPrefix(lowerPrompt, "buat gambar") ||
		strings.HasPrefix(lowerPrompt, "buatkan gambar") ||
		strings.HasPrefix(lowerPrompt, "buat foto") ||
		strings.HasPrefix(lowerPrompt, "buatkan foto") ||
		strings.HasPrefix(lowerPrompt, "buat ") ||
		strings.HasPrefix(lowerPrompt, "buatkan ") ||
		strings.HasPrefix(lowerPrompt, "foto ") ||
		strings.HasPrefix(lowerPrompt, "gambar ") ||
		strings.HasPrefix(lowerPrompt, "lukisan ") ||
		strings.HasPrefix(lowerPrompt, "lukis ") ||
		strings.HasPrefix(lowerPrompt, "photo of ") ||
		strings.HasPrefix(lowerPrompt, "picture of ") ||
		strings.HasPrefix(lowerPrompt, "image of ") ||
		strings.HasPrefix(lowerPrompt, "generate image") ||
		strings.HasPrefix(lowerPrompt, "generate an image") ||
		strings.HasPrefix(lowerPrompt, "create image") ||
		strings.HasPrefix(lowerPrompt, "create an image") ||
		strings.HasPrefix(lowerPrompt, "draw ") ||
		strings.Contains(lowerPrompt, "buat gambar ") ||
		strings.Contains(lowerPrompt, "buatkan gambar ") ||
		strings.Contains(lowerPrompt, "buat foto ") ||
		strings.Contains(lowerPrompt, "buatkan foto ")

	var modeID string
	if isImageRequest {
		// When generating images, modeID must be empty so Gemini invokes Imagen 3
		modeID = ""
		// CRITICAL: Send ONLY the actual image prompt without previous chat history transcript!
		prompt = lastUserMsg
		if !strings.HasPrefix(lowerPrompt, "buat") &&
			!strings.HasPrefix(lowerPrompt, "generate") &&
			!strings.HasPrefix(lowerPrompt, "create") &&
			!strings.HasPrefix(lowerPrompt, "draw") &&
			!strings.HasPrefix(lowerPrompt, "lukis") {
			prompt = "Buatkan gambar: " + prompt
		}
	} else {
		// Text chat mode: use specific model mode ID
		modeID = getModeID(req.Model)
		if strings.Contains(modelLower, "argon") {
			reasoning := ""
			if strings.Contains(modelLower, "high") || strings.Contains(modelLower, "tinggi") {
				reasoning = " - Tingkat Penalaran: Tinggi (Sangat Menyeluruh)"
			} else if strings.Contains(modelLower, "medium") || strings.Contains(modelLower, "sedang") {
				reasoning = " - Tingkat Penalaran: Sedang (Kedalaman Seimbang)"
			} else if strings.Contains(modelLower, "low") || strings.Contains(modelLower, "rendah") {
				reasoning = " - Tingkat Penalaran: Rendah (Cepat dan Efisien)"
			}
			prompt = fmt.Sprintf("[Active Model Mode: Gemini 4 Argon%s]\n", reasoning) + prompt
		} else if strings.Contains(modelLower, "3.1-pro") {
			prompt = "[Active Model Mode: Gemini 3.1 Pro]\n" + prompt
		} else if strings.Contains(modelLower, "3.8-flash") {
			prompt = "[Active Model Mode: Gemini 3.8 Flash]\n" + prompt
		} else if strings.Contains(modelLower, "3.5-flash-lite") {
			prompt = "[Active Model Mode: Gemini 3.5 Flash-Lite]\n" + prompt
		}
	}

	// Inner payload protobuf shape with modeID at index 3:
	// [ [ [prompt, 0, null, null, null, null, 0], null, ["", "", ""], mode_id ] ]
	var innerPayload []interface{}
	if modeID != "" {
		innerPayload = []interface{}{
			[]interface{}{prompt, 0, nil, nil, nil, nil, 0},
			nil,
			[]interface{}{"", "", ""},
			modeID,
		}
	} else {
		innerPayload = []interface{}{
			[]interface{}{prompt, 0, nil, nil, nil, nil, 0},
			nil,
			[]interface{}{"", "", ""},
		}
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

	log.Printf("[GEMINI SEND] Prompt: %q, at: %s, genURL: %s", prompt, at[:25], genURL)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", genURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	httpReq.Header.Set("Cookie", stripCustomTokens(norm))
	httpReq.Header.Set("Origin", "https://gemini.google.com")
	httpReq.Header.Set("Referer", "https://gemini.google.com/app")
	httpReq.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	httpReq.Header.Set("Accept", "*/*")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gemini StreamGenerate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Invalidate cache if 400, 401 or 403
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
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
	var activeFileID string
	var largestInnerStr string
	var bestImgBytes []byte

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

		if len(innerStr) > len(largestInnerStr) {
			largestInnerStr = innerStr
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
							// Strip internal Google reference markers (image_generation_content / lmdx_content)
							reInternalRef := regexp.MustCompile(`https?://googleusercontent\.com/(?:image_generation_content|lmdx_content)/[^\s]+`)
							tv = reInternalRef.ReplaceAllString(tv, "")
							textVal = strings.TrimSpace(tv)
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

		// 3. Extract any generated image URLs from Google User Content (Imagen 3 / gg-dl / gg)
		if strings.Contains(innerStr, "lh3.googleusercontent.com/gg-dl/") || strings.Contains(innerStr, "lh3.googleusercontent.com/gg/") {
			lh3Regex := regexp.MustCompile(`https://lh3\.googleusercontent\.com/(?:gg-dl|gg)/[a-zA-Z0-9\-_]+`)
			if matches := lh3Regex.FindAllString(innerStr, -1); len(matches) > 0 {
				if activeFileID == "" {
					activeFileID = fmt.Sprintf("img_%s", randomUUID()[:12])
				}

				// Pick best candidate: prefer gg/ over gg-dl/
				var pickedURL string
				for _, u := range matches {
					if strings.Contains(u, "/gg/") {
						pickedURL = u
						break
					}
				}
				if pickedURL == "" && len(matches) > 0 {
					pickedURL = matches[0]
				}

				fullResURL := pickedURL
				if downloadURL := p.resolveC8o8FeDownloadURL(ctx, norm, at, bl, innerStr, prompt); downloadURL != "" {
					log.Printf("[GEMINI] Successfully resolved master c8o8Fe download URL: %s", downloadURL)
					fullResURL = downloadURL
				} else if !strings.Contains(fullResURL, "=s") && !strings.Contains(fullResURL, "=w") {
					fullResURL += "=s0"
				}

				// Stream directly into Zero-Disk RAM buffer so the browser loads it without 403
				hdrs := map[string]string{
					"Cookie":  stripCustomTokens(norm),
					"Referer": "https://gemini.google.com/",
				}
				imgBytes, cType, err := media.FetchRemoteMediaStreamWithHeaders(ctx, fullResURL, hdrs)
				if err == nil && len(imgBytes) > len(bestImgBytes) {
					bestImgBytes = imgBytes
					media.GetManager().Store(&media.MediaItem{
						ID:            activeFileID, // REUSES activeFileID across all chunks!
						Type:          "image",
						ContentType:   cType,
						Data:          imgBytes,
						Prompt:        prompt,
						RevisedPrompt: textVal,
						Model:         req.Model,
						Provider:      "gemini",
						CreatedAt:     time.Now(),
					})
					log.Printf("[GEMINI] Stored/upgraded %s with %d bytes (cType=%s)", activeFileID, len(imgBytes), cType)
				}

				localURL := fmt.Sprintf("/v1/images/%s.png", activeFileID)
				if !strings.Contains(textVal, localURL) {
					if textVal != "" {
						textVal = textVal + fmt.Sprintf("\n\n![Generated Image](%s)\n\n", localURL)
					} else {
						textVal = fmt.Sprintf("![Generated Image](%s)\n\n", localURL)
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

	// Post-stream master upgrade: if we generated an image and its size is still low-res (< 500KB),
	// use the final complete largestInnerStr to fetch the master c8o8Fe asset
	if activeFileID != "" && len(bestImgBytes) < 500*1024 && largestInnerStr != "" {
		if masterURL := p.resolveC8o8FeDownloadURL(ctx, norm, at, bl, largestInnerStr, prompt); masterURL != "" {
			hdrs := map[string]string{
				"Cookie":  stripCustomTokens(norm),
				"Referer": "https://gemini.google.com/",
			}
			if masterBytes, cType, err := media.FetchRemoteMediaStreamWithHeaders(ctx, masterURL, hdrs); err == nil && len(masterBytes) > len(bestImgBytes) {
				log.Printf("[GEMINI] Post-stream upgraded %s to master c8o8Fe (%d bytes)", activeFileID, len(masterBytes))
				media.GetManager().Store(&media.MediaItem{
					ID:            activeFileID,
					Type:          "image",
					ContentType:   cType,
					Data:          masterBytes,
					Prompt:        prompt,
					RevisedPrompt: prompt,
					Model:         req.Model,
					Provider:      "gemini",
					CreatedAt:     time.Now(),
				})
			}
		}
	}

	return nil
}

// resolveC8o8FeDownloadURL invokes Google's batchexecute rpcids=c8o8Fe to retrieve
// the true master full-resolution original image URL (up to 2816x1536, ~700KB)
// matching the exact download mechanism of the official Gemini web app.
func (p *Provider) resolveC8o8FeDownloadURL(ctx context.Context, normCookie, atToken, blToken, innerStr, prompt string) string {
	reRef := regexp.MustCompile(`http://googleusercontent\.com/image_generation_content/\d+_\d+`)
	ref := reRef.FindString(innerStr)
	if ref == "" {
		return ""
	}

	reAes := regexp.MustCompile(`\$Aes[a-zA-Z0-9+/=]{100,}`)
	aesMatches := reAes.FindAllString(innerStr, -1)
	if len(aesMatches) < 2 {
		return ""
	}
	sig1 := aesMatches[0]
	sig2 := aesMatches[1]

	reTS := regexp.MustCompile(`thought_signature_\d+\.pb`)
	tsName := reTS.FindString(innerStr)
	if tsName == "" {
		return ""
	}

	reTSURL := regexp.MustCompile(`https://contribution\.usercontent\.google\.com/download[^\s"'\\<>]+`)
	tsURL := reTSURL.FindString(innerStr)
	tsURL = strings.ReplaceAll(tsURL, `\u003d`, "=")
	tsURL = strings.ReplaceAll(tsURL, `\u0026`, "&")

	reIM := regexp.MustCompile(`\bim_[0-9a-f]{8,}\b`)
	imID := reIM.FindString(innerStr)
	if imID == "" {
		return ""
	}

	reRC := regexp.MustCompile(`\brc_[0-9a-f]{8,}\b`)
	rcID := reRC.FindString(innerStr)

	reR := regexp.MustCompile(`\br_[0-9a-f]{8,}\b`)
	rID := reR.FindString(innerStr)

	reC := regexp.MustCompile(`\bc_[0-9a-f]{8,}\b`)
	cID := reC.FindString(innerStr)

	if cID == "" || rID == "" || rcID == "" {
		return ""
	}

	var sec int64 = time.Now().Unix()
	var nano int64 = 500000000
	reTSPair := regexp.MustCompile(`\[(17\d{8}),\s*(\d+)\]`)
	if mPair := reTSPair.FindStringSubmatch(innerStr); len(mPair) >= 3 {
		if s, err := strconv.ParseInt(mPair[1], 10, 64); err == nil {
			sec = s
		}
		if n, err := strconv.ParseInt(mPair[2], 10, 64); err == nil {
			nano = n
		}
	}

	salt := "3jedep3jedep3jed"

	aBlock := []interface{}{
		nil, nil, nil,
		[]interface{}{nil, nil, nil, nil, nil, sig1},
	}
	bBlock := []interface{}{ref, 0}
	thBlock := []interface{}{
		nil, 3, tsName, nil, nil, sig2, nil,
		[]interface{}{"", tsURL, ""},
		nil, []interface{}{sec, nano}, nil, "application/protobuf",
	}

	cBlock := make([]interface{}, 23)
	cBlock[0] = 23
	cBlock[1] = prompt
	cBlock[16] = 0
	cBlock[18] = "imagen_default.ultra.complaintflow"
	cBlock[22] = thBlock

	firstList := []interface{}{
		aBlock, bBlock, nil, cBlock,
		nil, nil, nil, nil,
		imID, salt,
	}
	dBlock := []interface{}{rID, rcID, cID, nil, salt}

	reqPayload := []interface{}{
		firstList,
		dBlock,
		1,
		0,
		1,
	}

	innerJSON, err := json.Marshal(reqPayload)
	if err != nil {
		return ""
	}

	freqJSON, err := json.Marshal([][][]interface{}{{{"c8o8Fe", string(innerJSON), nil, "generic"}}})
	if err != nil {
		return ""
	}

	cIDTrim := strings.TrimPrefix(cID, "c_")
	reqID := 1000000 + int(sec%9000000)
	targetURL := fmt.Sprintf("https://gemini.google.com/_/BardChatUi/data/batchexecute?rpcids=c8o8Fe&source-path=%%2Fapp%%2F%s&bl=%s&hl=id&_reqid=%d&rt=c",
		cIDTrim, url.QueryEscape(blToken), reqID)

	formData := url.Values{}
	formData.Set("f.req", string(freqJSON))
	formData.Set("at", atToken)

	reqRPC, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return ""
	}

	reqRPC.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	reqRPC.Header.Set("Cookie", stripCustomTokens(normCookie))
	reqRPC.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	reqRPC.Header.Set("Origin", "https://gemini.google.com")
	reqRPC.Header.Set("Referer", "https://gemini.google.com/app")
	reqRPC.Header.Set("X-Same-Domain", "1")
	reqRPC.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 15 * time.Second}
	respRPC, err := client.Do(reqRPC)
	if err != nil {
		return ""
	}
	defer respRPC.Body.Close()

	if respRPC.StatusCode != http.StatusOK {
		return ""
	}

	bodyBytes, err := io.ReadAll(respRPC.Body)
	if err != nil {
		return ""
	}
	bodyStr := string(bodyBytes)

	// Extract the returned URL from c8o8Fe response
	reDL := regexp.MustCompile(`https://lh3\.googleusercontent\.com/[^\s"'\\<>]+`)
	if dlMatch := reDL.FindString(bodyStr); dlMatch != "" {
		dlMatch = strings.ReplaceAll(dlMatch, `\u003d`, "=")
		dlMatch = strings.ReplaceAll(dlMatch, `\u0026`, "&")
		if !strings.Contains(dlMatch, "=s") && !strings.Contains(dlMatch, "=w") {
			dlMatch += "=s0"
		}
		return dlMatch
	}

	return ""
}

// GenerateVideo implements providers.VideoGenerator.
// It initiates video generation via Gemini's StreamGenerate, captures the conversation ID,
// polls Google's batchexecute rpcids=hNvQHb until the video is rendered, and streams
// the full master MP4 video directly into memory (Zero-Disk).
func (p *Provider) GenerateVideo(ctx context.Context, cookie, prompt, aspect string, duration int) ([]byte, string, error) {
	norm := normalizeCookie(cookie)
	if liveCookie := syncLiveGeminiCookie(); liveCookie != "" {
		norm = normalizeCookie(liveCookie)
	}
	if !strings.Contains(norm, "__Secure-1PSID") {
		return nil, "", errors.New("missing __Secure-1PSID in cookie")
	}

	at, bl, err := p.getTokens(ctx, norm)
	if err != nil {
		return nil, "", err
	}

	cleanPrompt := strings.TrimSpace(prompt)
	if cleanPrompt == "" {
		return nil, "", errors.New("empty prompt in video generation request")
	}

	// Format video prompt with aspect ratio instruction if applicable
	videoPrompt := cleanPrompt
	if !strings.HasPrefix(strings.ToLower(cleanPrompt), "video") && !strings.HasPrefix(strings.ToLower(cleanPrompt), "buat video") {
		videoPrompt = "video " + cleanPrompt
	}
	if aspect == "9:16" {
		videoPrompt += " (rasio aspek 9:16)"
	} else if aspect == "16:9" {
		videoPrompt += " (rasio aspek 16:9)"
	}

	modeID := "95a4c0abcba24364" // Gemini 4 Argon
	innerPayload := []interface{}{
		[]interface{}{videoPrompt, 0, nil, nil, nil, nil, 0},
		nil,
		[]interface{}{"", "", ""},
		modeID,
	}

	innerJSON, err := json.Marshal(innerPayload)
	if err != nil {
		return nil, "", fmt.Errorf("marshal inner payload: %w", err)
	}

	envelope := []interface{}{nil, string(innerJSON)}
	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, "", fmt.Errorf("marshal envelope: %w", err)
	}

	formData := url.Values{}
	formData.Set("f.req", string(envelopeJSON))
	formData.Set("at", at)

	reqID := 100000 + (time.Now().UnixNano() % 900000)
	genURL := fmt.Sprintf("https://gemini.google.com/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate?bl=%s&_reqid=%d&rt=c",
		url.QueryEscape(bl), reqID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", genURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, "", fmt.Errorf("create stream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	httpReq.Header.Set("Cookie", stripCustomTokens(norm))
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	httpReq.Header.Set("Origin", "https://gemini.google.com")
	httpReq.Header.Set("Referer", "https://gemini.google.com/app")
	httpReq.Header.Set("X-Same-Domain", "1")
	httpReq.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("gemini StreamGenerate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("gemini StreamGenerate HTTP %d: %s", resp.StatusCode, string(bodyPreview))
	}

	// Read StreamGenerate to extract conversation ID and initial status
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read stream response: %w", err)
	}
	streamBody := string(bodyBytes)

	// Extract conversation ID: c_[0-9a-f]{8,}
	reConv := regexp.MustCompile(`c_[0-9a-f]{8,}`)
	convMatches := reConv.FindAllString(streamBody, -1)
	if len(convMatches) == 0 {
		return nil, "", errors.New("no conversation ID found in Gemini video response")
	}
	convID := convMatches[0]
	log.Printf("[GEMINI VIDEO] Initiated video generation for conversation %s", convID)

	// Check if video URL is already present in StreamGenerate
	reVidURL := regexp.MustCompile(`https://contribution\.usercontent\.google\.com/download[^\s"'\\<>]+filename(?:\\u003d|=|\\\\u003d)video\.mp4[^\s"'\\<>]*`)
	if m := reVidURL.FindString(streamBody); m != "" {
		cleanURL := strings.ReplaceAll(m, `\\u003d`, "=")
		cleanURL = strings.ReplaceAll(cleanURL, `\u003d`, "=")
		cleanURL = strings.ReplaceAll(cleanURL, `\\u0026`, "&")
		cleanURL = strings.ReplaceAll(cleanURL, `\u0026`, "&")
		hdrs := map[string]string{
			"Cookie":  stripCustomTokens(norm),
			"Referer": "https://gemini.google.com/",
		}
		vidBytes, _, err := media.FetchRemoteMediaStreamWithHeaders(ctx, cleanURL, hdrs)
		if err == nil && len(vidBytes) > 1000 {
			return vidBytes, "Video Anda sudah siap!", nil
		}
	}

	// Poll Google's batchexecute rpcids=hNvQHb until video is completed (up to 35s for fast response)
	pollTicker := time.NewTicker(4 * time.Second)
	defer pollTicker.Stop()

	pollTimeout := 35 * time.Second
	pollCtx, pollCancel := context.WithTimeout(ctx, pollTimeout)
	defer pollCancel()

	pollPayload := []interface{}{convID, 10, nil, 1, []int{1}, []int{4}, nil, 1}
	pollInnerJSON, _ := json.Marshal(pollPayload)
	pollFreqJSON, _ := json.Marshal([][][]interface{}{{{"hNvQHb", string(pollInnerJSON), nil, "generic"}}})

	pollData := url.Values{}
	pollData.Set("f.req", string(pollFreqJSON))
	pollData.Set("at", at)

	pollURL := fmt.Sprintf("https://gemini.google.com/_/BardChatUi/data/batchexecute?rpcids=hNvQHb&source-path=%%2Fapp%%2F%s&bl=%s&hl=id&rt=c",
		strings.TrimPrefix(convID, "c_"), url.QueryEscape(bl))

	pollClient := &http.Client{Timeout: 15 * time.Second}

	for {
		select {
		case <-pollCtx.Done():
			return nil, "", fmt.Errorf("video generation polling timed out for conversation %s", convID)
		case <-pollTicker.C:
			pReq, pErr := http.NewRequestWithContext(pollCtx, "POST", pollURL, strings.NewReader(pollData.Encode()))
			if pErr != nil {
				continue
			}
			pReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
			pReq.Header.Set("Cookie", stripCustomTokens(norm))
			pReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
			pReq.Header.Set("Origin", "https://gemini.google.com")
			pReq.Header.Set("Referer", "https://gemini.google.com/app")
			pReq.Header.Set("X-Same-Domain", "1")
			pReq.Header.Set("Accept", "*/*")

			pResp, pErr := pollClient.Do(pReq)
			if pErr != nil {
				continue
			}
			pBytes, _ := io.ReadAll(pResp.Body)
			pResp.Body.Close()

			pStr := string(pBytes)
			if vMatch := reVidURL.FindString(pStr); vMatch != "" {
				cleanURL := strings.ReplaceAll(vMatch, `\\u003d`, "=")
				cleanURL = strings.ReplaceAll(cleanURL, `\u003d`, "=")
				cleanURL = strings.ReplaceAll(cleanURL, `\\u0026`, "&")
				cleanURL = strings.ReplaceAll(cleanURL, `\u0026`, "&")
				log.Printf("[GEMINI VIDEO] Found completed video URL: %s", cleanURL)

				hdrs := map[string]string{
					"Cookie":  stripCustomTokens(norm),
					"Referer": "https://gemini.google.com/",
				}
				vidBytes, _, err := media.FetchRemoteMediaStreamWithHeaders(ctx, cleanURL, hdrs)
				if err == nil && len(vidBytes) > 1000 {
					log.Printf("[GEMINI VIDEO] Downloaded master video: %d bytes (%.2f MB)", len(vidBytes), float64(len(vidBytes))/(1024*1024))
					return vidBytes, "Video Anda sudah siap!", nil
				}
			}
		}
	}
}
