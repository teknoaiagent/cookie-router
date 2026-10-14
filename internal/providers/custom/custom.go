package custom

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

func (p *Provider) ID() string   { return "custom" }
func (p *Provider) Name() string { return "Custom OpenAI Endpoint" }

func (p *Provider) DefaultModels() []string {
	return []string{"default"}
}

func parseEndpointAndKey(raw string) (baseURL string, apiKey string) {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "|") {
		parts := strings.SplitN(raw, "|", 2)
		baseURL = strings.TrimRight(strings.TrimSpace(parts[0]), "/")
		apiKey = strings.TrimSpace(parts[1])
		return
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		baseURL = strings.TrimRight(raw, "/")
		apiKey = ""
		return
	}
	// Default to OpenAI standard if only an API key is provided
	baseURL = "https://api.openai.com/v1"
	apiKey = raw
	return
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (p *Provider) Test(ctx context.Context, cookie string) error {
	baseURL, apiKey := parseEndpointAndKey(cookie)
	if baseURL == "" {
		return errors.New("empty custom base URL")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach custom endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("custom endpoint authorization failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("custom endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *Provider) GetAvailableModels(ctx context.Context, cookie string) ([]string, error) {
	baseURL, apiKey := parseEndpointAndKey(cookie)
	if baseURL == "" {
		return p.DefaultModels(), nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return p.DefaultModels(), err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return p.DefaultModels(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.DefaultModels(), nil
	}

	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return p.DefaultModels(), nil
	}

	var modelIDs []string
	for _, m := range data.Data {
		if m.ID != "" {
			modelIDs = append(modelIDs, m.ID)
		}
	}
	if len(modelIDs) == 0 {
		return p.DefaultModels(), nil
	}
	return modelIDs, nil
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatCompletionRequest, cookie string) (*models.ChatCompletionResponse, error) {
	baseURL, apiKey := parseEndpointAndKey(cookie)
	if baseURL == "" {
		return nil, errors.New("empty custom base URL")
	}

	// Disable streaming for direct Chat call
	reqCopy := *req
	reqCopy.Stream = false

	jsonBytes, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call custom endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("custom endpoint error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var chatResp models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("invalid json response from custom endpoint: %w", err)
	}

	return &chatResp, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatCompletionRequest, cookie string, chunkChan chan<- *models.ChatCompletionChunk) error {
	defer close(chunkChan)

	baseURL, apiKey := parseEndpointAndKey(cookie)
	if baseURL == "" {
		return errors.New("empty custom base URL")
	}

	reqCopy := *req
	reqCopy.Stream = true

	jsonBytes, err := json.Marshal(reqCopy)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call custom endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("custom endpoint upstream error HTTP %d: %s", resp.StatusCode, string(body))
	}

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
		return fmt.Errorf("error reading custom endpoint stream: %w", err)
	}

	return nil
}
