# Trading - SQLite Fake Testnet vs Simulation vs On-Chain v0.8.0

## Mode Default: SQLite Fake Testnet (3 bot + orchestrator)

`pkg/db/sqlite.go` + `pkg/telegram/manager.go`

- **Orchestrator supervisor** pegang DB, kasih tiap bot $100 fake USD awal (`db.EnsureAgent`)
- `db.Buy(agent, token, amountUSD, price)` → cek saldo, hitung `amountToken`, update `avg_price`, kurangi `usd`, insert `trades`, commit transaksi SQLite
- `db.Sell(agent, token, amountToken, price)` → cek holdings, tambah `usd`, delete/update holdings
- TxHash = `loop-buy-PEPE-10.00-123456` / `db-buy-...` / `stop-loss-...` untuk trace
- `db.Leaderboard(prices)` → `usd + holdings*price` per agent, `PnL = total - 100`
- Risk guard sebelum DB: `risk.ValidateBuy` cap 10%, `CheckStopLoss` auto -15%, `EstimateSlippage` 3%

```go
sqlite, _ := db.Open("./data/arena.db")
sqlite.EnsureAgent("degen", 100)
sqlite.Buy("degen", "PEPE", 10, 0.00001, "tx1", "viral") // 1M PEPE, usd 90
sqlite.Sell("degen", "PEPE", 500000, 0.00002, "tx2", "TP") // usd 100
board, _ := sqlite.Leaderboard(map[string]float64{"PEPE":0.00002})
```

Leaderboard: `arena leaderboard` atau `Manager.LeaderboardText()` baca dari SQLite (bukan map).

## Mode Lama: Simulation In-Memory (single-bot legacy)

`pkg/trading/simulated.go:SimulatedTrader`

- `map[string]float64` holdings + avg, tanpa DB, tanpa risk
- Dipakai `cmd/arena leaderboard` kalau `db` belum ada, atau `NewSimulatedTrader` manual
- TxHash `sim-buy-PEPE-10.00-0.000100`

## MarketWatcher Real (bukan mock) v0.7.0+

`pkg/trading/market.go:MarketWatcher` + `Manager.StartLoops`

- `GetPrices([SOL,USDC])` batch Jupiter `price.jup.ag/v6/price?ids=...` (1 request)
- `Snapshot()` → `SOL: $152, USDC: $1.00, BTC: $65000 RSI 28, ETH: $3500 RSI 45, Trending: PEPE $0.000012 vol 80k` + fallback kalau API fail
- `Manager.StartLoops` tiap agent panggil `watcher.Snapshot()` → `DecideTrade` → risk → `db.Buy/Sell` → broadcast ke `telegram.channel_id`
- Poll interval per agent: `strategy.params.poll_interval` (`konservatif 60m`, `degen/fomo 15m`)

## On-Chain Real (Next, masih stub)

`pkg/trading/solana.go` / `evm.go`

- Flow Solana: Jupiter Quote `GET /v6/quote` → `POST /v6/swap` + `solana-go` sign → broadcast → gas 0.000007 SOL (~Rp 17)
- EVM: 0x/1inch + `ethclient` → gas $0.01-0.5 di Base L2
- Butuh `solana-go` / `ethclient` (belum di `go.mod`, add saat wiring testnet)

## Risk v0.8.0

`pkg/risk/manager.go` — dipakai `Manager.runAgentTrade` sebelum `db.Buy/Sell`:

- `ValidateBuy` cap 10% portfolio per trade (50 → 10 jika saldo 100)
- `CheckStopLoss` scan holdings, `avg` vs `current` → list token drop ≤-15% → auto `db.Sell` all + broadcast `[degen] STOP-LOSS SELL PEPE ...`
- `ValidateSell` cek holdings cukup
- `EstimateSlippage` BUY *1.03, SELL *0.97, ditampilkan `BUY ... @ effPrice (slip 3.0%)`

## Portfolio & PnL

Via DB:
```go
usd, holdings, avg, _ := db.GetPortfolio("degen")
total := usd
for token, amt := range holdings { total += amt * prices[token] }
pnl := total - 100
```
Via Telegram: `/leaderboard` atau auto setelah tiap `runAgentTrade`.
