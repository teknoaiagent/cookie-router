package chatgpt

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/sha3"
)

var (
	sentinelCores = []int{8, 16, 24, 32}
	sentinelScreens = []int{1920 + 1080, 2560 + 1440, 1920 + 1200, 2560 + 1600}
	navigatorKeys = []string{
		"registerProtocolHandler−function registerProtocolHandler() { [native code] }",
		"storage−[object StorageManager]",
		"locks−[object LockManager]",
		"appCodeName−Mozilla",
		"webdriver−false",
		"vendor−Google Inc.",
		"cookieEnabled−true",
		"product−Gecko",
		"hardwareConcurrency−32",
	}
	documentKeys = []string{
		"_reactListeningo743lnnpvdg",
		"location",
	}
	windowKeys = []string{
		"0", "window", "self", "document", "name", "location", "customElements", "history",
	}
)

func getParseTime() string {
	loc := time.FixedZone("EST", -5*3600)
	now := time.Now().In(loc)
	return now.Format("Mon Jan 02 2006 15:04:05") + " GMT-0500 (Eastern Standard Time)"
}

func getBrowserConfig(userAgent string) []interface{} {
	rnd := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	perfMs := float64(time.Now().UnixNano()) / 1e6
	timeMs := float64(time.Now().UnixMilli())

	return []interface{}{
		sentinelScreens[rnd.Intn(len(sentinelScreens))],
		getParseTime(),
		int64(4294705152),
		0,
		userAgent,
		"https://cdn.oaistatic.com/_next/static/cXh69klOLzS0Gy2joLDRS/_ssgManifest.js?dpl=453ebaec0d44c2decab71692e1bfe39be35a24b3",
		"453ebaec0d44c2decab71692e1bfe39be35a24b3",
		"en-US",
		"en-US,es-US,en,es",
		0,
		navigatorKeys[rnd.Intn(len(navigatorKeys))],
		documentKeys[rnd.Intn(len(documentKeys))],
		windowKeys[rnd.Intn(len(windowKeys))],
		perfMs,
		randomUUID(),
		"",
		sentinelCores[rnd.Intn(len(sentinelCores))],
		timeMs - perfMs,
	}
}

func generateAnswer(seed, diff string, config []interface{}) (string, bool) {
	diffBytes, err := hex.DecodeString(diff)
	if err != nil || len(diffBytes) == 0 {
		return "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + base64.StdEncoding.EncodeToString([]byte(`"`+seed+`"`)), false
	}
	diffLen := len(diffBytes)
	seedBytes := []byte(seed)

	part1Bytes, _ := json.Marshal(config[:3])
	part2Bytes, _ := json.Marshal(config[4:9])
	part3Bytes, _ := json.Marshal(config[10:])

	// json.Marshal outputs valid JSON without extra spaces
	// config[:3] -> [A, B, C] -> we want [A, B, C,
	p1 := part1Bytes[:len(part1Bytes)-1]
	p1 = append(p1, ',')

	// config[4:9] -> [D, E, F, G, H] -> we want ,D, E, F, G, H,
	p2 := make([]byte, 0, len(part2Bytes))
	p2 = append(p2, ',')
	p2 = append(p2, part2Bytes[1:len(part2Bytes)-1]...)
	p2 = append(p2, ',')

	// config[10:] -> [J, K, ...] -> we want ,J, K, ...]
	p3 := make([]byte, 0, len(part3Bytes))
	p3 = append(p3, ',')
	p3 = append(p3, part3Bytes[1:]...)

	buf := make([]byte, 0, 1024)
	hashBuf := make([]byte, 64)

	for i := 0; i < 500000; i++ {
		buf = buf[:0]
		buf = append(buf, p1...)
		buf = strconv.AppendInt(buf, int64(i), 10)
		buf = append(buf, p2...)
		buf = strconv.AppendInt(buf, int64(i>>1), 10)
		buf = append(buf, p3...)

		baseLen := base64.StdEncoding.EncodedLen(len(buf))
		encoded := make([]byte, baseLen)
		base64.StdEncoding.Encode(encoded, buf)

		hasher := sha3.New512()
		hasher.Write(seedBytes)
		hasher.Write(encoded)
		hashBuf = hasher.Sum(hashBuf[:0])

		if bytes.Compare(hashBuf[:diffLen], diffBytes) <= 0 {
			return string(encoded), true
		}
	}

	return "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + base64.StdEncoding.EncodeToString([]byte(`"`+seed+`"`)), false
}

func getRequirementsToken(config []interface{}) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	seed := fmt.Sprintf("0.%d", time.Now().UnixNano())
	ans, _ := generateAnswer(seed, "0fffff", config)
	return "gAAAAAC" + ans
}

type sentinelRequirementsResponse struct {
	Persona     string `json:"persona"`
	Token       string `json:"token"`
	ExpireAfter int    `json:"expire_after"`
	ExpireAt    int64  `json:"expire_at"`
	Turnstile   struct {
		Required bool   `json:"required"`
		Dx       string `json:"dx"`
	} `json:"turnstile"`
	ProofOfWork struct {
		Required   bool   `json:"required"`
		Seed       string `json:"seed"`
		Difficulty string `json:"difficulty"`
	} `json:"proofofwork"`
}

func (p *Provider) getSentinelTokens(ctx context.Context, accessToken, accountID, cookieHeader, deviceID string) (requirementsToken string, proofToken string, err error) {
	config := getBrowserConfig(UserAgent)
	pReqToken := getRequirementsToken(config)

	payloadBytes, _ := json.Marshal(map[string]string{"p": pReqToken})
	req, err := http.NewRequestWithContext(ctx, "POST", ChatGptBase+"/backend-api/sentinel/chat-requirements", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create sentinel request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	req.Header.Set("Cookie", cookieHeader)
	req.Header.Set("Origin", ChatGptBase)
	req.Header.Set("Referer", ChatGptBase+"/")
	req.Header.Set("OAI-Language", "en-US")
	req.Header.Set("OAI-Device-Id", deviceID)
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to contact sentinel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("sentinel returned status %d: %s", resp.StatusCode, string(b))
	}

	var data sentinelRequirementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", fmt.Errorf("failed to decode sentinel response: %w", err)
	}

	requirementsToken = data.Token

	if data.ProofOfWork.Required && data.ProofOfWork.Seed != "" && data.ProofOfWork.Difficulty != "" {
		ans, _ := generateAnswer(data.ProofOfWork.Seed, data.ProofOfWork.Difficulty, config)
		proofToken = "gAAAAAB" + ans
	}

	return requirementsToken, proofToken, nil
}
