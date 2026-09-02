package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

// Chat sends simple chat completion and returns text.
// Temperature and maxTokens from model config if not overridden.
func Chat(ctx context.Context, c *Client, systemPrompt, userPrompt string, temperature *float64, maxTokens *int) (string, error) {
	if c == nil || c.OpenAI == nil {
		return "", fmt.Errorf("client nil")
	}
	params := openai.ChatCompletionNewParams{
		Model: c.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
	}
	if temperature != nil {
		params.Temperature = param.Opt[float64]{Value: *temperature}
	}
	if maxTokens != nil && *maxTokens > 0 {
		params.MaxTokens = param.Opt[int64]{Value: int64(*maxTokens)}
	}
	resp, err := c.OpenAI.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// ChatJSON sends chat and parses JSON response into out.
func ChatJSON(ctx context.Context, c *Client, systemPrompt, userPrompt string, out any) error {
	// Force JSON mode via system prompt
	sys := systemPrompt + "\n\nRespond ONLY with valid JSON, no markdown, no explanation."
	text, err := Chat(ctx, c, sys, userPrompt, nil, nil)
	if err != nil {
		return err
	}
	// Strip ```json fences if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	if err := json.Unmarshal([]byte(text), out); err != nil {
		return fmt.Errorf("json parse failed: %w (raw: %q)", err, text)
	}
	return nil
}

// NewClientWithOptions is helper to create client with custom baseURL/key overrides (for tests).
func NewClientWithOptions(model, apiKey, baseURL string) (*Client, error) {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if strings.TrimSpace(baseURL) != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	oc := openai.NewClient(opts...)
	return &Client{OpenAI: &oc, Model: model}, nil
}
