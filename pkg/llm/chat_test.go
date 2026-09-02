package llm

import (
	"testing"
)

func TestChat_NilClientError(t *testing.T) {
	_, err := Chat(nil, nil, "system", "hello", nil, nil)
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c, err := NewClientWithOptions("gpt-4o-mini", "sk-test", "https://api.openai.com/v1")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if c.Model != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini got %q", c.Model)
	}
	if c.OpenAI == nil {
		t.Error("openai client nil")
	}
}
