package llm

import (
	"os"
	"testing"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

func TestNewClient_EnvExpansion(t *testing.T) {
	os.Setenv("TEST_GEMINI_KEY", "test-gemini-123")
	defer os.Unsetenv("TEST_GEMINI_KEY")

	mc := &config.ModelConfig{
		ModelName: "gemini-flash",
		Model:     "google/gemini-2.0-flash",
		APIBase:   "https://generativelanguage.googleapis.com/v1beta/openai/",
		APIKey:    "${TEST_GEMINI_KEY}",
	}
	c, err := NewClient(mc)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.Model != "gemini-2.0-flash" {
		t.Errorf("expected gemini-2.0-flash got %q", c.Model)
	}
	// provider.go uses ResolvedAPIBase with env expansion
	if c.Config.ResolvedAPIBase() != "https://generativelanguage.googleapis.com/v1beta/openai/" {
		t.Errorf("unexpected base %q", c.Config.ResolvedAPIBase())
	}
}

func TestNewClient_OpenAIStrip(t *testing.T) {
	os.Setenv("TEST_OPENAI_KEY", "sk-test")
	defer os.Unsetenv("TEST_OPENAI_KEY")

	mc := &config.ModelConfig{
		ModelName: "gpt-mini",
		Model:     "openai/gpt-4o-mini",
		APIKey:    "${TEST_OPENAI_KEY}",
	}
	c, err := NewClient(mc)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if c.Model != "gpt-4o-mini" {
		t.Errorf("expected stripped gpt-4o-mini got %q", c.Model)
	}
}

func TestNewClient_OpenRouterKeepFull(t *testing.T) {
	os.Setenv("TEST_OR_KEY", "sk-or-test")
	defer os.Unsetenv("TEST_OR_KEY")

	mc := &config.ModelConfig{
		ModelName: "openrouter-claude",
		Model:     "openrouter/anthropic/claude-3.5-sonnet",
		APIBase:   "https://openrouter.ai/api/v1",
		APIKey:    "${TEST_OR_KEY}",
	}
	c, err := NewClient(mc)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if c.Model != "openrouter/anthropic/claude-3.5-sonnet" {
		t.Errorf("openrouter should keep full, got %q", c.Model)
	}
}

func TestNewClient_EmptyKeyError(t *testing.T) {
	mc := &config.ModelConfig{ModelName: "bad", Model: "openai/gpt-4o-mini", APIKey: ""}
	_, err := NewClient(mc)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestNewClientFromConfig(t *testing.T) {
	os.Setenv("TEST_GEMINI_KEY2", "key2")
	defer os.Unsetenv("TEST_GEMINI_KEY2")

	cfg := &config.Config{
		Agents: config.AgentsConfig{Defaults: config.AgentDefaults{ModelName: "gemini-flash"}},
		ModelList: []config.ModelConfig{
			{ModelName: "gemini-flash", Model: "google/gemini-2.0-flash", APIBase: "https://generativelanguage.googleapis.com/v1beta/openai/", APIKey: "${TEST_GEMINI_KEY2}"},
			{ModelName: "groq-llama", Model: "groq/llama-3.3-70b-versatile", APIBase: "https://api.groq.com/openai/v1", APIKey: "fake"},
		},
	}
	c, err := NewClientFromConfig(cfg, "gemini-flash")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if c.Config.ModelName != "gemini-flash" {
		t.Error("wrong model")
	}
	// fallback to defaults
	c2, err := NewClientFromConfig(cfg, "")
	if err != nil {
		t.Fatalf("fallback failed: %v", err)
	}
	if c2.Config.ModelName != "gemini-flash" {
		t.Error("fallback should be defaults")
	}
}

func TestResolveModelForAgent(t *testing.T) {
	os.Setenv("TEST_KEY_X", "keyx")
	defer os.Unsetenv("TEST_KEY_X")

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{ModelName: "gemini-flash"},
			List: []config.AgentConfig{
				{ID: "degen", Model: &config.AgentModelConfig{Primary: "groq-llama"}},
				{ID: "konservatif", Model: &config.AgentModelConfig{Primary: "gemini-flash"}},
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "gemini-flash", Model: "google/gemini-2.0-flash", APIBase: "https://generativelanguage.googleapis.com/v1beta/openai/", APIKey: "${TEST_KEY_X}"},
			{ModelName: "groq-llama", Model: "groq/llama-3.3-70b", APIBase: "https://api.groq.com/openai/v1", APIKey: "${TEST_KEY_X}"},
		},
	}
	c, err := ResolveModelForAgent(cfg, "degen")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if c.Config.ModelName != "groq-llama" {
		t.Errorf("expected groq-llama got %q", c.Config.ModelName)
	}
}
