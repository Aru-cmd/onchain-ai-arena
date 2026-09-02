package risk

import (
	"fmt"
	"strings"
)

// Config for risk management (orchestrator supervisor).
type Config struct {
	MaxPerTradePct float64 // 0.10 = max 10% portfolio per trade
	StopLossPct    float64 // 0.15 = auto sell if -15% from avg
	MaxSlippagePct float64 // 0.03 = 3% slippage
	MaxHoldings    int     // max distinct tokens (0 = unlimited)
}

func DefaultConfig() Config {
	return Config{
		MaxPerTradePct: 0.10,
		StopLossPct:    0.15,
		MaxSlippagePct: 0.03,
		MaxHoldings:    0,
	}
}

// ValidateBuy checks if BUY signal is allowed and adjusts amount.
// Returns adjustedAmountUSD and error if blocked.
func ValidateBuy(amountUSD, portfolioUSD float64, holdingsCount int, cfg Config) (float64, error) {
	if amountUSD <= 0 {
		return 0, fmt.Errorf("amount must be >0")
	}
	if portfolioUSD <= 0 {
		return 0, fmt.Errorf("no USD balance")
	}
	maxAllowed := portfolioUSD * cfg.MaxPerTradePct
	if amountUSD > maxAllowed {
		// auto-adjust to max allowed instead of reject (orchestrator caps)
		amountUSD = maxAllowed
	}
	if amountUSD < 1 {
		return 0, fmt.Errorf("amount too small after cap: %.2f", amountUSD)
	}
	if cfg.MaxHoldings > 0 && holdingsCount >= cfg.MaxHoldings {
		return 0, fmt.Errorf("max holdings %d reached", cfg.MaxHoldings)
	}
	return amountUSD, nil
}

// CheckStopLoss checks if any holding should be auto-sold.
// Returns list of tokens to sell (and amount).
func CheckStopLoss(holdings map[string]float64, avgPrice map[string]float64, currentPrices map[string]float64, cfg Config) []string {
	var toSell []string
	for token, amt := range holdings {
		avg, ok := avgPrice[token]
		if !ok || avg <= 0 {
			continue
		}
		cur, ok := currentPrices[token]
		if !ok || cur <= 0 {
			continue
		}
		drawdown := (cur - avg) / avg
		if drawdown <= -cfg.StopLossPct {
			toSell = append(toSell, token)
			_ = amt
		}
	}
	return toSell
}

// ValidateSell checks if SELL is allowed.
func ValidateSell(token string, amountToken, held float64, cfg Config) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token empty")
	}
	if amountToken <= 0 {
		return fmt.Errorf("amount must be >0")
	}
	if held < amountToken-1e-9 {
		return fmt.Errorf("insufficient holdings: have %.6f need %.6f", held, amountToken)
	}
	_ = cfg
	return nil
}

// EstimateSlippage returns price with slippage applied (for simulation).
// For BUY, effective price is higher; for SELL, lower.
func EstimateSlippage(price float64, isBuy bool, cfg Config) float64 {
	if cfg.MaxSlippagePct <= 0 {
		return price
	}
	if isBuy {
		return price * (1 + cfg.MaxSlippagePct)
	}
	return price * (1 - cfg.MaxSlippagePct)
}
