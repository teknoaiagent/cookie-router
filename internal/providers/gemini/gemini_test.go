package gemini

import (
	"context"
	"testing"
)

func TestGeminiDefaultModels(t *testing.T) {
	p := New()
	models := p.DefaultModels()
	if len(models) < 10 {
		t.Fatalf("expected at least 10 default models, got %d", len(models))
	}

	foundPro := false
	foundFlash := false
	foundThinking := false
	foundImagen := false

	for _, m := range models {
		if m == "gemini-2.5-pro" {
			foundPro = true
		}
		if m == "gemini-2.5-flash" {
			foundFlash = true
		}
		if m == "gemini-2.0-flash-thinking" {
			foundThinking = true
		}
		if m == "imagen-3" {
			foundImagen = true
		}
	}

	if !foundPro || !foundFlash || !foundThinking || !foundImagen {
		t.Errorf("missing key Gemini models: pro=%v, flash=%v, thinking=%v, imagen=%v",
			foundPro, foundFlash, foundThinking, foundImagen)
	}
}

func TestGeminiTestInvalidCookie(t *testing.T) {
	p := New()
	err := p.Test(context.Background(), "session_id=bogus; theme=dark")
	if err == nil {
		t.Fatal("expected error on invalid cookie lacking __Secure-1PSID")
	}
}
