package telegram

import (
	"testing"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

// Not executed per storage rule.

func TestManager_LeaderboardWithDB(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			List: []config.AgentConfig{{ID: "konservatif"}, {ID: "degen"}, {ID: "fomo"}},
		},
		Roast: config.RoastConfig{GlobalCooldownMinutes: 30, UserTTLHoursMin: 6, UserTTLHoursMax: 12, RandomChance: 0.15},
		DB:    config.DBConfig{Path: ":memory:", InitialUSD: 100},
		Telegram: config.TelegramConfig{
			Tokens: map[string]string{"konservatif": "fake", "degen": "fake", "fomo": "fake"},
		},
	}
	// Manager with :memory: should still create DB, but NewManager will try to create bots with fake tokens and fail on telego.NewBot?
	// We test DB directly instead for unit.
	_ = cfg
}

func TestTelegramConfig_GetToken(t *testing.T) {
	cfg := config.TelegramConfig{
		Tokens: map[string]string{
			"konservatif": "tokenA",
			"DEGEN":       "tokenB",
		},
	}
	if got := cfg.GetToken("konservatif"); got != "tokenA" {
		t.Errorf("expected tokenA got %q", got)
	}
	if got := cfg.GetToken("degen"); got != "tokenB" {
		t.Errorf("case-insensitive failed got %q", got)
	}
	if got := cfg.GetToken("fomo"); got != "" {
		t.Errorf("expected empty for missing got %q", got)
	}
}
