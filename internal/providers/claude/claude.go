package claude

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
	"sync"
	"time"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
	"cookie-router/internal/utils"
)

const (
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	ClaudeApiBase = "https://claude.ai/api"
)

type Provider struct {
	client   *http.Client
	orgCache sync.Map // sessionKey -> orgID
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

func (p *Provider) ID() string   { return "claude" }
func (p *Provider) Name() string { return "Claude Web" }
func (p *Provider) DefaultModels() []string {
	return []string{
		"claude-3-7-sonnet",
		"claude-3-7-sonnet-thinking",
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
		"claude-3-opus",
		"claude-3-haiku",
	}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func normalizeSessionKey(raw string) string {
	return utils.NormalizeCookie(raw, "claude")
}

func (p *Provider) getOrgID(ctx context.Context, rawCookie string) (string, error) {
	cookieHeader := normalizeSessionKey(rawCookie)
	if val, ok := p.orgCache.Load(cookieHeader); ok {
		return val.(string), nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ClaudeApiBase+"/organizations", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", cookieHeader)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Claude organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("organizations API returned status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return "", fmt.Errorf("invalid json in organizations response: %w", err)
	}

	if len(orgs) == 0 || orgs[0].UUID == "" {
		return "", errors.New("no organization found for this Claude session")
	}

	orgID := orgs[0].UUID
	p.orgCache.Store(cookieHeader, orgID)
	return orgID, nil
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	orgID, err := p.getOrgID(ctx, cookie)
	if err != nil {
		return err
	}
	if orgID == "" {
		return errors.New("empty organization ID")
	}
	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	_, err := p.getOrgID(ctx, cookie)
	if err != nil {
		return p.DefaultModels(), err
	}
	return []string{
		"claude-3-7-sonnet",
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
		"claude-3-opus",
	}, nil
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

	respID := "chatcmpl-claude-" + randomUUID()[:8]
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
	orgID, err := p.getOrgID(ctx, cookie)
	if err != nil {
		return err
	}

	cookieHeader := normalizeSessionKey(cookie)
	chatUUID := randomUUID()

	// 1. Create temporary conversation
	convURL := fmt.Sprintf("%s/organizations/%s/chat_conversations", ClaudeApiBase, orgID)
	convBody := map[string]string{
		"uuid": chatUUID,
		"name": "",
	}
	convBytes, _ := json.Marshal(convBody)
	createReq, err := http.NewRequestWithContext(ctx, "POST", convURL, bytes.NewReader(convBytes))
	if err != nil {
		return err
	}
	createReq.Header.Set("User-Agent", UserAgent)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Cookie", cookieHeader)

	cResp, err := p.client.Do(createReq)
	if err != nil {
		return fmt.Errorf("failed to create Claude conversation: %w", err)
	}
	cResp.Body.Close()

	// 2. Build prompt
	var promptBuilder strings.Builder
	for _, m := range req.Messages {
		text := m.TextContent()
		if m.Role == "system" {
			promptBuilder.WriteString(fmt.Sprintf("\n\nSystem: %s", text))
		} else if m.Role == "user" {
			promptBuilder.WriteString(fmt.Sprintf("\n\nHuman: %s", text))
		} else if m.Role == "assistant" {
			promptBuilder.WriteString(fmt.Sprintf("\n\nAssistant: %s", text))
		}
	}
	promptBuilder.WriteString("\n\nAssistant:")
	fullPrompt := promptBuilder.String()

	// 3. Send completion stream
	streamURL := fmt.Sprintf("%s/organizations/%s/chat_conversations/%s/completion_stream", ClaudeApiBase, orgID, chatUUID)
	payload := map[string]interface{}{
		"prompt":      fullPrompt,
		"timezone":    "Asia/Jakarta",
		"model":       req.Model,
		"attachments": []interface{}{},
	}
	payloadBytes, _ := json.Marshal(payload)

	streamReq, err := http.NewRequestWithContext(ctx, "POST", streamURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	streamReq.Header.Set("User-Agent", UserAgent)
	streamReq.Header.Set("Content-Type", "application/json")
	streamReq.Header.Set("Accept", "text/event-stream")
	streamReq.Header.Set("Cookie", cookieHeader)

	resp, err := p.client.Do(streamReq)
	if err != nil {
		return fmt.Errorf("failed to stream from Claude: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Claude completion stream error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	scanner := bufio.NewScanner(reader)

	chunkID := "chatcmpl-claude-" + randomUUID()[:8]

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event struct {
				Completion string `json:"completion"`
				StopReason string `json:"stop_reason"`
				Error      string `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				if event.Completion != "" {
					chunkChan <- &models.ChatCompletionChunk{
						ID:      chunkID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []models.ChunkChoice{
							{
								Index: 0,
								Delta: models.ChunkDelta{
									Content: event.Completion,
								},
							},
						},
					}
				}
				if event.StopReason != "" {
					break
				}
			}
		}
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

	return scanner.Err()
}
