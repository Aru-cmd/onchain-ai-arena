# LLM - Multi-Provider OpenAI Compatible v0.8.0

Semua provider format OpenAI didukung via `model_list` + `pkg/llm` pakai `openai-go/v3` + `baseURL`.

## model_list

`config/config.json`:
```json
"model_list": [
  {"model_name":"gemini-flash","model":"google/gemini-2.0-flash","api_base":"https://generativelanguage.googleapis.com/v1beta/openai/","api_key":"${GEMINI_API_KEY}"},
  {"model_name":"gpt-4o-mini","model":"openai/gpt-4o-mini","api_base":"https://api.openai.com/v1","api_key":"${OPENAI_API_KEY}"},
  {"model_name":"openrouter-claude","model":"openrouter/anthropic/claude-3.5-sonnet","api_base":"https://openrouter.ai/api/v1","api_key":"${OPENROUTER_API_KEY}"},
  {"model_name":"groq-llama","model":"groq/llama-3.3-70b-versatile","api_base":"https://api.groq.com/openai/v1","api_key":"${GROQ_API_KEY}"}
]
```

`api_key` support `${ENV}` expansion via `os.ExpandEnv`.

## Per-Agent Model

`agents.list[].model.primary` refer ke `model_list[].model_name`:

```json
{"id":"degen","model":{"primary":"openrouter-claude"}},
{"id":"konservatif","model":{"primary":"gemini-flash"}}
```

Fallback: `agents.defaults.model_name` kalau agent gak set.

## pkg/llm

`pkg/llm/provider.go`:
- `NewClient(mc)` → `openai.NewClient(option.WithAPIKey(mc.GetAPIKey()), option.WithBaseURL(mc.ResolvedAPIBase()))` + `extractModelID` (strip `openai/` / `google/` prefix, keep `openrouter/...` full)
- `NewClientFromConfig(cfg, "gemini-flash")`
- `ResolveModelForAgent(cfg, "degen")` → cari `agents.list` → `model.primary` → `GetModelConfig`

`pkg/llm/chat.go`:
- `Chat(ctx, client, system, user, temp, maxTokens)` → `client.OpenAI.Chat.Completions.New`
- `ChatJSON(ctx, client, system, user, out)` → force JSON + strip ``` fences + `json.Unmarshal`

## Arena Wiring

`pkg/arena/decision.go`:
- `PersonaPrompts[konservatif/degen/fomo]` → system prompt JSON: `{"action":"BUY/SELL/HOLD","token":"...","amount_usd":10,"reason":"...","confidence":0.0-1.0}`
- `DecideTrade(ctx, cfg, agentID, marketData)` → `ResolveModelForAgent` → `ChatJSON` → `TradeSignal`
- `GenerateRoast(ctx, cfg, persona, name, msg)` → `roast.PersonaRoast` → `Chat`
- `GenerateTrashTalk` → setelah BUY sukses

Test via tanpa Telegram:
```bash
GEMINI_API_KEY=xxx ./build/arena chat degen "PEPE pump 30% vol 100k, beli ga?"
```
