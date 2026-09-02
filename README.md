# On-Chain AI Agent Trading Arena

**Tanding AI vs AI - Konservatif vs Degen vs FOMO di DEX secara otomatis + roasting arena**

[![Version](https://img.shields.io/badge/version-0.1.0--alpha-blue)]()
[![Go](https://img.shields.io/badge/go-1.22+-00ADD8)]()
[![Chain](https://img.shields.io/badge/chain-Solana%20%7C%20EVM-lightgrey)]()
[![Mode](https://img.shields.io/badge/mode-simulation%20%7C%20testnet-green)]()

---

## Konsep

Buat 3-4 AI Agent dengan persona trading bertolak belakang, kasih modal testnet/micro, biarkan tanding trading di DEX otomatis. Komunitas nonton di Telegram/Discord + agent saling trash-talk + bisa ngeroast manusia yang nimbrung (TTL 6-12 jam).

```
[Telegram Group] <-> [Orchestrator (main agent)] <-> [LLM Brain (1 API Key)]
                         | spawns (whitelist)
        +----------------+----------------+
        |                |                |
  Konservatif         Degen            FOMO
  RSI<30 BTC/ETH   Viral Memecoin   Pump Chaser
```

**Kenapa Fun:** Leaderboard PnL real-time + roast otomatis tiap profit/rugpull.
**Sisi Serius:** Go + ethclient/solana-go + Jupiter/0x aggregator + risk management + event loop.

---

## Arsitektur

| Komponen | File | Fungsi |
|---|---|---|
| `AgentRegistry` | `pkg/arena/registry.go` | whitelist `CanSpawnSubagent` |
| `NormalizeAgentID` | `pkg/arena/routing.go` | sanitasi ID |
| `SubagentsConfig` | `pkg/config/config.go` | `allow_agents` |
| `SubagentManager` | `pkg/arena/subagent.go` | async/sync spawn |
| `InboundContext` | `pkg/bus/types.go` | routing pesan |

### Flow Event Loop

1. **Tick** 1-15 menit (`time.Ticker`) per agent (Konservatif 60m, Degen 15m)
2. **Fetch** harga via `MarketWatcher` (Jupiter Price API / DexScreener - gratis, off-chain)
3. **Decision** LLM dengan system prompt persona -> JSON `{action, token, reason}`
4. **Risk Check** off-chain (max loss 10%, slippage 3%)
5. **Execution** via `Trader` interface -> `SimulatedTrader` (Rp 0) atau `SolanaTrader`/`EVMTrader` (gas Rp 17)
6. **Publish** ke Telegram + trigger roast ke agent lain
7. **Human Roast** via `roast.Manager` (TTL 6-12 jam, global cooldown 30m, chance 15%)

---

## Struktur Folder

```
onchain-ai-arena/
├── cmd/arena/main.go          # cobra CLI: run, leaderboard, version
├── config/config.json         # contoh 4 agent + dispatch
├── pkg/
│   ├── arena/                 # core: registry, agent, subagent, routing
│   ├── bus/                   # InboundContext
│   ├── config/                # Config struct
│   ├── trading/               # Trader interface + Simulated/Solana/EVM + MarketWatcher
│   └── roast/                 # TTL manager 6-12 jam
├── docs/                      # docs lengkap
├── go.mod
├── Makefile
└── VERSION (0.1.0-alpha)
```

---

## Quick Start (Simulation - Rp 0)

```bash
git clone https://github.com/Aru-cmd/onchain-ai-arena.git
cd onchain-ai-arena

# 1. cek config
cat config/config.json

# 2. build
make build

# 3. run orchestrator (simulation mode)
./build/arena run -c config/config.json

# 4. cek leaderboard
./build/arena leaderboard -c config/config.json
```

**Butuh:** Go 1.22+, tanpa API key untuk simulation. Untuk LLM roast, isi `.env`:

```bash
cp .env.example .env
# isi OPENAI_API_KEY atau GEMINI_API_KEY, TELEGRAM_BOT_TOKEN
```

---

## Chain & Mode

| Mode | Biaya | Verifiable | Cocok buat |
|---|---|---|---|
| `simulation` | Rp 0 selamanya | Tidak | MVP, test strategi, roast |
| `testnet` | Rp 0 (faucet) | Ya (Devnet) | Pamer on-chain, gas ~Rp 17 |
| `mainnet` | $5/agent + gas Rp 17/tx | Ya | Production |

**Rekomendasi:** Mulai `simulation` dulu. SOL/EVM sama murah di simulation. Nanti ganti `chain.mode` ke `testnet` dan implement `SolanaTrader`/`EVMTrader`.

- **Solana Devnet:** RPC `https://api.devnet.solana.com` (public gratis), Jupiter API gratis, faucet `solfaucet.com`
- **EVM Base Sepolia:** RPC `https://sepolia.base.org`, aggregator 0x/1inch

Trader interface sudah chain-agnostic, tinggal `NewSolanaTrader` / `NewEVMTrader`:

```go
var trader trading.Trader
if cfg.Chain.Active == "solana" {
    trader = trading.NewSolanaTrader(cfg.Chain.SolanaRPC, cfg.Chain.JupiterAPI, agentID, 100)
} else {
    trader = trading.NewEVMTrader(cfg.Chain.EVMRPC, "0x", 84532, agentID, 100)
}
```

---

## Roast Manusia (TTL 6-12 Jam)

`pkg/roast/manager.go` - in-memory, ganti Redis nanti.

```go
mgr := roast.NewManager(roast.Config{
    GlobalCooldownMinutes: 30,
    TTLHoursMin: 6, TTLHoursMax: 12,
    RandomChance: 0.15,
})
should, _ := mgr.ShouldRoast(userID, mentioned, hasKeyword)
if should {
    prompt := roast.PersonaRoast("degen", "Budi", "btc scam?")
    // call LLM with prompt -> send to Telegram
}
```

- `mentionOnly=false` -> 15% random sniper + 100% kalau mention bot
- Per-user TTL random 6-12 jam, global cooldown 30 menit
- Persona beda gaya ejekan

---

## Konfigurasi Agent

`config/config.json` - `agents.list`:

```json
{
  "id": "orchestrator",
  "subagents": {"allow_agents": ["konservatif","degen","fomo"]},
  "strategy": {"type": "orchestrator"}
}
```

Whitelist `CanSpawnSubagent` cek `allow_agents` - kalau `nil` atau kosong = gak boleh spawn. `["*"]` = boleh semua.

---

## Testing

Tests sudah ditulis tapi **belum dijalankan** sesuai aturan (cek dulu dependensi di `~/go`).

```bash
# nanti setelah cek:
go test ./... -count=1
# atau
make check
```

Coverage:
- `pkg/arena/registry_test.go` - spawn whitelist, routing
- `pkg/trading/simulated_test.go` - buy/sell, portfolio value
- `pkg/roast/manager_test.go` - TTL, mentionOnly, global cooldown
- `pkg/config/config_test.go` - validation

---

## Roadmap

- [x] v0.1.0-alpha: Registry + SimulatedTrader + RoastManager + MarketWatcher stub + docs
- [ ] v0.2.0: Wire Telegram bot (telego), LLM call (resty), real MarketWatcher polling
- [ ] v0.3.0: Solana on-chain via `solana-go` + Jupiter Swap (testnet)
- [ ] v0.4.0: EVM on-chain via `ethclient` + 0x (Base Sepolia)
- [ ] v0.5.0: Redis for roast TTL, Postgres for portfolio, Web leaderboard
- [ ] v1.0.0: Mainnet micro + dashboard

---

## Biaya

| Komponen | Gratis? | Catatan |
|---|---|---|
| Watch Market (Jupiter/DexScreener) | Ya | 300 req/menit free |
| Telegram Bot | Ya | @BotFather |
| LLM (Gemini Flash/Groq) | Free tier | 1500 req/hari cukup |
| SOL Devnet gas | Ya (faucet) | 2 SOL/request |
| SOL Mainnet gas | Rp 17/tx | 100 tx = Rp 1.700 |
| Simulasi | Rp 0 selamanya | No gas |

---

## Aturan Proyek

- Tests ditulis dulu, dijalankan nanti setelah cek `~/go` cache
- Reuse dependensi dari `~/go/pkg/mod` (resty, zerolog, cobra, uuid sudah cached)
- Versioning via `VERSION` + git tags

---

## Lisensi

MIT - Bebas dipakai buat trading arena komunitas. Jangan pakai buat scam.
```

