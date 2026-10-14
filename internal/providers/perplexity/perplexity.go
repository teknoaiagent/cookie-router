package perplexity

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
)

const (
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	PplxAskURL = "https://www.perplexity.ai/rest/sse/perplexity_ask"
)

type Provider struct {
	client *http.Client
}

func init() {
	providers.Register(New())
}

func New() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (p *Provider) ID() string   { return "perplexity" }
func (p *Provider) Name() string { return "Perplexity Web" }

func (p *Provider) DefaultModels() []string {
	return []string{
		"best",
		"claude-sonnet-5.5",
		"gpt-6.1-sol",
		"gemini-3.8-flash",
		"kimi-k3",
		"sonar",
		"sonar-reasoning",
		"sonar-deep-research",
		"pplx-pro",
		"gpt-4o",
		"claude-3-5-sonnet",
	}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	normCookie := utils.NormalizeCookie(cookie, "perplexity")
	if normCookie == "" {
		return errors.New("empty perplexity cookie")
	}

	reqBody := map[string]interface{}{
		"version":          "2.18",
		"source":           "default",
		"language":         "en-US",
		"timezone":         "UTC",
		"search_focus":     "internet",
		"mode":             "concise",
		"model_preference": "default",
		"query_str":        "ping",
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", PplxAskURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Origin", "https://www.perplexity.ai")
	req.Header.Set("Referer", "https://www.perplexity.ai/")
	if normCookie != "" {
		req.Header.Set("Cookie", normCookie)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Perplexity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("perplexity authentication failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("perplexity returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	return p.DefaultModels(), nil
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatCompletionRequest, cookie string) (*models.ChatCompletionResponse, error) {
	chunkChan := make(chan *models.ChatCompletionChunk, 50)
	errChan := make(chan error, 1)

	go func() {
		err := p.ChatStream(ctx, req, cookie, chunkChan)
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	var fullContent strings.Builder
	var lastChunk *models.ChatCompletionChunk

	for chunk := range chunkChan {
		lastChunk = chunk
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	if err := <-errChan; err != nil {
		return nil, err
	}

	finishReason := "stop"
	resp := &models.ChatCompletionResponse{
		ID:      "chatcmpl-pplx-" + randomUUID(),
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
			PromptTokens:     10,
			CompletionTokens: len(fullContent.String()) / 4,
			TotalTokens:      10 + (len(fullContent.String()) / 4),
		},
	}

	if lastChunk != nil && lastChunk.ID != "" {
		resp.ID = lastChunk.ID
	}

	return resp, nil
}

type pplxBlock struct {
	IntendedUsage string `json:"intended_usage"`
	MarkdownBlock *struct {
		Chunks               []string `json:"chunks"`
		ChunkStartingOffset int      `json:"chunk_starting_offset"`
		Answer               string   `json:"answer"`
		Progress             string   `json:"progress"`
	} `json:"markdown_block"`
}

type pplxEvent struct {
	BackendUUID  string      `json:"backend_uuid"`
	Status       string      `json:"status"`
	Final        bool        `json:"final"`
	Blocks       []pplxBlock `json:"blocks"`
	Text         string      `json:"text"`
	ErrorMessage string      `json:"error_message"`
	ErrorCode    string      `json:"error_code"`
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	defer close(chunkChan)

	normCookie := utils.NormalizeCookie(cookie, "perplexity")
	if normCookie == "" {
		return errors.New("empty perplexity cookie")
	}

	// Determine model preference and mode
	modelPref, isCopilot := mapModelToPreference(req.Model)
	mode := "concise"
	if isCopilot {
		mode = "copilot"
	}

	// Build query prompt
	var promptBuilder strings.Builder
	for _, m := range req.Messages {
		text := m.TextContent()
		if m.Role == "system" {
			promptBuilder.WriteString("[System Instructions]: ")
			promptBuilder.WriteString(text)
			promptBuilder.WriteString("\n\n")
		} else if m.Role == "user" {
			promptBuilder.WriteString(text)
			promptBuilder.WriteString("\n")
		} else if m.Role == "assistant" {
			promptBuilder.WriteString("\n[Previous Assistant Response]: ")
			promptBuilder.WriteString(text)
			promptBuilder.WriteString("\n")
		}
	}
	queryStr := strings.TrimSpace(promptBuilder.String())
	if queryStr == "" {
		return errors.New("empty prompt messages")
	}

	reqBody := map[string]interface{}{
		"version":          "2.18",
		"source":           "default",
		"attachments":      []interface{}{},
		"language":         "en-US",
		"timezone":         "UTC",
		"search_focus":     "internet",
		"mode":             mode,
		"model_preference": modelPref,
		"query_str":        queryStr,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", PplxAskURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Origin", "https://www.perplexity.ai")
	httpReq.Header.Set("Referer", "https://www.perplexity.ai/")
	if normCookie != "" {
		httpReq.Header.Set("Cookie", normCookie)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call Perplexity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("perplexity upstream error HTTP %d: %s", resp.StatusCode, string(body))
	}

	chatUUID := "chatcmpl-pplx-" + randomUUID()
	chunkMap := make(map[int]string)
	sentLen := 0

	reconstructText := func() string {
		keys := make([]int, 0, len(chunkMap))
		for k := range chunkMap {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(chunkMap[k])
		}
		return sb.String()
	}

	scanner := bufio.NewScanner(resp.Body)
	// Allocate 1MB max line buffer
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: end_of_stream") {
			break
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "" || data == "{}" || data == "[DONE]" {
			continue
		}

		var ev pplxEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}

		if ev.ErrorCode != "" || ev.ErrorMessage != "" {
			return fmt.Errorf("perplexity error: %s (code: %s)", ev.ErrorMessage, ev.ErrorCode)
		}

		if ev.BackendUUID != "" {
			chatUUID = "chatcmpl-" + ev.BackendUUID
		}

		// Update chunks from ask_text block
		for _, b := range ev.Blocks {
			if b.MarkdownBlock != nil {
				mb := b.MarkdownBlock
				for idx, chk := range mb.Chunks {
					chunkMap[mb.ChunkStartingOffset+idx] = chk
				}
				if mb.Answer != "" && mb.Progress == "DONE" {
					// Final answer override if provided
					current := reconstructText()
					if len(mb.Answer) > len(current) {
						chunkMap[0] = mb.Answer
						for k := range chunkMap {
							if k != 0 {
								delete(chunkMap, k)
							}
						}
					}
				}
			}
		}

		currentText := reconstructText()
		if len(currentText) > sentLen {
			delta := currentText[sentLen:]
			sentLen = len(currentText)

			chunkChan <- &models.ChatCompletionChunk{
				ID:      chatUUID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []models.ChunkChoice{
					{
						Index: 0,
						Delta: models.ChunkDelta{
							Role:    "assistant",
							Content: delta,
						},
					},
				},
			}
		}

		if ev.Status == "COMPLETED" && ev.Final {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading Perplexity stream: %w", err)
	}

	// Send terminal chunk
	termReason := "stop"
	chunkChan <- &models.ChatCompletionChunk{
		ID:      chatUUID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChunkChoice{
			{
				Index:        0,
				Delta:        models.ChunkDelta{},
				FinishReason: &termReason,
			},
		},
	}

	return nil
}

func mapModelToPreference(model string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(model))
	isCopilot := strings.Contains(lower, "reason") || strings.Contains(lower, "deep") || strings.Contains(lower, "k3") || strings.Contains(lower, "pro")

	switch {
	case lower == "best" || lower == "auto" || lower == "default":
		return "default", false
	case strings.Contains(lower, "sonar-reasoning"):
		return "sonar_reasoning", true
	case strings.Contains(lower, "deep-research") || strings.Contains(lower, "sonar-deep"):
		return "sonar_deep_research", true
	case strings.Contains(lower, "pplx-pro") || strings.Contains(lower, "pro"):
		return "pplx_pro", true
	case strings.Contains(lower, "gpt-6") || strings.Contains(lower, "sol"):
		return "gpt_6_1_sol", false
	case strings.Contains(lower, "gemini-3") || strings.Contains(lower, "3.8"):
		return "gemini_3_8_flash", false
	case strings.Contains(lower, "sonnet-5") || strings.Contains(lower, "5.5"):
		return "claude_sonnet_5_5", false
	case strings.Contains(lower, "kimi") || strings.Contains(lower, "k3"):
		return "kimi_k3", true
	case strings.Contains(lower, "gpt-4o"):
		return "gpt_4o", false
	case strings.Contains(lower, "claude-3-5-sonnet") || strings.Contains(lower, "claude"):
		return "claude_3_5_sonnet", false
	case strings.Contains(lower, "sonar"):
		return "sonar", false
	default:
		// Convert e.g. claude-sonnet-5.5 -> claude_sonnet_5_5
		cleaned := strings.ReplaceAll(strings.ReplaceAll(lower, "-", "_"), ".", "_")
		return cleaned, isCopilot
	}
}
