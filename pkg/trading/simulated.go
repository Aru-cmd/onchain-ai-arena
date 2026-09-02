package trading

import (
	"context"
	"fmt"
	"sync"
)

// SimulatedTrader is paper trading implementation - Rp 0 selamanya.
// No private key, no gas, no RPC. Just DB in memory.
// Docs: simulasi = harga real dari API, eksekusi fake di map.
type SimulatedTrader struct {
	chain     string
	portfolio *Portfolio
	prices    map[string]float64 // mock price feed
	mu        sync.RWMutex
}

func NewSimulatedTrader(chain, agentID string, initialUSD float64) *SimulatedTrader {
	return &SimulatedTrader{
		chain:     chain,
		portfolio: NewPortfolio(agentID, initialUSD),
		prices:    make(map[string]float64),
	}
}

func (s *SimulatedTrader) Chain() string { return s.chain }

func (s *SimulatedTrader) SetPrice(token string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prices[token] = price
}

func (s *SimulatedTrader) GetPrice(_ context.Context, tokenMint string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.prices[tokenMint]
	if !ok {
		return 0, fmt.Errorf("price not found for %s", tokenMint)
	}
	return p, nil
}

func (s *SimulatedTrader) GetBalance(_ context.Context) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.portfolio.USD, nil
}

func (s *SimulatedTrader) Buy(_ context.Context, tokenMint string, amountUSD float64) (string, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	price, ok := s.prices[tokenMint]
	if !ok {
		return "", 0, fmt.Errorf("price not found for %s", tokenMint)
	}
	if s.portfolio.USD < amountUSD {
		return "", 0, fmt.Errorf("insufficient balance: have %.2f need %.2f", s.portfolio.USD, amountUSD)
	}
	amountTokens := amountUSD / price
	// update avg buy
	oldAmt := s.portfolio.Holdings[tokenMint]
	oldAvg := s.portfolio.AvgBuy[tokenMint]
	newAmt := oldAmt + amountTokens
	if oldAmt == 0 {
		s.portfolio.AvgBuy[tokenMint] = price
	} else {
		s.portfolio.AvgBuy[tokenMint] = (oldAmt*oldAvg + amountTokens*price) / newAmt
	}
	s.portfolio.Holdings[tokenMint] = newAmt
	s.portfolio.USD -= amountUSD
	txHash := fmt.Sprintf("sim-buy-%s-%.2f-%.6f", tokenMint, amountUSD, price)
	return txHash, price, nil
}

func (s *SimulatedTrader) Sell(_ context.Context, tokenMint string, amountTokens float64) (string, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	price, ok := s.prices[tokenMint]
	if !ok {
		return "", 0, fmt.Errorf("price not found for %s", tokenMint)
	}
	held := s.portfolio.Holdings[tokenMint]
	if held < amountTokens {
		return "", 0, fmt.Errorf("insufficient holdings: have %.6f need %.6f", held, amountTokens)
	}
	s.portfolio.Holdings[tokenMint] -= amountTokens
	if s.portfolio.Holdings[tokenMint] < 1e-9 {
		delete(s.portfolio.Holdings, tokenMint)
		delete(s.portfolio.AvgBuy, tokenMint)
	}
	proceeds := amountTokens * price
	s.portfolio.USD += proceeds
	txHash := fmt.Sprintf("sim-sell-%s-%.6f-%.6f", tokenMint, amountTokens, price)
	return txHash, price, nil
}

func (s *SimulatedTrader) GetPortfolio() *Portfolio {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// deep copy
	cp := *s.portfolio
	cp.Holdings = make(map[string]float64, len(s.portfolio.Holdings))
	for k, v := range s.portfolio.Holdings {
		cp.Holdings[k] = v
	}
	cp.AvgBuy = make(map[string]float64, len(s.portfolio.AvgBuy))
	for k, v := range s.portfolio.AvgBuy {
		cp.AvgBuy[k] = v
	}
	return &cp
}
