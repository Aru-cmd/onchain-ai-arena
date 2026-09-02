package arena

import (
	"testing"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/bus"
)

func testConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{ModelName: "test-model"},
			List: []config.AgentConfig{
				{ID: "orchestrator", Name: "Master", Default: true, Subagents: &config.SubagentsConfig{AllowAgents: []string{"konservatif", "degen", "fomo"}}},
				{ID: "konservatif", Name: "Boomer"},
				{ID: "degen", Name: "Degen"},
				{ID: "fomo", Name: "Fomo"},
			},
		},
	}
}

func TestRegistry_CanSpawnSubagent_Allowed(t *testing.T) {
	cfg := testConfig()
	r := NewAgentRegistry(cfg)
	if !r.CanSpawnSubagent("orchestrator", "degen") {
		t.Error("expected orchestrator can spawn degen")
	}
	if !r.CanSpawnSubagent("orchestrator", "konservatif") {
		t.Error("expected orchestrator can spawn konservatif")
	}
}

func TestRegistry_CanSpawnSubagent_Denied(t *testing.T) {
	cfg := testConfig()
	r := NewAgentRegistry(cfg)
	if r.CanSpawnSubagent("degen", "fomo") {
		t.Error("degen should NOT be able to spawn fomo (no allow_agents)")
	}
	if r.CanSpawnSubagent("orchestrator", "unknown") {
		t.Error("should deny unknown target")
	}
}

func TestRegistry_CanSpawnWildcard(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{ModelName: "m"},
			List: []config.AgentConfig{
				{ID: "parent", Subagents: &config.SubagentsConfig{AllowAgents: []string{"*"}}},
				{ID: "child1"},
				{ID: "child2"},
			},
		},
	}
	r := NewAgentRegistry(cfg)
	if !r.CanSpawnSubagent("parent", "child1") {
		t.Error("wildcard should allow child1")
	}
	if !r.CanSpawnSubagent("parent", "anything") {
		t.Error("wildcard should allow anything existing")
	}
}

func TestRegistry_ResolveRoute(t *testing.T) {
	cfg := testConfig()
	// add dispatch rule
	mention := true
	cfg.Agents.Dispatch = &config.DispatchConfig{
		Rules: []config.DispatchRule{
			{Name: "tg", Agent: "orchestrator", When: config.DispatchSelector{Channel: "telegram"}},
			{Name: "mention", Agent: "degen", When: config.DispatchSelector{Mentioned: &mention}},
		},
	}
	r := NewAgentRegistry(cfg)
	route := r.ResolveRoute(bus.InboundContext{Channel: "telegram", ChatID: "123"})
	if route != "orchestrator" {
		t.Errorf("expected orchestrator got %q", route)
	}
}

func TestNormalizeAgentID(t *testing.T) {
	cases := map[string]string{
		"Konservatif": "konservatif",
		"DEGEN!!":     "degen",
		"":            "main",
		"  fomo  ":    "fomo",
	}
	for in, want := range cases {
		if got := NormalizeAgentID(in); got != want {
			t.Errorf("NormalizeAgentID(%q)=%q want %q", in, got, want)
		}
	}
}
