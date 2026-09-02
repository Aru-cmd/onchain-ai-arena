package trading

import (
	"context"
	"fmt"
)

// SolanaTrader is stub for real on-chain Solana execution via Jupiter + solana-go.
// For MVP, use SimulatedTrader. This shows interface for future expansion.
// Real impl needs: solana-go client, Jupiter Swap API, private key signer.

type SolanaTrader struct {
	rpcURL     string
	jupiterAPI string
	// client *solana.Client // add when wiring real deps
	// wallet *wallet.PrivateKey
	portfolio *Portfolio
}

func NewSolanaTrader(rpcURL, jupiterAPI, agentID string, initialUSD float64) *SolanaTrader {
	return &SolanaTrader{
		rpcURL:     rpcURL,
		jupiterAPI: jupiterAPI,
		portfolio:  NewPortfolio(agentID, initialUSD),
	}
}

func (s *SolanaTrader) Chain() string { return "solana" }

func (s *SolanaTrader) GetPrice(ctx context.Context, tokenMint string) (float64, error) {
	// TODO: call Jupiter Price API via MarketWatcher
	return 0, fmt.Errorf("solana GetPrice: not implemented - use MarketWatcher or SimulatedTrader for MVP")
}

func (s *SolanaTrader) GetBalance(ctx context.Context) (float64, error) {
	return s.portfolio.USD, nil
}

func (s *SolanaTrader) Buy(ctx context.Context, tokenMint string, amountUSD float64) (string, float64, error) {
	// Real flow:
	// 1. GET https://quote-api.jup.ag/v6/quote?inputMint=So111...&outputMint=tokenMint&amount=...
	// 2. POST https://quote-api.jup.ag/v6/swap with userPublicKey
	// 3. sign with solana-go
	// 4. broadcast via RPC - pay gas ~0.000007 SOL (~Rp 17)
	return "", 0, fmt.Errorf("solana Buy: not implemented - wire solana-go + Jupiter Swap API; use SimulatedTrader for simulation mode")
}

func (s *SolanaTrader) Sell(ctx context.Context, tokenMint string, amountTokens float64) (string, float64, error) {
	return "", 0, fmt.Errorf("solana Sell: not implemented - wire solana-go + Jupiter Swap API")
}
