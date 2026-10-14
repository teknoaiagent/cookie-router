package chatgpt

import (
	"strings"
	"testing"
)

func TestSentinelPoW(t *testing.T) {
	config := getBrowserConfig(UserAgent)
	if len(config) != 18 {
		t.Fatalf("expected config length 18, got %d", len(config))
	}

	pReq := getRequirementsToken(config)
	if !strings.HasPrefix(pReq, "gAAAAAC") {
		t.Fatalf("expected p token prefix gAAAAAC, got: %s", pReq[:min(20, len(pReq))])
	}

	// Test solving an easy PoW difficulty
	seed := "0.123456789"
	diff := "0fffff"
	ans, solved := generateAnswer(seed, diff, config)
	if !solved {
		t.Fatalf("failed to solve easy difficulty %s", diff)
	}
	if len(ans) == 0 {
		t.Fatalf("empty answer returned")
	}

	proofToken := "gAAAAAB" + ans
	if !strings.HasPrefix(proofToken, "gAAAAAB") {
		t.Fatalf("expected proof token prefix gAAAAAB, got: %s", proofToken[:min(20, len(proofToken))])
	}
}
