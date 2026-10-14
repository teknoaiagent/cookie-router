package kimi

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
	"strings"
	"time"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
)

const (
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	KimiBaseURL = "https://www.kimi.ai"
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
			Timeout: 120 * time.Second,
		},
	}
}

func (p *Provider) ID() string   { return "kimi" }
func (p *Provider) Name() string { return "Kimi Moonshot Web" }

func (p *Provider) DefaultModels() []string {
	return []string{
		"kimi-k1.5",
		"kimi-k3",
		"kimi-search",
		"moonshot-v1-8k",
		"moonshot-v1-32k",
		"moonshot-v1-128k",
	}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func extractToken(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if strings.HasPrefix(cookie, "Bearer ") {
		return strings.TrimPrefix(cookie, "Bearer ")
	}
	if strings.Contains(cookie, "access_token=") {
		parts := strings.Split(cookie, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "access_token=") {
				return strings.TrimPrefix(part, "access_token=")
			}
		}
	}
	if strings.Contains(cookie, "kimi-auth=") {
		parts := strings.Split(cookie, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "kimi-auth=") {
				return strings.TrimPrefix(part, "kimi-auth=")
			}
		}
	}
	return cookie
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	token := extractToken(cookie)
	if token == "" {
		return errors.New("empty kimi access token or cookie")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", KimiBaseURL+"/api/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Kimi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("kimi authentication failed (HTTP %d)", resp.StatusCode)
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
		ID:      "chatcmpl-kimi-" + randomUUID(),
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

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	defer close(chunkChan)

	token := extractToken(cookie)
	if token == "" {
		return errors.New("empty kimi access token or cookie")
	}

	var promptBuilder strings.Builder
	for _, m := range req.Messages {
		if m.Role == "user" {
			promptBuilder.WriteString(m.TextContent())
			promptBuilder.WriteString("\n")
		}
	}
	prompt := strings.TrimSpace(promptBuilder.String())
	if prompt == "" {
		return errors.New("empty prompt messages")
	}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"model":  req.Model,
		"stream": true,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// Target Kimi completion endpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", KimiBaseURL+"/api/chat/completion", bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Cookie", utils.NormalizeCookie(cookie, "kimi"))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("kimi request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("kimi error HTTP %d: %s", resp.StatusCode, string(body))
	}

	chatUUID := "chatcmpl-kimi-" + randomUUID()
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var event struct {
			Text   string `json:"text"`
			Delta  string `json:"delta"`
			Action string `json:"action"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		content := event.Delta
		if content == "" {
			content = event.Text
		}

		if content != "" {
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
							Content: content,
						},
					},
				},
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading kimi stream: %w", err)
	}

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
