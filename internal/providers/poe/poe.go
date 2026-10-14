package poe

import (
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
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	PoeBase   = "https://www.poe.com"
	PoeGQL    = "https://www.poe.com/api/gql_POST"
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

func (p *Provider) ID() string   { return "poe" }
func (p *Provider) Name() string { return "Poe Web" }

func (p *Provider) DefaultModels() []string {
	return []string{
		"Claude-3.7-Sonnet",
		"Claude-3.7-Sonnet-Reasoning",
		"Claude-3.5-Sonnet",
		"GPT-4o",
		"o3-mini",
		"o1",
		"Gemini-2.0-Flash",
		"Gemini-2.5-Pro",
		"DeepSeek-R1",
		"DeepSeek-V3",
		"Assistant",
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
	normCookie := utils.NormalizeCookie(cookie, "poe")
	if normCookie == "" {
		return errors.New("empty poe cookie (p-b required)")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", PoeBase, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Cookie", normCookie)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Poe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("poe authentication failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	return p.DefaultModels(), nil
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatCompletionRequest, cookie string) (*models.ChatCompletionResponse, error) {
	normCookie := utils.NormalizeCookie(cookie, "poe")
	if normCookie == "" {
		return nil, errors.New("empty poe cookie")
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
		return nil, errors.New("empty prompt messages")
	}

	botName := req.Model
	if botName == "" || strings.EqualFold(botName, "default") {
		botName = "Assistant"
	}

	gqlBody := map[string]interface{}{
		"operationName": "ChatViewQuery",
		"query": `query ChatViewQuery($bot: String!, $query: String!) {
			chatWithBot(bot: $bot, query: $query) {
				messageId
				text
				state
			}
		}`,
		"variables": map[string]interface{}{
			"bot":   botName,
			"query": prompt,
		},
	}

	jsonBytes, err := json.Marshal(gqlBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", PoeGQL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Origin", PoeBase)
	httpReq.Header.Set("Referer", PoeBase+"/")
	httpReq.Header.Set("Cookie", normCookie)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("poe request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("poe error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp struct {
		Data struct {
			ChatWithBot struct {
				MessageID string `json:"messageId"`
				Text      string `json:"text"`
				State     string `json:"state"`
			} `json:"chatWithBot"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("invalid json response from Poe: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("poe gql error: %s", gqlResp.Errors[0].Message)
	}

	answer := gqlResp.Data.ChatWithBot.Text
	if answer == "" && gqlResp.Data.ChatWithBot.MessageID == "" {
		answer = "Received response from Poe"
	}

	stopReason := "stop"
	return &models.ChatCompletionResponse{
		ID:      "chatcmpl-poe-" + randomUUID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: answer,
				},
				FinishReason: &stopReason,
			},
		},
		Usage: models.ChatCompletionUsage{
			PromptTokens:     len(prompt) / 4,
			CompletionTokens: len(answer) / 4,
			TotalTokens:      (len(prompt) + len(answer)) / 4,
		},
	}, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	defer close(chunkChan)

	resp, err := p.Chat(ctx, req, cookie)
	if err != nil {
		return err
	}

	var content string
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.TextContent()
	}

	// Split content into chunks to simulate stream
	words := strings.Fields(content)
	chunkSize := 3
	for i := 0; i < len(words); i += chunkSize {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		part := strings.Join(words[i:end], " ") + " "
		chunkChan <- &models.ChatCompletionChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []models.ChunkChoice{
				{
					Index: 0,
					Delta: models.ChunkDelta{
						Role:    "assistant",
						Content: part,
					},
				},
			},
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Terminal chunk
	termReason := "stop"
	chunkChan <- &models.ChatCompletionChunk{
		ID:      resp.ID,
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
