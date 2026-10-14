package qwen

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
	QwenBaseURL = "https://chat.qwenlm.ai"
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

func (p *Provider) ID() string   { return "qwen" }
func (p *Provider) Name() string { return "Qwen Web" }

func (p *Provider) DefaultModels() []string {
	return []string{
		"qwen-max",
		"qwen-plus",
		"qwen-turbo",
		"qwen-2.5-72b",
		"qwen-2.5-coder-32b",
		"qwq-32b-preview",
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
	normCookie := utils.NormalizeCookie(cookie, "qwen")
	if normCookie == "" {
		return errors.New("empty qwen token or cookie")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", QwenBaseURL+"/api/v1/auth/session", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	if strings.HasPrefix(normCookie, "Bearer ") || !strings.Contains(normCookie, "=") {
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(normCookie, "Bearer "))
	} else {
		req.Header.Set("Cookie", normCookie)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Qwen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("qwen authentication failed (HTTP %d)", resp.StatusCode)
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
		ID:      "chatcmpl-qwen-" + randomUUID(),
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

	normCookie := utils.NormalizeCookie(cookie, "qwen")
	if normCookie == "" {
		return errors.New("empty qwen token or cookie")
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
		"model":  req.Model,
		"prompt": prompt,
		"stream": true,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", QwenBaseURL+"/api/v1/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if strings.HasPrefix(normCookie, "Bearer ") || !strings.Contains(normCookie, "=") {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(normCookie, "Bearer "))
	} else {
		httpReq.Header.Set("Cookie", normCookie)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("qwen request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("qwen error HTTP %d: %s", resp.StatusCode, string(body))
	}

	chatUUID := "chatcmpl-qwen-" + randomUUID()
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

		var chunk models.ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		chunkChan <- &chunk
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading qwen stream: %w", err)
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
