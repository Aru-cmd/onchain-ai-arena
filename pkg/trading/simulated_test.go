package trading

import (
	"context"
	"testing"
)

func TestSimulatedTrader_BuySell(t *testing.T) {
	tr := NewSimulatedTrader("solana", "degen", 100)
	tr.SetPrice("PEPE", 0.0001)
	ctx := context.Background()

	tx, price, err := tr.Buy(ctx, "PEPE", 10)
	if err != nil {
		t.Fatalf("buy failed: %v", err)
	}
	if tx == "" || price != 0.0001 {
		t.Errorf("unexpected buy result %q %f", tx, price)
	}
	p := tr.GetPortfolio()
	if p.Holdings["PEPE"] != 100000 {
		t.Errorf("expected 100k PEPE got %f", p.Holdings["PEPE"])
	}
	if p.USD != 90 {
		t.Errorf("expected USD 90 got %f", p.USD)
	}

	// price up, sell half
	tr.SetPrice("PEPE", 0.0002)
	tx2, _, err := tr.Sell(ctx, "PEPE", 50000)
	if err != nil {
		t.Fatalf("sell failed: %v", err)
	}
	if tx2 == "" {
		t.Error("empty sell tx")
	}
	p2 := tr.GetPortfolio()
	if p2.USD != 100 { // 90 + 50000*0.0002 = 100
		t.Errorf("expected USD 100 got %f", p2.USD)
	}
}

func TestSimulatedTrader_InsufficientBalance(t *testing.T) {
	tr := NewSimulatedTrader("solana", "fomo", 5)
	tr.SetPrice("BTC", 50000)
	_, _, err := tr.Buy(context.Background(), "BTC", 10)
	if err == nil {
		t.Error("expected insufficient balance error")
	}
}

func TestPortfolio_Value(t *testing.T) {
	p := NewPortfolio("konservatif", 50)
	p.Holdings["SOL"] = 1
	prices := map[string]float64{"SOL": 150}
	if v := p.Value(prices); v != 200 {
		t.Errorf("expected 200 got %f", v)
	}
}
