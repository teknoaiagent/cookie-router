package chatgpt

import (
	"context"
	"cookie-router/internal/models"
	"encoding/json"
	"os"
	"testing"
)

type rootConfig struct {
	Accounts []struct {
		Cookie string `json:"cookie"`
	} `json:"accounts"`
}

func TestSingleSlug(t *testing.T) {
	data, err := os.ReadFile("../../../config.json")
	if err != nil {
		t.Fatalf("read config err: %v", err)
	}
	var cfg rootConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	p := New()
	req := &models.ChatCompletionRequest{
		Model: "auto",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "halo"},
		},
		Stream: false,
	}
	ch := make(chan *models.ChatCompletionChunk, 100)
	err = p.ChatStream(context.Background(), req, cfg.Accounts[0].Cookie, ch)
	t.Logf("RESULT: %v", err)
}
