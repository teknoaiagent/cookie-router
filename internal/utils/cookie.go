package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
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

// ExtractEmailFromRaw attempts to extract an email address from cookie strings, JSON, or JWT tokens.
func ExtractEmailFromRaw(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// 1. Try decoding JWT payloads
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ' ' || r == '\n' || r == '	' || r == '"' || r == '\'' || r == ','
	})
	for _, p := range parts {
		if strings.Contains(p, "=") {
			p = p[strings.Index(p, "=")+1:]
		}
		dots := strings.Split(p, ".")
		if len(dots) == 3 && strings.HasPrefix(dots[0], "eyJ") {
			// Payload is dots[1]
			seg := dots[1]
			if l := len(seg) % 4; l > 0 {
				seg += strings.Repeat("=", 4-l)
			}
			data, err := base64.URLEncoding.DecodeString(seg)
			if err == nil {
				var claims map[string]interface{}
				if err := json.Unmarshal(data, &claims); err == nil {
					if email, ok := claims["email"].(string); ok && strings.Contains(email, "@") {
						return email
					}
					if prof, ok := claims["https://api.openai.com/profile"].(map[string]interface{}); ok {
						if email, ok := prof["email"].(string); ok && strings.Contains(email, "@") {
							return email
						}
					}
					if sub, ok := claims["sub"].(string); ok && strings.Contains(sub, "@") {
						return sub
					}
				}
			}
		}
	}

	// 2. Try URL decoding
	decoded := raw
	if strings.Contains(decoded, "%40") || strings.Contains(decoded, "%20") {
		if unescaped, err := url.QueryUnescape(decoded); err == nil {
			decoded = unescaped
		}
	}

	// 3. Fallback to Regex extraction
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	matches := emailRegex.FindAllString(decoded, -1)
	for _, m := range matches {
		lower := strings.ToLower(m)
		// Filter common false positives
		if strings.Contains(lower, "example.com") ||
			strings.Contains(lower, "domain.com") ||
			strings.HasSuffix(lower, ".png") ||
			strings.HasSuffix(lower, ".jpg") ||
			strings.HasSuffix(lower, ".gif") ||
			strings.HasSuffix(lower, ".webp") {
			continue
		}
		return m
	}

	return ""
}

// ParseImageData parses base64, data URIs, or remote URLs into raw image bytes and mime type.
func ParseImageData(raw string) ([]byte, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", errors.New("empty image data")
	}

	// 1. Data URI: data:image/png;base64,...
	if strings.HasPrefix(raw, "data:") {
		commaIdx := strings.Index(raw, ",")
		if commaIdx > 0 {
			header := raw[:commaIdx]
			b64 := raw[commaIdx+1:]
			mime := "image/png"
			if strings.Contains(header, ":") && strings.Contains(header, ";") {
				start := strings.Index(header, ":") + 1
				end := strings.Index(header, ";")
				if end > start {
					mime = header[start:end]
				}
			}
			data, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				data, err = base64.URLEncoding.DecodeString(b64)
			}
			return data, mime, err
		}
	}

	// 2. HTTP/HTTPS URL
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(raw)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		mime := resp.Header.Get("Content-Type")
		if mime == "" {
			mime = "image/png"
		}
		return data, mime, err
	}

	// 3. Raw Base64 string
	data, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(data) > 0 {
		return data, "image/png", nil
	}

	return nil, "", errors.New("unsupported image format")
}
