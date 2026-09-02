# Architecture - On-Chain AI Trading Arena

## Overview

Orchestrator is main agent, traders are subagents with whitelist.

```
Inbound (Telegram/Discord) -> bus.InboundContext -> arena.Registry.ResolveRoute -> Orchestrator
Orchestrator --CanSpawnSubagent--> Konservatif/Degen/FOMO (subagents)
Subagents -> trading.Trader (Simulated | Solana | EVM) -> MarketWatcher (Jupiter/DexScreener)
All agents -> roast.Manager -> Telegram out
```

## Agent Mapping

| Komponen | Deskripsi |
|---|---|
| `AgentRegistry` + `AgentInstance` | registry multi-agent dengan workspace/model/fallback, add `Persona` + `Strategy` |
| `NormalizeAgentID` | sanitasi ID, lowercased |
| `SubagentsConfig.AllowAgents` | whitelist spawn, `*` wildcard |
| `SubagentManager.Spawn/SpawnSync` | async untuk tugas lama, sync untuk decision cepat |
| `bus.InboundContext` | channel-agnostic routing |
| `DispatchRule` | route telegram->orchestrator |

## Package Dependency

```
cmd/arena -> pkg/arena, pkg/config, pkg/trading, pkg/roast
pkg/arena -> pkg/config, pkg/bus
pkg/trading -> stdlib + resty (cached)
pkg/roast -> stdlib
pkg/config -> stdlib
```

No circular dependencies.

## Data Flow: Trading

1. `MarketWatcher.Poll` every 60s fetches prices (off-chain, free)
2. Orchestrator ticker calls each subagent's strategy function
3. Strategy returns `TradeSignal{BUY/SELL/HOLD, token, reason}`
4. Risk check (balance, slippage)
5. `Trader.Buy/Sell` -> simulated (map) or on-chain (Jupiter->sign->broadcast)
6. Update Portfolio, publish to Telegram, trigger roast

## Data Flow: Roast

1. Human message -> `bus.InboundContext` with `SenderID`, `Mentioned`, `Content`
2. `roast.Manager.ShouldRoast(userID, mentioned, hasKeyword)` checks:
   - global cooldown (30m)
   - per-user TTL (6-12h random)
   - mentionOnly flag
   - random chance (15% or 30% if keyword)
3. If true, pick random persona (konservatif/degen/fomo) -> `PersonaRoast` prompt -> LLM -> send
4. Set cooldowns, cleanup expired

## Storage

- MVP: in-memory maps (portfolio, cooldowns)
- Next: Postgres for portfolio, Redis for roast TTL (swap `roast.Manager` to use `go-redis` which is not yet in cache, so in-memory first)

## Config

Single `config.json` versioned, validated via `Config.Validate()`. Supports env override later.
