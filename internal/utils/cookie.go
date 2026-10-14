package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NormalizeCookie parses various cookie formats:
// 1. Cookie-Editor TSV / Tab-separated export (Name \t Value \t Domain \t Path \t Expiry ...)
// 2. Cookie-Editor JSON array export [{"name":"...", "value":"..."}]
// 3. HTTP Cookie header string (name=value; name2=value2; ...)
// 4. Raw token string (bare JWT or token value)
// It also automatically joins split tokens like __Secure-next-auth.session-token.0 and .1 into __Secure-next-auth.session-token
func NormalizeCookie(raw string, provider string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	cookieMap := make(map[string]string)

	// Check if JSON array or wrapped object (e.g. Cookie-Editor JSON export)
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		var items []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(raw), &items); err == nil && len(items) > 0 {
			for _, item := range items {
				if item.Name != "" && item.Value != "" {
					cookieMap[item.Name] = item.Value
				}
			}
		}
	} else if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		var wrapper struct {
			Cookies []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"cookies"`
		}
		if err := json.Unmarshal([]byte(raw), &wrapper); err == nil && len(wrapper.Cookies) > 0 {
			for _, item := range wrapper.Cookies {
				if item.Name != "" && item.Value != "" {
					cookieMap[item.Name] = item.Value
				}
			}
		}
	}

	// If not parsed from JSON, check for multiline TSV / text table or standard cookie headers
	if len(cookieMap) == 0 {
		lines := strings.Split(raw, "\n")
		isMultiLine := len(lines) > 1

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Tab-separated (typical Cookie-Editor export)
			if strings.Contains(line, "\t") {
				parts := strings.Split(line, "\t")
				if len(parts) >= 2 {
					k := strings.TrimSpace(parts[0])
					v := strings.TrimSpace(parts[1])
					if k != "" && v != "" {
						cookieMap[k] = v
						continue
					}
				}
			}

			// If line contains semicolons with '=', e.g. name=value; name2=value2
			if strings.Contains(line, ";") && strings.Contains(line, "=") {
				subparts := strings.Split(line, ";")
				for _, sp := range subparts {
					sp = strings.TrimSpace(sp)
					if eq := strings.Index(sp, "="); eq > 0 {
						k := strings.TrimSpace(sp[:eq])
						v := strings.TrimSpace(sp[eq+1:])
						if k != "" && v != "" {
							cookieMap[k] = v
						}
					}
				}
				continue
			}

			// Single line with '='
			if eq := strings.Index(line, "="); eq > 0 {
				k := strings.TrimSpace(line[:eq])
				v := strings.TrimSpace(line[eq+1:])
				if k != "" && v != "" {
					cookieMap[k] = v
				}
				continue
			}

			// Multiline space/column separated table
			if isMultiLine {
				fields := strings.Fields(line)
				if len(fields) >= 2 && !strings.Contains(fields[0], "=") {
					k := strings.TrimSpace(fields[0])
					v := strings.TrimSpace(fields[1])
					if k != "" && v != "" {
						cookieMap[k] = v
					}
				}
			}
		}
	}

	// For ChatGPT chunked session tokens:
	// If __Secure-next-auth.session-token.0 and .1 are present, KEEP them intact.
	// Do NOT inject a duplicate __Secure-next-auth.session-token alongside them,
	// because sending both chunks and merged token invalidates NextAuth and overflows header size.

	// If we successfully parsed cookies into cookieMap:
	if len(cookieMap) > 0 {
		var parts []string
		for k, v := range cookieMap {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(parts, "; ")
	}

	// Fallback for bare token without cookie name
	if !strings.Contains(raw, "=") {
		switch provider {
		case "chatgpt":
			return fmt.Sprintf("__Secure-next-auth.session-token=%s", raw)
		case "claude":
			return fmt.Sprintf("sessionKey=%s", raw)
		case "gemini":
			return fmt.Sprintf("__Secure-1PSID=%s", raw)
		case "grok":
			return fmt.Sprintf("sso=%s", raw)
		case "perplexity":
			return fmt.Sprintf("__Secure-next-auth.session-token=%s", raw)
		case "poe":
			return fmt.Sprintf("p-b=%s", raw)
		case "kimi":
			return raw
		case "qwen":
			return raw
		case "custom":
			return raw
		case "deepseek":
			return raw
		default:
			return raw
		}
	}

	return raw
}
