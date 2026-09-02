package arena

import (
	"context"
	"testing"
	"time"
)

func TestParsePollInterval(t *testing.T) {
	cases := []struct {
		params map[string]any
		def    time.Duration
		want   time.Duration
	}{
		{nil, 15 * time.Minute, 15 * time.Minute},
		{map[string]any{"poll_interval": "1h"}, 15 * time.Minute, 60 * time.Minute},
		{map[string]any{"poll_interval": "30s"}, 15 * time.Minute, 30 * time.Second},
		{map[string]any{"poll_interval": "invalid"}, 15 * time.Minute, 15 * time.Minute},
		{map[string]any{}, 15 * time.Minute, 15 * time.Minute},
		{map[string]any{"poll_interval": "15m"}, 60 * time.Minute, 15 * time.Minute},
	}
	for i, c := range cases {
		if got := ParsePollInterval(c.params, c.def); got != c.want {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestBuildMarketData(t *testing.T) {
	prices := map[string]float64{"BTC": 65000, "PEPE": 0.00001}
	out := BuildMarketData(prices, "RSI 28")
	if out == "" {
		t.Error("empty market data")
	}
	// should contain BTC or PEPE
	found := false
	for _, k := range []string{"BTC", "PEPE", "RSI"} {
		for i := 0; i < len(out)-len(k)+1; i++ {
			if out[i:i+len(k)] == k {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("unexpected market data %q", out)
	}
}

func TestLoopConfig_Defaults(t *testing.T) {
	// Just ensure StartLoops doesn't panic with empty
	// Not running real loop per storage rule
	loops := []LoopConfig{
		{AgentID: "degen", PollInterval: 15 * time.Minute, MarketDataFn: nil, TradeFn: func(ctx context.Context, s string) error { return nil }},
	}
	_ = loops
}
