package db

import (
	"testing"
)

// Not executed per storage rule - manual check only.

func TestDB_OpenMemory(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer db.Close()
	if err := db.EnsureAgent("konservatif", 100); err != nil {
		t.Fatalf("ensure failed: %v", err)
	}
	usd, holdings, avg, err := db.GetPortfolio("konservatif")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if usd != 100 || len(holdings) != 0 || len(avg) != 0 {
		t.Errorf("unexpected portfolio usd=%v holdings=%v", usd, holdings)
	}
}

func TestDB_BuySell(t *testing.T) {
	db, _ := Open(":memory:")
	defer db.Close()
	_ = db.EnsureAgent("degen", 100)
	if err := db.Buy("degen", "PEPE", 10, 0.00001, "tx1", "viral"); err != nil {
		t.Fatalf("buy failed: %v", err)
	}
	usd, holdings, avg, _ := db.GetPortfolio("degen")
	if usd != 90 {
		t.Errorf("expected 90 got %v", usd)
	}
	if holdings["PEPE"] != 1000000 {
		t.Errorf("expected 1M PEPE got %v", holdings["PEPE"])
	}
	if avg["PEPE"] != 0.00001 {
		t.Errorf("avg mismatch")
	}
	// price up, sell half
	if err := db.Sell("degen", "PEPE", 500000, 0.00002, "tx2", "take profit"); err != nil {
		t.Fatalf("sell failed: %v", err)
	}
	usd2, holdings2, _, _ := db.GetPortfolio("degen")
	if usd2 != 100 { // 90 + 500k*0.00002 = 100
		t.Errorf("expected 100 got %v", usd2)
	}
	if holdings2["PEPE"] != 500000 {
		t.Errorf("expected 500k got %v", holdings2["PEPE"])
	}
}

func TestDB_Leaderboard(t *testing.T) {
	db, _ := Open(":memory:")
	defer db.Close()
	_ = db.EnsureAgent("a", 100)
	_ = db.EnsureAgent("b", 100)
	_ = db.Buy("a", "PEPE", 10, 0.00001, "tx", "")
	prices := map[string]float64{"PEPE": 0.00002}
	board, err := db.Leaderboard(prices)
	if err != nil {
		t.Fatalf("board failed: %v", err)
	}
	if board["a"] != 110 { // 90 + 1M*0.00002 = 110
		t.Errorf("expected 110 got %v", board["a"])
	}
	if board["b"] != 100 {
		t.Errorf("expected 100 got %v", board["b"])
	}
}

func TestDB_Insufficient(t *testing.T) {
	db, _ := Open(":memory:")
	defer db.Close()
	_ = db.EnsureAgent("fomo", 5)
	if err := db.Buy("fomo", "BTC", 10, 50000, "tx", ""); err == nil {
		t.Error("expected insufficient error")
	}
}
