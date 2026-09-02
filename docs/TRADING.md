# Trading - Simulation vs On-Chain

## Simulation (Default, Rp 0)

`pkg/trading/simulated.go:SimulatedTrader`

- No private key, no gas, no RPC
- Prices from `MarketWatcher` (Jupiter/DexScreener) - real market, fake execution
- Portfolio in memory `map[string]float64`
- TxHash = `sim-buy-PEPE-10.00-0.000100` for traceability
- Avg buy price tracked for PnL

```go
tr := trading.NewSimulatedTrader("solana", "degen", 100)
tr.SetPrice("PEPE", 0.0001)
tx, price, _ := tr.Buy(ctx, "PEPE", 10) // 100k PEPE
```

## On-Chain (Next Phase)

### Solana
`pkg/trading/solana.go:SolanaTrader` (stub)

Real flow:
1. `GET /v6/quote?inputMint=So111...&outputMint=EPjF...&amount=10000000`
2. `POST /v6/swap` with `userPublicKey`
3. Sign with `github.com/gagliardetto/solana-go` (needs to be added, not in ~/go cache yet)
4. Broadcast via `solana-go/rpc` -> pay gas 0.000007 SOL (~Rp 17)

Needs: `solana-go`, wallet key, Jupiter API.

### EVM
`pkg/trading/evm.go:EVMTrader` (stub)

Real flow: 0x API quote -> ethclient sign -> broadcast -> gas $0.01-0.5 (Base L2 cheap)

Needs: `go-ethereum/ethclient` (not in ~/go cache yet, add when wiring).

## MarketWatcher

`pkg/trading/market.go:MarketWatcher`

- Off-chain, no SOL needed
- Jupiter Price API: `price.jup.ag/v6/price?ids=So111...` - free
- DexScreener: `api.dexscreener.com/latest/dex/...` - free, 300 req/min
- Polling via `time.Ticker`, not websocket for MVP
- Handler pattern: `Poll(ctx, tokens, func(token, price))`

Gas? Per transaction, not one-time. Simulation avoids gas forever. On-chain per tx Rp 17.

## Portfolio & PnL

```go
p := trader.GetPortfolio()
total := p.Value(prices) // USD + holdings*price
pnl := total - initial
```

Leaderboard via `arena leaderboard` command.

## Risk

- Check `GetBalance` before Buy
- Max 10% per trade (to be added in risk pkg)
- Slippage 3% max
