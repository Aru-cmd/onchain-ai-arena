# Architecture - On-Chain AI Trading Arena v0.8.0

## Overview v0.8.0

3 bot Telegram terpisah + 1 orchestrator DB supervisor, bukan 1 bot multi-persona.

```
Telegram Channel/Group (@arena)
  ├─ @KonservatifBot (token A) ─┐
  ├─ @DegenBot (token B) ───────┤─> Manager (orchestrator) ─> SQLite ./data/arena.db
  └─ @FomoBot (token C) ────────┘     ├─ roast.Manager (shared TTL 6-12h, 30m global)
                                      ├─ arena.Registry + arena.Loop (15m/60m cron)
                                      ├─ trading.MarketWatcher (Jupiter + DexScreener real)
                                      ├─ risk.Manager (10% cap, 15% stop-loss, 3% slippage)
                                      └─ llm.Client (openai-go, OpenAI/OpenRouter/Groq/AIStudio compat)

Flow trading: MarketWatcher.Snapshot → arena.DecideTrade (LLM per persona) → risk.ValidateBuy/Sell → db.Buy/Sell (fake testnet) → telegram broadcast + trash-talk + leaderboard
Flow roast: bus.InboundContext → roast.Manager.ShouldRoast → arena.GenerateRoast (LLM) → telegram (per-agent persona)
```

Single-bot mode legacy tetap ada (1 token handle 3 persona) untuk hemat.

## Package Dependency v0.8.0

```
cmd/arena -> pkg/arena, pkg/config, pkg/db, pkg/llm, pkg/roast, pkg/telegram, pkg/trading, pkg/risk
pkg/arena -> pkg/config, pkg/llm, pkg/roast, pkg/trading
pkg/telegram -> pkg/arena, pkg/config, pkg/db, pkg/roast, pkg/trading, pkg/risk
pkg/trading -> resty + zerolog
pkg/db -> modernc.org/sqlite (WAL)
pkg/llm -> openai-go/v3
pkg/risk -> stdlib
pkg/roast -> stdlib
pkg/config -> stdlib + os.ExpandEnv
```

No circular dependencies. Semua provider OpenAI-compatible via `model_list` + `baseURL`.

## Data Flow: Trading (Real Market)

1. `Manager.StartLoops` per agent: `konservatif 60m`, `degen 15m`, `fomo 15m` (dari `strategy.params.poll_interval`)
2. `MarketWatcher.Snapshot()` → Jupiter `price.jup.ag/v6/price?ids=SOL,USDC` + DexScreener trending → string `SOL: $152, PEPE trending $0.000012`
3. `arena.DecideTrade(ctx, cfg, agentID, marketData)` → `llm.ChatJSON` dengan `PersonaPrompts` → `TradeSignal{BUY/SELL/HOLD}`
4. `risk.CheckStopLoss` sebelum LLM (auto sell -15%)
5. `risk.ValidateBuy` cap 10% portfolio + `EstimateSlippage` 3% → `db.Buy/Sell` (SQLite, orchestrator supervisor)
6. Broadcast ke `telegram.channel_id` via bot yang sesuai + `GenerateTrashTalk` + `LeaderboardText()` dari DB

## Data Flow: Roast

1. Human msg → `bus.InboundContext` (telegram, chatID, senderID, Mentioned)
2. `roast.Manager.ShouldRoast(userID, mentioned, hasKeyword)` shared via `Manager` (global cooldown 30m + per-user TTL 6-12h random + 15%/30% chance)
3. Jika true, `arena.GenerateRoast(ctx, cfg, persona(agentID), name, msg)` → `llm.Chat` → kirim via bot per-agent (bukan random lagi, persona sesuai bot yang kena mention)

## Storage v0.8.0

- **SQLite** `./data/arena.db` (WAL, 1 writer): `portfolios(agent_id, usd)`, `holdings(agent_id, token, amount, avg_price)`, `trades(...)`. Master `Manager` yang `EnsureAgent(id, 100)` dan `Buy/Sell`. `:memory:` untuk test.
- **Roast** in-memory `map[userID]expiry` shared via `Manager` (next: Redis `SETEX` kalau mau persist cross-restart, swap tanpa ganti API)
- **Config** single `config.json` + `os.ExpandEnv` untuk `${GEMINI_API_KEY}` etc.

## Config v0.8.0

```json
"model_list": [{"model_name":"gemini-flash","model":"google/gemini-2.0-flash","api_base":".../v1beta/openai/","api_key":"${GEMINI_API_KEY}"}],
"telegram": {"channel_id":"${TELEGRAM_CHANNEL_ID}","tokens":{"konservatif":"${TOKEN_K}","degen":"${TOKEN_D}","fomo":"${TOKEN_F}"}},
"db": {"path":"./data/arena.db","initial_usd":100}
```
