package grok

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
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	GrokChatAPI = "https://grok.com/rest/app-chat/conversations/new"
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

func (p *Provider) ID() string   { return "grok" }
func (p *Provider) Name() string { return "Grok Web (xAI)" }
func (p *Provider) DefaultModels() []string {
	return []string{
		"grok-3",
		"grok-3-mini",
		"grok-3-reasoning",
		"grok-3-deepsearch",
		"grok-2",
		"grok-2-vision",
		"grok-beta",
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
	if !strings.Contains(cookie, "sso") {
		return errors.New("cookie must contain sso (or sso-rw)")
	}
	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	return []string{"grok-3", "grok-2", "grok-2-vision", "grok-beta"}, nil
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

	respID := "chatcmpl-grok-" + randomUUID()[:8]
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
	var promptBuilder strings.Builder
	for _, m := range req.Messages {
		text := m.TextContent()
		promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(m.Role), text))
	}
	fullPrompt := promptBuilder.String()

	modelName := "grok-2"
	if req.Model != "" && req.Model != "auto" {
		modelName = req.Model
	}

	payload := map[string]interface{}{
		"temporary":                 true,
		"modelName":                 modelName,
		"message":                   fullPrompt,
		"fileAttachments":           []interface{}{},
		"imageAttachments":          []interface{}{},
		"disableSearch":             false,
		"enableImageGeneration":     false,
		"returnImageBytes":          false,
		"returnRawGrokInXaiRequest": false,
		"enableImageStreaming":      false,
		"imageGenerationCount":      1,
		"forceConcise":              false,
		"toolOverrides":             map[string]interface{}{},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", GrokChatAPI, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Origin", "https://grok.com")
	httpReq.Header.Set("Referer", "https://grok.com/")
	httpReq.Header.Set("Cookie", cookie)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Grok API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Grok API error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	scanner := bufio.NewScanner(reader)

	chunkID := "chatcmpl-grok-" + randomUUID()[:8]

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		var event struct {
			Result struct {
				Response struct {
					Token         string `json:"token"`
					IsSoftStop    bool   `json:"isSoftStop"`
					ModelResponse struct {
						Message string `json:"message"`
					} `json:"modelResponse"`
				} `json:"response"`
			} `json:"result"`
			Error string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &event); err == nil {
			if event.Error != "" {
				return errors.New(event.Error)
			}

			token := event.Result.Response.Token
			if token != "" {
				chunkChan <- &models.ChatCompletionChunk{
					ID:      chunkID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   modelName,
					Choices: []models.ChunkChoice{
						{
							Index: 0,
							Delta: models.ChunkDelta{
								Content: token,
							},
						},
					},
				}
			}

			if event.Result.Response.IsSoftStop {
				break
			}
		}
	}

	stop := "stop"
	chunkChan <- &models.ChatCompletionChunk{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []models.ChunkChoice{
			{
				Index:        0,
				FinishReason: &stop,
			},
		},
	}

	return scanner.Err()
}
