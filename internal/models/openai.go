package models

import (
	"encoding/json"
	"fmt"
)

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // can be string or []interface{}
}

func (m *ChatMessage) TextContent() string {
	if m.Content == nil {
		return ""
	}
	switch v := m.Content.(type) {
	case string:
		return v
	case []interface{}:
		var fullText string
		for _, part := range v {
			if pMap, ok := part.(map[string]interface{}); ok {
				if pMap["type"] == "text" {
					if t, ok := pMap["text"].(string); ok {
						fullText += t
					}
				}
			}
		}
		return fullText
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (m *ChatMessage) ImageURLs() []string {
	if m.Content == nil {
		return nil
	}
	var urls []string
	if parts, ok := m.Content.([]interface{}); ok {
		for _, part := range parts {
			if pMap, ok := part.(map[string]interface{}); ok {
				if pMap["type"] == "image_url" {
					if imgObj, ok := pMap["image_url"].(map[string]interface{}); ok {
						if u, ok := imgObj["url"].(string); ok && u != "" {
							urls = append(urls, u)
						}
					}
				}
			}
		}
	}
	return urls
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason *string     `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage    `json:"usage"`
}

type ChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

func (c *ChatCompletionChunk) ToSSE() string {
	b, _ := json.Marshal(c)
	return fmt.Sprintf("data: %s\n\n", string(b))
}

type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelList struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}

type ImageGenerationRequest struct {
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model,omitempty"`
	AccountID      string   `json:"account_id,omitempty"`
	Image          string   `json:"image,omitempty"` // Base64 or URL for img2img
	Attachments    []string `json:"attachments,omitempty"`
	N              int      `json:"n,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"` // "url" or "b64_json"
	Size           string   `json:"size,omitempty"`
	Style          string   `json:"style,omitempty"`
	User           string   `json:"user,omitempty"`
}

type ImageGenerationData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64                 `json:"created"`
	Data    []ImageGenerationData `json:"data"`
}

type VideoGenerationRequest struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	AspectRatio    string `json:"aspect_ratio,omitempty"` // "16:9", "9:16", "1:1", "4:3"
	Duration       int    `json:"duration,omitempty"`     // 5, 10
	Quality        string `json:"quality,omitempty"`      // "standard", "hd"
	ResponseFormat string `json:"response_format,omitempty"` // "url" or "b64_json"
	User           string `json:"user,omitempty"`
}

type VideoGenerationData struct {
	URL           string `json:"url,omitempty"`
	Format        string `json:"format,omitempty"` // "mp4"
	MimeType      string `json:"mime_type,omitempty"`
	Status        string `json:"status,omitempty"` // "completed", "processing"
	Duration      int    `json:"duration,omitempty"`
	AspectRatio   string `json:"aspect_ratio,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
}

type VideoGenerationResponse struct {
	ID      string                `json:"id,omitempty"`
	Object  string                `json:"object,omitempty"`
	Created int64                 `json:"created"`
	Model   string                `json:"model,omitempty"`
	Status  string                `json:"status,omitempty"`
	Data    []VideoGenerationData `json:"data"`
}
