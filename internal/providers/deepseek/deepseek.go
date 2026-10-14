package deepseek

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
)

const (
	UserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	DeepSeekBaseURL = "https://chat.deepseek.com"
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

func (p *Provider) ID() string   { return "deepseek" }
func (p *Provider) Name() string { return "DeepSeek Web" }
func (p *Provider) DefaultModels() []string {
	return []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v3", "deepseek-r1"}
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func cleanToken(raw string) string {
	t := strings.TrimSpace(raw)
	t = strings.TrimPrefix(t, "Bearer ")
	t = strings.TrimPrefix(t, "userToken=")
	return t
}

func (p *Provider) Test(ctx context.Context, token string) error {
	tok := cleanToken(token)
	if tok == "" {
		return errors.New("userToken is required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", DeepSeekBaseURL+"/api/v0/users/current", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("unauthorized (userToken invalid or expired)")
	}
	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, token string) ([]string, error) {
	return []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v3", "deepseek-r1"}, nil
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

	respID := "chatcmpl-deepseek-" + randomUUID()[:8]
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
	tok := cleanToken(cookie)
	if tok == "" {
		return errors.New("empty DeepSeek userToken")
	}

	var promptBuilder strings.Builder
	for _, m := range req.Messages {
		text := m.TextContent()
		promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(m.Role), text))
	}
	fullPrompt := promptBuilder.String()

	thinkingEnabled := strings.Contains(strings.ToLower(req.Model), "reasoner") || strings.Contains(strings.ToLower(req.Model), "r1")

	// 1. Create chat session
	sessionURL := DeepSeekBaseURL + "/api/v0/chat_session/create"
	sessionBody := map[string]interface{}{"agent": "chat"}
	sessBytes, _ := json.Marshal(sessionBody)

	sReq, err := http.NewRequestWithContext(ctx, "POST", sessionURL, bytes.NewReader(sessBytes))
	if err != nil {
		return err
	}
	sReq.Header.Set("User-Agent", UserAgent)
	sReq.Header.Set("Content-Type", "application/json")
	sReq.Header.Set("Authorization", "Bearer "+tok)

	sResp, err := p.client.Do(sReq)
	if err != nil {
		return fmt.Errorf("failed to create DeepSeek chat session: %w", err)
	}
	defer sResp.Body.Close()

	var sessResult struct {
		Data struct {
			BizData struct {
				ID string `json:"id"`
			} `json:"biz_data"`
		} `json:"data"`
	}
	_ = json.NewDecoder(sResp.Body).Decode(&sessResult)
	sessionID := sessResult.Data.BizData.ID
	if sessionID == "" {
		sessionID = randomUUID()
	}

	// 2. Send completion request
	compURL := DeepSeekBaseURL + "/api/v0/chat/completion"
	compPayload := map[string]interface{}{
		"chat_session_id":   sessionID,
		"parent_message_id": nil,
		"prompt":            fullPrompt,
		"ref_file_ids":      []interface{}{},
		"thinking_enabled":  thinkingEnabled,
	}

	compBytes, _ := json.Marshal(compPayload)
	compReq, err := http.NewRequestWithContext(ctx, "POST", compURL, bytes.NewReader(compBytes))
	if err != nil {
		return err
	}

	compReq.Header.Set("User-Agent", UserAgent)
	compReq.Header.Set("Content-Type", "application/json")
	compReq.Header.Set("Accept", "text/event-stream")
	compReq.Header.Set("Authorization", "Bearer "+tok)

	resp, err := p.client.Do(compReq)
	if err != nil {
		return fmt.Errorf("DeepSeek completion error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DeepSeek error (status %d): %s", resp.StatusCode, string(b))
	}

	reader := bufio.NewReader(resp.Body)
	scanner := bufio.NewScanner(reader)
	chunkID := "chatcmpl-deepseek-" + randomUUID()[:8]

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &event); err == nil {
				if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
					chunkChan <- &models.ChatCompletionChunk{
						ID:      chunkID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []models.ChunkChoice{
							{
								Index: 0,
								Delta: models.ChunkDelta{
									Content: event.Choices[0].Delta.Content,
								},
							},
						},
					}
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
