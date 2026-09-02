# DB - SQLite Fake Testnet v0.8.0

Orchestrator supervisor pegang DB, kasih tiap bot fake USD, semua trade lewat DB (bukan map).

## Schema

`pkg/db/sqlite.go` → `modernc.org/sqlite` WAL:

```sql
portfolios(agent_id TEXT PK, usd REAL, updated_at DATETIME)
holdings(agent_id TEXT, token TEXT, amount REAL, avg_price REAL, PK(agent_id,token))
trades(id INTEGER PK, agent_id TEXT, token TEXT, action TEXT, amount_usd REAL, amount_token REAL, price REAL, tx_hash TEXT, reason TEXT, created_at DATETIME)
```

## Go API

```go
db, _ := db.Open("./data/arena.db") // atau ":memory:" buat test
defer db.Close()
db.EnsureAgent("degen", 100) // INSERT OR IGNORE
usd, holdings, avg, _ := db.GetPortfolio("degen")
db.Buy("degen", "PEPE", 10, 0.00001, "tx1", "viral") // cek saldo, avg_price, kurangi usd
db.Sell("degen", "PEPE", 500000, 0.00002, "tx2", "TP")
board, _ := db.Leaderboard(map[string]float64{"PEPE":0.00002}) // usd+holdings*price
```

`Manager` (`pkg/telegram/manager.go`) → `NewManager(cfg)` → `db.Open(cfg.DB.GetPath())` → `EnsureAgent` untuk konservatif/degen/fomo → `Bots` share `*db.DB` + `roast.Manager`.

Env:
```bash
DB_PATH=./data/arena.db # atau :memory:
```

Leaderboard:
```bash
./build/arena leaderboard -c config/config.json # baca SQLite kalau ada, fallback in-memory
# atau di Telegram: /leaderboard
```

Test: `pkg/db/sqlite_test.go` (OpenMemory, BuySell, Leaderboard, Insufficient) — ditulis tapi gak dijalankan per aturan storage.
