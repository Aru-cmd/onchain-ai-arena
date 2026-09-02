package config

import "testing"

func TestConfig_Validate_Empty(t *testing.T) {
	c := &Config{}
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty agents")
	}
}

func TestConfig_Validate_DuplicateID(t *testing.T) {
	c := &Config{
		Agents: AgentsConfig{
			List: []AgentConfig{{ID: "a"}, {ID: "a"}},
		},
		Roast: RoastConfig{UserTTLHoursMin: 6, UserTTLHoursMax: 12},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected duplicate id error")
	}
}

func TestConfig_Validate_TTL(t *testing.T) {
	c := &Config{
		Agents: AgentsConfig{List: []AgentConfig{{ID: "a"}}},
		Roast:  RoastConfig{UserTTLHoursMin: 12, UserTTLHoursMax: 6},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected ttl error")
	}
}

func TestConfig_Validate_OK(t *testing.T) {
	c := &Config{
		Agents: AgentsConfig{List: []AgentConfig{{ID: "orchestrator"}, {ID: "degen"}}},
		Roast:  RoastConfig{UserTTLHoursMin: 6, UserTTLHoursMax: 12},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
