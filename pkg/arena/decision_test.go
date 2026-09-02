package arena

import (
	"context"
	"testing"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

func TestPersonaPrompts_Exist(t *testing.T) {
	for _, p := range []string{"konservatif", "degen", "fomo", "orchestrator"} {
		if _, ok := PersonaPrompts[p]; !ok {
			t.Errorf("missing prompt for %q", p)
		}
	}
}

func TestDecideTrade_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{ModelName: "missing-model"},
			List: []config.AgentConfig{
				{ID: "degen", Persona: "degen", Strategy: config.StrategyConfig{Type: "degen"}},
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "missing-model", Model: "openai/gpt-4o-mini", APIKey: ""}, // empty key -> should error
		},
	}
	// Use context but key empty should fail before HTTP
	_, err := DecideTrade(context.Background(), cfg, "degen", "BTC $60000 RSI 28")
	// DecideTrade will try to create client and fail on empty key
	if err == nil {
		t.Error("expected error for empty API key")
	}
}

func TestFindAgent_Normalized(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			List: []config.AgentConfig{{ID: "Degen"}, {ID: "konservatif"}},
		},
	}
	if _, ok := findAgent(cfg, "degen"); !ok {
		t.Error("should find degen case-insensitive")
	}
	if _, ok := findAgent(cfg, "KONSERVATIF"); !ok {
		t.Error("should find konservatif")
	}
	if _, ok := findAgent(cfg, "unknown"); ok {
		t.Error("should not find unknown")
	}
}

func TestParseSignal(t *testing.T) {
	var sig struct {
		Action string `json:"action"`
		Token  string `json:"token"`
	}
	raw := "```json\n{\"action\":\"BUY\",\"token\":\"PEPE\"}\n```"
	if err := parseSignal(raw, &sig); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if sig.Action != "BUY" || sig.Token != "PEPE" {
		t.Errorf("unexpected %+v", sig)
	}
	// with noisy prefix
	raw2 := "Sure! Here is JSON: {\"action\":\"HOLD\",\"token\":\"BTC\"} thanks"
	if err := parseSignal(raw2, &sig); err != nil {
		t.Fatalf("parse2 failed: %v", err)
	}
	if sig.Action != "HOLD" {
		t.Error("expected HOLD")
	}
}
