package trading

import (
	"context"
	"time"
)

// Trader is chain-agnostic interface for trading execution.
// Abstracts Solana vs EVM differences - from discussion about Trader interface
type Trader interface {
	Chain() string // solana, evm
	GetPrice(ctx context.Context, tokenMint string) (float64, error)
	Buy(ctx context.Context, tokenMint string, amountUSD float64) (txHash string, execPrice float64, err error)
	Sell(ctx context.Context, tokenMint string, amountTokens float64) (txHash string, execPrice float64, err error)
	GetBalance(ctx context.Context) (float64, error) // USD balance
}

type MarketCandle struct {
	Token     string
	Price     float64
	Timestamp time.Time
	Volume    float64
}

type TradeSignal struct {
	Action     string  `json:"action"` // BUY, SELL, HOLD
	Token      string  `json:"token"`
	AmountUSD  float64 `json:"amount_usd,omitempty"`
	AmountToken float64 `json:"amount_token,omitempty"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type Portfolio struct {
	AgentID   string             `json:"agent_id"`
	USD       float64            `json:"usd"`
	Holdings  map[string]float64 `json:"holdings"` // token -> amount
	AvgBuy    map[string]float64 `json:"avg_buy"`  // token -> avg price
	UpdatedAt time.Time          `json:"updated_at"`
}

func NewPortfolio(agentID string, initialUSD float64) *Portfolio {
	return &Portfolio{
		AgentID:  agentID,
		USD:      initialUSD,
		Holdings: make(map[string]float64),
		AvgBuy:   make(map[string]float64),
		UpdatedAt: time.Now(),
	}
}

func (p *Portfolio) Value(prices map[string]float64) float64 {
	total := p.USD
	for token, amt := range p.Holdings {
		if price, ok := prices[token]; ok {
			total += amt * price
		}
	}
	return total
}
