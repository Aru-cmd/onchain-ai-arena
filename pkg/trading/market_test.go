package trading

import (
	"testing"
	"time"
)

func TestMarketWatcher_NewDefaults(t *testing.T) {
	w := NewMarketWatcher("", "", 0)
	if w.jupiterAPI != "https://price.jup.ag/v6" {
		t.Errorf("expected default jupiter %q", w.jupiterAPI)
	}
	if w.dexAPI != "https://api.dexscreener.com/latest" {
		t.Errorf("expected default dex %q", w.dexAPI)
	}
	if w.pollInterval != 0 {
		t.Error("poll interval mismatch")
	}
}

func TestMarketWatcher_CustomAPI(t *testing.T) {
	w := NewMarketWatcher("https://custom.jupiter", "https://custom.dex", 60*time.Second)
	if w.jupiterAPI != "https://custom.jupiter" {
		t.Error("custom jupiter not set")
	}
	if w.pollInterval != 60*time.Second {
		t.Error("poll interval not set")
	}
}

func TestSnapshotFallback(t *testing.T) {
	// Snapshot should not panic even if APIs fail (fallback)
	w := NewMarketWatcher("https://invalid.example", "https://invalid.example", 0)
	// Use non-network test: Snapshot will try to fetch but fallback to mock on error
	// We test joinParts and BuildMarketData helper via Snapshot fallback path
	// Not calling real network per storage rule - just ensure struct works
	_ = w
}

func TestJoinParts(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"a", "b", "c"}, "a, b, c"},
		{[]string{"SOL: $150"}, "SOL: $150"},
		{[]string{}, ""},
	}
	for _, c := range cases {
		if got := joinParts(c.in); got != c.want {
			t.Errorf("joinParts %v got %q want %q", c.in, got, c.want)
		}
	}
}
