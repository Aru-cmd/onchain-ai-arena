package llm

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

// Client wraps openai-go client with resolved model info.
type Client struct {
	OpenAI *openai.Client
	Model  string // actual model identifier to send
	Config *config.ModelConfig
}

// NewClient creates OpenAI-compatible client from ModelConfig.
// Supports OpenAI, OpenRouter, Groq, AIStudio (via compat endpoint), etc.
// Just set api_base accordingly.
func NewClient(mc *config.ModelConfig) (*Client, error) {
	if mc == nil {
		return nil, fmt.Errorf("model config is nil")
	}
	key := mc.GetAPIKey()
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("api_key empty for model %q", mc.ModelName)
	}
	model := mc.Model
	if model == "" {
		model = mc.ModelName
	}
	// Strip protocol prefix like "openai/gpt-4o-mini" -> "gpt-4o-mini" for API call
	// Keep full for openrouter ("openrouter/anthropic/claude-3.5") as-is.
	apiModel := extractModelID(model)

	opts := []option.RequestOption{
		option.WithAPIKey(key),
	}
	if strings.TrimSpace(mc.APIBase) != "" {
		opts = append(opts, option.WithBaseURL(strings.TrimSpace(mc.APIBase)))
	}
	// Custom baseURL already handles AIStudio compat:
	// https://generativelanguage.googleapis.com/v1beta/openai/

	c := openai.NewClient(opts...)

	return &Client{
		OpenAI: &c,
		Model:  apiModel,
		Config: mc,
	}, nil
}

// NewClientFromConfig resolves model by name from Config.ModelList.
func NewClientFromConfig(cfg *config.Config, modelName string) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(modelName) == "" {
		// fallback to defaults
		modelName = strings.TrimSpace(cfg.Agents.Defaults.ModelName)
		if modelName == "" && len(cfg.ModelList) > 0 {
			modelName = cfg.ModelList[0].ModelName
		}
	}
	mc, ok := cfg.GetModelConfig(modelName)
	if !ok {
		return nil, fmt.Errorf("model %q not found in model_list", modelName)
	}
	return NewClient(mc)
}

// ResolveModelForAgent returns client for agent's configured model.
func ResolveModelForAgent(cfg *config.Config, agentID string) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config nil")
	}
	var modelName string
	for _, a := range cfg.Agents.List {
		if strings.EqualFold(a.ID, agentID) && a.Model != nil && a.Model.Primary != "" {
			modelName = a.Model.Primary
			break
		}
	}
	if modelName == "" {
		modelName = cfg.Agents.Defaults.ModelName
	}
	return NewClientFromConfig(cfg, modelName)
}

func extractModelID(model string) string {
	// For "openai/gpt-4o-mini" -> "gpt-4o-mini"
	// For "openrouter/anthropic/claude-3.5-sonnet" -> keep as is (openrouter needs full)
	// Heuristic: if starts with "openai/" strip, else keep full.
	// AIStudio compat uses "google/gemini-..." -> strip to "gemini-..."
	if strings.HasPrefix(model, "openai/") {
		return strings.TrimPrefix(model, "openai/")
	}
	if strings.HasPrefix(model, "google/") {
		return strings.TrimPrefix(model, "google/")
	}
	return model
}
