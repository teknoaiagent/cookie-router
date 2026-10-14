package utils

import (
	"strings"
	"testing"
)

func TestNormalizeCookieTSVWithChunkedToken(t *testing.T) {
	input := `__cf_bm	NBjfWi9ocHz9qE5GTfRB5QM6plHasVIGioUiC37TMfE-1791981552	.chatgpt.com	/	2026-10-14T13:09:12.807Z	206			None			Medium
__cflb	0H28vzvP5FJafnkHxihKZRMPTs8EfEnopqbd4zDN2RZ	chatgpt.com	/	2026-10-14T13:32:42.063Z	49			None			Medium
__Host-next-auth.csrf-token	41916684494dc7d03cb8e8f33da3978e2c19ad63d5645ed9791bfc23d09105c2%7C564de61d4c277f038dce8b186b79d7bf0fbfad59fd08fdefe2fa50d9d9948359	chatgpt.com	/	Session	158			Lax			Medium
__Secure-next-auth.session-token.0	PART1_TOKEN_DATA_
__Secure-next-auth.session-token.1	PART2_TOKEN_DATA_END`

	out := NormalizeCookie(input, "chatgpt")

	if !strings.Contains(out, "__Secure-next-auth.session-token.0=PART1_TOKEN_DATA_") {
		t.Errorf("Expected chunk .0, got: %s", out)
	}
	if !strings.Contains(out, "__Secure-next-auth.session-token.1=PART2_TOKEN_DATA_END") {
		t.Errorf("Expected chunk .1, got: %s", out)
	}
	if !strings.Contains(out, "__cf_bm=NBjfWi9ocHz9qE5GTfRB5QM6plHasVIGioUiC37TMfE-1791981552") {
		t.Errorf("Expected __cf_bm cookie, got: %s", out)
	}
}

func TestNormalizeCookieWrappedJSON(t *testing.T) {
	input := `{"source": "Cookie-Editor", "count": 1, "cookies": [{"name":"sessionKey","value":"test_session"}]}`
	out := NormalizeCookie(input, "claude")
	if !strings.Contains(out, "sessionKey=test_session") {
		t.Errorf("Expected sessionKey=test_session, got: %s", out)
	}
}

func TestNormalizeCookieJSON(t *testing.T) {
	input := `[{"name":"sessionKey","value":"sk-ant-test-12345"}]`
	out := NormalizeCookie(input, "claude")
	if !strings.Contains(out, "sessionKey=sk-ant-test-12345") {
		t.Errorf("Expected sessionKey=sk-ant-test-12345, got: %s", out)
	}
}

func TestNormalizeCookieBareToken(t *testing.T) {
	input := `sk-ant-session-plain`
	out := NormalizeCookie(input, "claude")
	if out != "sessionKey=sk-ant-session-plain" {
		t.Errorf("Expected sessionKey=sk-ant-session-plain, got: %s", out)
	}
}
