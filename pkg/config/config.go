package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config is root config for onchain-ai-arena.
type Config struct {
	Version   int            `json:"version"`
	Agents    AgentsConfig   `json:"agents"`
	Session   SessionConfig  `json:"session,omitempty"`
	ModelList []ModelConfig  `json:"model_list"`
	Market    MarketConfig   `json:"market"`
	Roast     RoastConfig    `json:"roast"`
	Chain     ChainConfig    `json:"chain"`
	Telegram  TelegramConfig `json:"telegram"`
	DB        DBConfig       `json:"db"`
}

// ModelConfig defines a single LLM provider entry (OpenAI-compatible).
// Mirrors multi-provider pattern: model_name is the alias used by agents,
// model is the actual identifier sent to the provider, api_base + api_key
// allow routing to OpenAI, OpenRouter, Groq, AIStudio (via compat), etc.
type ModelConfig struct {
	ModelName string   `json:"model_name"` // alias, e.g. "gemini-flash", "openrouter-claude"
	Model     string   `json:"model"`      // e.g. "openai/gpt-4o-mini", "google/gemini-2.0-flash", "openrouter/anthropic/claude-3.5-sonnet"
	APIBase   string   `json:"api_base,omitempty"`
	APIKey    string   `json:"api_key,omitempty"`
	APIKeys   []string `json:"api_keys,omitempty"` // alternative: multiple keys for rotation
	Fallbacks []string `json:"fallbacks,omitempty"`
	Enabled   *bool    `json:"enabled,omitempty"`
	RPM       int      `json:"rpm,omitempty"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// APIKey returns first available key with ${ENV} expansion.
func (m *ModelConfig) GetAPIKey() string {
	raw := ""
	if m.APIKey != "" {
		raw = m.APIKey
	} else if len(m.APIKeys) > 0 {
		raw = m.APIKeys[0]
	}
	if raw == "" {
		return ""
	}
	// Expand ${GEMINI_API_KEY} style from env
	if strings.HasPrefix(raw, "${") && strings.HasSuffix(raw, "}") {
		key := raw[2 : len(raw)-1]
		if v := os.Getenv(key); v != "" {
			return v
		}
		// fallback to raw if env not set
		return ""
	}
	// Also expand $VAR and ${VAR} anywhere
	return os.ExpandEnv(raw)
}

// ResolvedAPIBase returns api_base with env expansion.
func (m *ModelConfig) ResolvedAPIBase() string {
	return os.ExpandEnv(m.APIBase)
}

// IsEnabled returns true if model is enabled (default true if nil).
func (m *ModelConfig) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

// GetModelList returns enabled models.
func (c *Config) GetModelList() []ModelConfig {
	out := make([]ModelConfig, 0, len(c.ModelList))
	for _, m := range c.ModelList {
		if m.IsEnabled() {
			out = append(out, m)
		}
	}
	return out
}

// GetModelConfig finds model by alias (model_name) or full model string.
func (c *Config) GetModelConfig(name string) (*ModelConfig, bool) {
	name = strings.TrimSpace(name)
	for i := range c.ModelList {
		m := &c.ModelList[i]
		if m.ModelName == name || m.Model == name {
			return m, true
		}
	}
	return nil, false
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list"`
	Dispatch *DispatchConfig `json:"dispatch,omitempty"`
}

type AgentDefaults struct {
	Workspace         string  `json:"workspace"`
	RestrictToWorkspace bool `json:"restrict_to_workspace"`
	Provider          string  `json:"provider"`
	ModelName         string  `json:"model_name"`
	ModelFallbacks    []string `json:"model_fallbacks,omitempty"`
	MaxTokens         int     `json:"max_tokens"`
	Temperature       *float64 `json:"temperature,omitempty"`
	MaxToolIterations int     `json:"max_tool_iterations"`
}

func (d AgentDefaults) GetModelName() string { return d.ModelName }

type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Persona   string            `json:"persona,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
	Strategy  StrategyConfig    `json:"strategy,omitempty"`
}

type StrategyConfig struct {
	Type   string         `json:"type"` // konservatif, degen, fomo
	Params map[string]any `json:"params,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type DispatchConfig struct {
	Rules []DispatchRule `json:"rules,omitempty"`
}

type DispatchRule struct {
	Name   string           `json:"name,omitempty"`
	Agent  string           `json:"agent"`
	When   DispatchSelector `json:"when"`
}

type DispatchSelector struct {
	Channel   string `json:"channel,omitempty"`
	Account   string `json:"account,omitempty"`
	Chat      string `json:"chat,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Mentioned *bool  `json:"mentioned,omitempty"`
}

type SessionConfig struct {
	Dimensions    []string            `json:"dimensions,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
}

type MarketConfig struct {
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	PriceProvider       string `json:"price_provider"` // jupiter, dexscreener, coingecko
	EnableSimulation    bool   `json:"enable_simulation"`
}

type RoastConfig struct {
	Enabled               bool    `json:"enabled"`
	GlobalCooldownMinutes int     `json:"global_cooldown_minutes"`
	UserTTLHoursMin       int     `json:"user_ttl_hours_min"`
	UserTTLHoursMax       int     `json:"user_ttl_hours_max"`
	RandomChance          float64 `json:"random_chance"` // 0.15 = 15%
	MentionOnly           bool    `json:"mention_only"`
}

type ChainConfig struct {
	Active       string `json:"active"` // solana, evm, both
	SolanaRPC    string `json:"solana_rpc"`
	EVMRPC       string `json:"evm_rpc"`
	Mode         string `json:"mode"` // simulation, testnet, mainnet
	JupiterAPI   string `json:"jupiter_api"`
	DexScreenerAPI string `json:"dexscreener_api"`
}

type TelegramConfig struct {
	Tokens    map[string]string `json:"tokens"` // agent_id -> bot token, value may be ${TELEGRAM_BOT_TOKEN_KONSERVATIF}
	ChannelID string            `json:"channel_id,omitempty"` // arena channel/group id for 3 bots to join
}

func (t TelegramConfig) GetToken(agentID string) string {
	if t.Tokens == nil {
		return ""
	}
	raw, ok := t.Tokens[agentID]
	if !ok {
		// try case-insensitive
		for k, v := range t.Tokens {
			if strings.EqualFold(k, agentID) {
				raw = v
				ok = true
				break
			}
		}
		if !ok {
			return ""
		}
	}
	return os.ExpandEnv(raw)
}

type DBConfig struct {
	Path       string  `json:"path"` // e.g. "./data/arena.db" or ":memory:" for tests
	InitialUSD float64 `json:"initial_usd"`
}

func (d DBConfig) GetPath() string {
	if strings.TrimSpace(d.Path) == "" {
		return "./data/arena.db"
	}
	return os.ExpandEnv(d.Path)
}

func (d DBConfig) GetInitialUSD() float64 {
	if d.InitialUSD <= 0 {
		return 100
	}
	return d.InitialUSD
}

// Validate minimal checks.
func (c *Config) Validate() error {
	if len(c.Agents.List) == 0 {
		return fmt.Errorf("agents.list empty, need at least 1 agent")
	}
	ids := map[string]bool{}
	for _, a := range c.Agents.List {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			return fmt.Errorf("agent id empty")
		}
		if ids[id] {
			return fmt.Errorf("duplicate agent id %q", id)
		}
		ids[id] = true
	}
	if c.Roast.UserTTLHoursMin > c.Roast.UserTTLHoursMax {
		return fmt.Errorf("roast ttl min > max")
	}
	return nil
}
