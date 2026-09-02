package arena

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// LoopConfig per agent ticker.
type LoopConfig struct {
	AgentID      string
	PollInterval time.Duration
	MarketDataFn func(ctx context.Context) (string, error) // fetch market snapshot
	TradeFn      func(ctx context.Context, marketData string) error
}

// ParsePollInterval parses "15m", "1h", "60s" from strategy params, default 15m.
func ParsePollInterval(params map[string]any, def time.Duration) time.Duration {
	if params == nil {
		return def
	}
	raw, ok := params["poll_interval"]
	if !ok {
		return def
	}
	s, ok := raw.(string)
	if !ok {
		return def
	}
	if d, err := time.ParseDuration(strings.TrimSpace(s)); err == nil {
		return d
	}
	return def
}

// StartLoops starts per-agent tickers until ctx done. Blocks.
func StartLoops(ctx context.Context, loops []LoopConfig) {
	for _, lc := range loops {
		lc := lc
		go func() {
			ticker := time.NewTicker(lc.PollInterval)
			defer ticker.Stop()
			log.Info().Str("agent", lc.AgentID).Dur("interval", lc.PollInterval).Msg("loop started")
			for {
				select {
				case <-ctx.Done():
					log.Info().Str("agent", lc.AgentID).Msg("loop stopped")
					return
				case <-ticker.C:
					marketData := "BTC $65000 RSI 28, ETH $3500 RSI 45, SOL $150, PEPE +120% vol"
					if lc.MarketDataFn != nil {
						if md, err := lc.MarketDataFn(ctx); err == nil && md != "" {
							marketData = md
						} else if err != nil {
							log.Warn().Err(err).Str("agent", lc.AgentID).Msg("market fetch failed, using fallback")
						}
					}
					if err := lc.TradeFn(ctx, marketData); err != nil {
						log.Warn().Err(err).Str("agent", lc.AgentID).Msg("trade tick failed")
					}
				}
			}
		}()
	}
	<-ctx.Done()
}

// BuildMarketData helper to format prices for LLM prompt.
func BuildMarketData(prices map[string]float64, extra string) string {
	var sb strings.Builder
	for token, price := range prices {
		sb.WriteString(fmt.Sprintf("%s: $%.6f, ", token, price))
	}
	if extra != "" {
		sb.WriteString(extra)
	}
	return strings.TrimSuffix(sb.String(), ", ")
}
