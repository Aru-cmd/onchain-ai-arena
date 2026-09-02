package risk

import (
	"testing"
)

func TestValidateBuy_Cap10Pct(t *testing.T) {
	cfg := DefaultConfig() // 10%
	// portfolio 100, try buy 50 -> should cap to 10
	capped, err := ValidateBuy(50, 100, 0, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capped != 10 {
		t.Errorf("expected capped 10 got %v", capped)
	}
	// buy 5 within cap
	capped2, err := ValidateBuy(5, 100, 0, cfg)
	if err != nil || capped2 != 5 {
		t.Errorf("expected 5 got %v err %v", capped2, err)
	}
}

func TestValidateBuy_Insufficient(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := ValidateBuy(10, 0, 0, cfg); err == nil {
		t.Error("expected error for zero balance")
	}
}

func TestCheckStopLoss(t *testing.T) {
	cfg := DefaultConfig() // 15%
	holdings := map[string]float64{"PEPE": 1000000}
	avg := map[string]float64{"PEPE": 0.00001}
	// price drops 20% -> should trigger
	current := map[string]float64{"PEPE": 0.000008} // -20%
	toSell := CheckStopLoss(holdings, avg, current, cfg)
	if len(toSell) != 1 || toSell[0] != "PEPE" {
		t.Errorf("expected PEPE stop loss, got %v", toSell)
	}
	// price drops 10% -> no trigger
	current2 := map[string]float64{"PEPE": 0.000009} // -10%
	toSell2 := CheckStopLoss(holdings, avg, current2, cfg)
	if len(toSell2) != 0 {
		t.Errorf("should not trigger, got %v", toSell2)
	}
}

func TestValidateSell(t *testing.T) {
	cfg := DefaultConfig()
	if err := ValidateSell("PEPE", 100, 1000, cfg); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if err := ValidateSell("PEPE", 2000, 1000, cfg); err == nil {
		t.Error("expected insufficient")
	}
}

func TestSlippage(t *testing.T) {
	cfg := DefaultConfig() // 3%
	if got := EstimateSlippage(100, true, cfg); got != 103 {
		t.Errorf("buy slippage expected 103 got %v", got)
	}
	if got := EstimateSlippage(100, false, cfg); got != 97 {
		t.Errorf("sell slippage expected 97 got %v", got)
	}
}
