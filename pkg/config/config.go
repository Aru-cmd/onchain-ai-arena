package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config is root config for onchain-ai-arena.
type Config struct {
	Version int          `json:"version"`
	Agents  AgentsConfig `json:"agents"`
	Session SessionConfig `json:"session,omitempty"`
	Market  MarketConfig `json:"market"`
	Roast   RoastConfig  `json:"roast"`
	Chain   ChainConfig  `json:"chain"`
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
