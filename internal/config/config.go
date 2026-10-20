package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cookie-router/internal/utils"
)

type Account struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"`     // chatgpt, claude, grok, deepseek, gemini
	Name            string    `json:"name"`         // User friendly label
	Email           string    `json:"email,omitempty"`
	Plan            string    `json:"plan,omitempty"` // free, plus, team, etc.
	Cookie          string    `json:"cookie"`       // Session cookie or token
	DefaultModel    string    `json:"defaultModel"` // e.g. gpt-4o, claude-3-5-sonnet, grok-2
	AvailableModels []string  `json:"availableModels,omitempty"`
	Enabled         bool      `json:"enabled"`
	Status          string    `json:"status"` // active, error, untested
	LastError       string    `json:"lastError,omitempty"`
	LastTested      time.Time `json:"lastTested,omitempty"`
}

type Config struct {
	Port      int        `json:"port"`
	AuthToken string     `json:"authToken"` // Optional bearer token for API access
	Accounts  []*Account `json:"accounts"`
}

type Manager struct {
	mu         sync.RWMutex
	filePath   string
	cfg        *Config
	rrCounters sync.Map // map[string]*uint64
}

var (
	defaultConfigFile = "config.json"
)

func NewManager(customPath string) (*Manager, error) {
	path := customPath
	if path == "" {
		if _, err := os.Stat("/etc/cookie-router/config.json"); err == nil {
			path = "/etc/cookie-router/config.json"
		} else {
			exePath, err := os.Executable()
			if err == nil {
				path = filepath.Join(filepath.Dir(exePath), "config.json")
			} else {
				path = defaultConfigFile
			}
		}
	}

	m := &Manager{
		filePath: path,
		cfg: &Config{
			Port:      8080,
			AuthToken: "",
			Accounts:  make([]*Account, 0),
		},
	}

	if err := m.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = m.Save()
	}

	return m, nil
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if cfg.Port <= 0 {
		cfg.Port = 8080
	}
	if cfg.Accounts == nil {
		cfg.Accounts = make([]*Account, 0)
	}

	m.cfg = &cfg
	return nil
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(m.filePath, data, 0644)
}

func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accCopy := make([]*Account, len(m.cfg.Accounts))
	for i, a := range m.cfg.Accounts {
		copyA := *a
		if len(a.AvailableModels) > 0 {
			copyA.AvailableModels = make([]string, len(a.AvailableModels))
			copy(copyA.AvailableModels, a.AvailableModels)
		}
		accCopy[i] = &copyA
	}

	return Config{
		Port:      m.cfg.Port,
		AuthToken: m.cfg.AuthToken,
		Accounts:  accCopy,
	}
}

func (m *Manager) UpdateSettings(port int, authToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if port > 0 {
		m.cfg.Port = port
	}
	m.cfg.AuthToken = authToken
	return m.Save()
}

func (m *Manager) AddAccount(acc *Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if acc.ID == "" {
		acc.ID = time.Now().Format("20060102150405")
	}
	acc.Cookie = utils.NormalizeCookie(acc.Cookie, acc.Provider)
	acc.Enabled = true
	acc.Status = "untested"

	m.cfg.Accounts = append(m.cfg.Accounts, acc)
	return m.Save()
}

func (m *Manager) UpdateAccount(acc *Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, a := range m.cfg.Accounts {
		if a.ID == acc.ID {
			if acc.Cookie != "" {
				acc.Cookie = utils.NormalizeCookie(acc.Cookie, acc.Provider)
			} else {
				acc.Cookie = a.Cookie
			}
			m.cfg.Accounts[i] = acc
			return m.Save()
		}
	}
	return errors.New("account not found")
}

func (m *Manager) DeleteAccount(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, a := range m.cfg.Accounts {
		if a.ID == id {
			m.cfg.Accounts = append(m.cfg.Accounts[:i], m.cfg.Accounts[i+1:]...)
			return m.Save()
		}
	}
	return errors.New("account not found")
}

func (m *Manager) SetAccountStatus(id, status, lastErr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.cfg.Accounts {
		if a.ID == id {
			a.Status = status
			a.LastError = lastErr
			a.LastTested = time.Now()
			_ = m.Save()
			return
		}
	}
}

func (m *Manager) SetAccountModels(id string, models []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.cfg.Accounts {
		if a.ID == id {
			a.AvailableModels = models
			_ = m.Save()
			return
		}
	}
}

func (m *Manager) SetAccountProfile(id, email, plan string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.cfg.Accounts {
		if a.ID == id {
			if email != "" {
				a.Email = email
			}
			if plan != "" {
				a.Plan = plan
			}
			_ = m.Save()
			return
		}
	}
}

func (m *Manager) FindCandidateAccounts(modelName string) []*Account {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []*Account
	seen := make(map[string]bool)

	addMatch := func(a *Account) {
		if !seen[a.ID] {
			seen[a.ID] = true
			copyA := *a
			matches = append(matches, &copyA)
		}
	}

	// 1. Exact defaultModel match or in availableModels
	for _, a := range m.cfg.Accounts {
		if !a.Enabled {
			continue
		}
		if a.DefaultModel == modelName {
			addMatch(a)
			continue
		}
		for _, am := range a.AvailableModels {
			if am == modelName {
				addMatch(a)
				break
			}
		}
	}

	// 2. Prefix / provider match
	if len(matches) == 0 {
		for _, a := range m.cfg.Accounts {
			if !a.Enabled {
				continue
			}
			switch a.Provider {
			case "chatgpt":
				if modelName == "gpt-4o" || modelName == "gpt-4" || modelName == "chatgpt-4o" || modelName == "auto" || modelName == "gpt-4o-mini" || modelName == "o1" || modelName == "o3-mini" || modelName == "dall-e-3" || strings.HasPrefix(modelName, "dall-e") || strings.HasPrefix(modelName, "gpt-") {
					addMatch(a)
				}
			case "claude":
				if modelName == "claude-3-5-sonnet" || modelName == "claude-3-7-sonnet" || modelName == "claude-3-opus" || modelName == "claude-3-5-haiku" || modelName == "claude-web" || strings.HasPrefix(modelName, "claude-") {
					addMatch(a)
				}
			case "grok":
				if modelName == "grok-2" || modelName == "grok-3" || modelName == "grok-web" || modelName == "grok-beta" || strings.HasPrefix(modelName, "grok-") {
					addMatch(a)
				}
			case "deepseek":
				if modelName == "deepseek-chat" || modelName == "deepseek-reasoner" || modelName == "deepseek-v3" || modelName == "deepseek-r1" || strings.HasPrefix(modelName, "deepseek-") {
					addMatch(a)
				}
			case "gemini":
				if modelName == "gemini-2.0-flash" || modelName == "gemini-1.5-pro" || modelName == "gemini-1.5-flash" || modelName == "gemini-web" || strings.HasPrefix(modelName, "gemini-") {
					addMatch(a)
				}
			}
		}
	}

	// 3. Fallback to any enabled account
	if len(matches) == 0 {
		for _, a := range m.cfg.Accounts {
			if a.Enabled {
				addMatch(a)
			}
		}
	}

	if len(matches) <= 1 {
		return matches
	}

	// Round-Robin rotation
	val, _ := m.rrCounters.LoadOrStore(modelName, new(uint64))
	counter := val.(*uint64)
	idx := atomic.AddUint64(counter, 1) - 1
	n := uint64(len(matches))
	start := int(idx % n)

	rotated := make([]*Account, len(matches))
	for i := 0; i < len(matches); i++ {
		rotated[i] = matches[(start+i)%len(matches)]
	}
	return rotated
}

func (m *Manager) FindAccountForModel(modelName string) *Account {
	candidates := m.FindCandidateAccounts(modelName)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}
