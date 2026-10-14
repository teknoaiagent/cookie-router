package perplexity

import (
	"context"
	"testing"

	"cookie-router/internal/models"
	"cookie-router/internal/providers"
)

func TestPerplexityRegistration(t *testing.T) {
	p := providers.Get("perplexity")
	if p == nil {
		t.Fatal("perplexity provider not registered")
	}

	if p.ID() != "perplexity" {
		t.Fatalf("expected ID 'perplexity', got %s", p.ID())
	}

	modelsList := p.DefaultModels()
	if len(modelsList) == 0 {
		t.Fatal("expected default models, got empty")
	}
}

func TestPerplexityModelPreferenceMapping(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		isCopilot bool
	}{
		{"best", "default", false},
		{"claude-sonnet-5.5", "claude_sonnet_5_5", false},
		{"gpt-6.1-sol", "gpt_6_1_sol", false},
		{"gemini-3.8-flash", "gemini_3_8_flash", false},
		{"kimi-k3", "kimi_k3", true},
		{"sonar", "sonar", false},
		{"sonar-reasoning", "sonar_reasoning", true},
		{"sonar-deep-research", "sonar_deep_research", true},
		{"pplx-pro", "pplx_pro", true},
		{"gpt-4o", "gpt_4o", false},
		{"claude-3-5-sonnet", "claude_3_5_sonnet", false},
	}

	for _, tc := range cases {
		pref, isCo := mapModelToPreference(tc.input)
		if pref != tc.expected {
			t.Errorf("for %s: expected pref %s, got %s", tc.input, tc.expected, pref)
		}
		if isCo != tc.isCopilot {
			t.Errorf("for %s: expected copilot %v, got %v", tc.input, tc.isCopilot, isCo)
		}
	}
}

func TestPerplexityEmptyCookie(t *testing.T) {
	p := providers.Get("perplexity")
	err := p.Test(context.Background(), "")
	if err == nil {
		t.Fatal("expected error on empty cookie, got nil")
	}

	req := &models.ChatCompletionRequest{
		Model: "sonar",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}
	_, err = p.Chat(context.Background(), req, "")
	if err == nil {
		t.Fatal("expected error on empty cookie in Chat, got nil")
	}
}
