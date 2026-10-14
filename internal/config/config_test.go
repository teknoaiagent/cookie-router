package config

import (
	"testing"
)

func TestFindCandidateAccounts_RoundRobin(t *testing.T) {
	m := &Manager{
		cfg: &Config{
			Accounts: []*Account{
				{
					ID:              "acc1",
					Provider:        "chatgpt",
					Name:            "ChatGPT Account 1",
					DefaultModel:    "gpt-5-5-instant",
					AvailableModels: []string{"gpt-5-5-instant", "gpt-5-5"},
					Enabled:         true,
					Status:          "active",
				},
				{
					ID:              "acc2",
					Provider:        "chatgpt",
					Name:            "ChatGPT Account 2",
					DefaultModel:    "gpt-5-5-instant",
					AvailableModels: []string{"gpt-5-5-instant", "gpt-5-5"},
					Enabled:         true,
					Status:          "active",
				},
			},
		},
	}

	// Call 1
	c1 := m.FindCandidateAccounts("gpt-5-5-instant")
	if len(c1) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(c1))
	}
	if c1[0].ID != "acc1" || c1[1].ID != "acc2" {
		t.Errorf("call 1: expected [acc1, acc2], got [%s, %s]", c1[0].ID, c1[1].ID)
	}

	// Call 2 (Round Robin should rotate!)
	c2 := m.FindCandidateAccounts("gpt-5-5-instant")
	if len(c2) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(c2))
	}
	if c2[0].ID != "acc2" || c2[1].ID != "acc1" {
		t.Errorf("call 2: expected [acc2, acc1], got [%s, %s]", c2[0].ID, c2[1].ID)
	}

	// Call 3 (Round Robin wraps around!)
	c3 := m.FindCandidateAccounts("gpt-5-5-instant")
	if len(c3) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(c3))
	}
	if c3[0].ID != "acc1" || c3[1].ID != "acc2" {
		t.Errorf("call 3: expected [acc1, acc2], got [%s, %s]", c3[0].ID, c3[1].ID)
	}
}
