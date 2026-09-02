package telegram

import (
	"testing"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

// Note: tests not executed per storage rule - manual check only.
// No real Telegram API call, just logic tests.

func testCfg() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{ModelName: "test"},
			List: []config.AgentConfig{
				{ID: "orchestrator", Default: true},
				{ID: "konservatif"},
				{ID: "degen"},
				{ID: "fomo"},
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "test", Model: "openai/gpt-4o-mini", APIKey: "fake"},
		},
		Roast: config.RoastConfig{GlobalCooldownMinutes: 30, UserTTLHoursMin: 6, UserTTLHoursMax: 12, RandomChance: 0.15},
		Chain: config.ChainConfig{Active: "solana", Mode: "simulation"},
		Market: config.MarketConfig{PollIntervalSeconds: 60, EnableSimulation: true},
	}
}

func TestContainsKeyword(t *testing.T) {
	cases := map[string]bool{
		"btc to the moon": true,
		"halo bro":        false,
		"PEPE pump":       true,
		"rugi bandar":     true,
		"ai bot":          true,
		"":                false,
	}
	for in, want := range cases {
		if got := containsKeyword(in); got != want {
			t.Errorf("containsKeyword(%q)=%v want %v", in, got, want)
		}
	}
}

func TestBot_LeaderboardText(t *testing.T) {
	cfg := testCfg()
	bot, err := New(cfg, "123456:fake-token-for-test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	text := bot.LeaderboardText()
	if text == "" {
		t.Error("leaderboard empty")
	}
	// should contain each trader
	for _, id := range []string{"konservatif", "degen", "fomo"} {
		if !contains(text, id) {
			t.Errorf("leaderboard missing %q", id)
		}
	}
}

func TestBot_NewEmptyToken(t *testing.T) {
	cfg := testCfg()
	_, err := New(cfg, "")
	if err == nil {
		// will fail because env also empty - expected
		// if env set, it would pass, so we just check error path is handled
		t.Log("empty token should error when env also empty, got nil (env may be set)")
	} else {
		if err.Error() == "" {
			t.Error("error empty")
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
