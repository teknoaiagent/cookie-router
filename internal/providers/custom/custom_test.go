package custom

import (
	"context"
	"testing"

	"cookie-router/internal/providers"
)

func TestCustomRegistration(t *testing.T) {
	p := providers.Get("custom")
	if p == nil {
		t.Fatal("custom provider not registered")
	}

	if p.ID() != "custom" {
		t.Fatalf("expected ID 'custom', got %s", p.ID())
	}
}

func TestParseCredentials(t *testing.T) {
	cases := []struct {
		input       string
		expectedURL string
		expectedKey string
	}{
		{"https://api.groq.com/openai/v1|gsk_test123", "https://api.groq.com/openai/v1", "gsk_test123"},
		{"http://localhost:11434/v1", "http://localhost:11434/v1", ""},
		{"sk-testkey", "https://api.openai.com/v1", "sk-testkey"},
	}

	for _, tc := range cases {
		u, k := parseEndpointAndKey(tc.input)
		if u != tc.expectedURL {
			t.Errorf("for %s: expected URL %s, got %s", tc.input, tc.expectedURL, u)
		}
		if k != tc.expectedKey {
			t.Errorf("for %s: expected Key %s, got %s", tc.input, tc.expectedKey, k)
		}
	}
}

func TestCustomEmpty(t *testing.T) {
	p := providers.Get("custom")
	err := p.Test(context.Background(), "")
	if err == nil {
		t.Fatal("expected error on empty credential, got nil")
	}
}
