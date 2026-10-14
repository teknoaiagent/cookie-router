package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestFindWorkingModels(t *testing.T) {
	data, err := os.ReadFile("../../../config.json")
	if err != nil {
		t.Fatalf("config error: %v", err)
	}
	var cfg struct {
		Accounts []struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
			Cookie   string `json:"cookie"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	var cgptCookie string
	for _, a := range cfg.Accounts {
		if a.Provider == "chatgpt" {
			cgptCookie = a.Cookie
			break
		}
	}

	p := New()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	accessToken, err := p.getAccessToken(ctx, cgptCookie)
	if err != nil {
		t.Fatalf("token err: %v", err)
	}
	cookieHeader := normalizeCookie(cgptCookie)
	streamDeviceID := extractCookieField(cookieHeader, "oai-did")
	if streamDeviceID == "" {
		streamDeviceID = randomUUID()
	}

	reqToken, proofToken, sErr := p.getSentinelTokens(ctx, accessToken, "", cookieHeader, streamDeviceID)
	t.Logf("Sentinel sErr=%v reqToken len=%d proofToken len=%d", sErr, len(reqToken), len(proofToken))

	modelsToTest := []string{
		"auto",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4",
		"text-davinci-002-render-sha",
		"gpt-5",
		"gpt-5-mini",
		"gpt-5-6",
		"gpt-5-6-instant",
		"gpt-5-6-thinking",
	}

	endpoints := []string{
		ChatGptBase + "/backend-api/conversation",
		ChatGptBase + "/backend-api/f/conversation",
	}

	for _, ep := range endpoints {
		t.Logf("=== Testing Endpoint: %s ===", ep)
		for _, m := range modelsToTest {
			payload := map[string]interface{}{
				"action": "next",
				"messages": []map[string]interface{}{
					{
						"id":     randomUUID(),
						"author": map[string]string{"role": "user"},
						"content": map[string]interface{}{
							"content_type": "text",
							"parts":        []string{"hi"},
						},
					},
				},
				"model":                         m,
				"parent_message_id":             randomUUID(),
				"timezone_offset_min":           -420,
				"history_and_training_disabled": false,
				"conversation_mode":             map[string]string{"kind": "primary_assistant"},
			}

			jsonBytes, _ := json.Marshal(payload)
			httpReq, _ := http.NewRequestWithContext(ctx, "POST", ep, bytes.NewReader(jsonBytes))
			httpReq.Header.Set("User-Agent", UserAgent)
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Accept", "text/event-stream")
			httpReq.Header.Set("Authorization", "Bearer "+accessToken)
			httpReq.Header.Set("Cookie", cookieHeader)
			httpReq.Header.Set("Origin", ChatGptBase)
			httpReq.Header.Set("Referer", ChatGptBase+"/")
			httpReq.Header.Set("Sec-Ch-Ua", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
			httpReq.Header.Set("Sec-Ch-Ua-Mobile", "?0")
			httpReq.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
			httpReq.Header.Set("Sec-Fetch-Dest", "empty")
			httpReq.Header.Set("Sec-Fetch-Mode", "cors")
			httpReq.Header.Set("Sec-Fetch-Site", "same-origin")
			httpReq.Header.Set("OAI-Language", "en-US")
			httpReq.Header.Set("OAI-Device-Id", streamDeviceID)

			if reqToken != "" {
				httpReq.Header.Set("Openai-Sentinel-Chat-Requirements-Token", reqToken)
			}
			if proofToken != "" {
				httpReq.Header.Set("Openai-Sentinel-Proof-Token", proofToken)
			}

			resp, err := p.client.Do(httpReq)
			if err != nil {
				t.Logf("[%s] Model: %s -> Error: %v", ep, m, err)
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Logf("[%s] Model: %s -> Status: %d Body: %s", ep, m, resp.StatusCode, string(body)[:minInt(len(body), 100)])
		}
	}
}
