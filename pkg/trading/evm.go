package trading

import (
	"context"
	"fmt"
)

// EVMTrader stub for EVM chains (Base, BSC) via 0x / 1inch aggregator + ethclient.
// For MVP, use SimulatedTrader.

type EVMTrader struct {
	rpcURL    string
	aggregator string // 0x, 1inch
	chainID   int64
	portfolio *Portfolio
}

func NewEVMTrader(rpcURL, aggregator string, chainID int64, agentID string, initialUSD float64) *EVMTrader {
	return &EVMTrader{
		rpcURL:     rpcURL,
		aggregator: aggregator,
		chainID:    chainID,
		portfolio:  NewPortfolio(agentID, initialUSD),
	}
}

func (e *EVMTrader) Chain() string { return "evm" }

func (e *EVMTrader) GetPrice(ctx context.Context, tokenAddr string) (float64, error) {
	return 0, fmt.Errorf("evm GetPrice: not implemented - use MarketWatcher or SimulatedTrader")
}

func (e *EVMTrader) GetBalance(ctx context.Context) (float64, error) {
	return e.portfolio.USD, nil
}

func (e *EVMTrader) Buy(ctx context.Context, tokenAddr string, amountUSD float64) (string, float64, error) {
	// Real flow: 0x quote -> sign with ethclient -> broadcast -> pay gas $0.01-0.5
	return "", 0, fmt.Errorf("evm Buy: not implemented - wire ethclient + 0x API")
}

func (e *EVMTrader) Sell(ctx context.Context, tokenAddr string, amountTokens float64) (string, float64, error) {
	return "", 0, fmt.Errorf("evm Sell: not implemented")
}
