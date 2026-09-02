# Chain - Solana vs EVM vs Gabungan

## Rekomendasi: SOL Dulu (MVP), EVM Season 2

| Aspek | SOL Only | EVM Only | Gabungan |
|---|---|---|---|
| Memecoin market | Juara (Pump.fun) | Base/BSC ada tapi kalah | Paling luas |
| Konservatif BTC/ETH | Wrapped | Native juara | Luas |
| Fee | $0.001 (murah) | L1 mahal, L2 murah | 2x complexity |
| Dev lib | solana-go + Jupiter | ethclient + 0x | 2x |
| Testnet | Devnet mudah | Sepolia ribet | 2x |

## Interface Sudah Chain-Agnostic

`pkg/trading/trader.go:Trader`

```go
type Trader interface {
    Chain() string
    GetPrice(ctx, tokenMint) (float64, error)
    Buy(ctx, tokenMint, amountUSD) (txHash, price, error)
    Sell(ctx, tokenMint, amountToken) (txHash, price, error)
    GetBalance(ctx) (float64, error)
}
```

Implementasi:
- `SimulatedTrader` - for all chains in simulation
- `SolanaTrader` - stub, needs solana-go + Jupiter Swap
- `EVMTrader` - stub, needs ethclient + 0x

Factory:

```go
func NewTrader(chain, agentID string, cfg ChainConfig) Trader {
    if cfg.Mode == "simulation" {
        return NewSimulatedTrader(chain, agentID, 100)
    }
    if chain == "solana" {
        return NewSolanaTrader(cfg.SolanaRPC, cfg.JupiterAPI, agentID, 100)
    }
    return NewEVMTrader(cfg.EVMRPC, "0x", 84532, agentID, 100)
}
```

## Gas

- Per transaksi wajib, gak bisa sekali bayar gratis selamanya (anti-spam validator)
- Solana: 0.000007 SOL ~Rp 17
- Sponsored (Paymaster) bisa bikin user merasa gratis, tapi treasury yang bayar tetap per tx
- Simulation: Rp 0 selamanya (off-chain)

## Market Watch vs Execution

- Watch: off-chain via Jupiter/DexScreener (no gas, no SOL)
- Execution: on-chain via Trader (needs gas)

Jangan campur. Watch dulu gratis, baru kalau BUY/SELL pakai SOL.

## RPC

- MVP: `https://api.devnet.solana.com` public gratis (no key)
- Prod: Helius Free Tier 100k req/mo or Alchemy
- Already in ~/go cache: resty for HTTP, need to add solana-go/ethclient later (not in cache now, so stub first)

