package trading

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// MarketWatcher watches market via off-chain APIs - no SOL needed.
// Uses Jupiter Price API + DexScreener for trending.
// SOL is only for execution, not for watching.

type MarketWatcher struct {
	client       *resty.Client
	jupiterAPI   string
	dexAPI       string
	pollInterval time.Duration
}

func NewMarketWatcher(jupiterAPI, dexAPI string, pollInterval time.Duration) *MarketWatcher {
	if jupiterAPI == "" {
		jupiterAPI = "https://price.jup.ag/v6"
	}
	if dexAPI == "" {
		dexAPI = "https://api.dexscreener.com/latest"
	}
	return &MarketWatcher{
		client:       resty.New().SetTimeout(10 * time.Second),
		jupiterAPI:   jupiterAPI,
		dexAPI:       dexAPI,
		pollInterval: pollInterval,
	}
}

// JupiterPriceResponse subset
type JupiterPriceResp struct {
	Data map[string]struct {
		Price float64 `json:"price"`
	} `json:"data"`
}

// GetPrice fetches price from Jupiter (Solana) - free, no key.
func (w *MarketWatcher) GetPrice(ctx context.Context, mint string) (float64, error) {
	var resp JupiterPriceResp
	_, err := w.client.R().
		SetContext(ctx).
		SetQueryParam("ids", mint).
		SetResult(&resp).
		Get(w.jupiterAPI + "/price")
	if err != nil {
		return 0, err
	}
	if d, ok := resp.Data[mint]; ok {
		return d.Price, nil
	}
	return 0, fmt.Errorf("price not found for %s", mint)
}

type DexScreenerPair struct {
	ChainID string `json:"chainId"`
	PairAddress string `json:"pairAddress"`
	BaseToken struct {
		Symbol string `json:"symbol"`
		Address string `json:"address"`
	} `json:"baseToken"`
	PriceUsd string `json:"priceUsd"`
	Volume struct {
		H24 float64 `json:"h24"`
	} `json:"volume"`
	PriceChange struct {
		H24 float64 `json:"h24"`
	} `json:"priceChange"`
}

type DexTrendingResp struct {
	Pairs []DexScreenerPair `json:"pairs"`
}

// GetTrending fetches trending pairs from DexScreener - free.
func (w *MarketWatcher) GetTrending(ctx context.Context, chain string) ([]DexScreenerPair, error) {
	var resp DexTrendingResp
	// DexScreener trending is via search with boost
	url := w.dexAPI + "/dex/tokens/trending"
	if chain != "" {
		url = fmt.Sprintf("%s/dex/search/?q=%s", w.dexAPI, chain)
	}
	_, err := w.client.R().
		SetContext(ctx).
		SetResult(&resp).
		Get(url)
	if err != nil {
		log.Error().Err(err).Str("chain", chain).Msg("GetTrending failed")
		return nil, err
	}
	return resp.Pairs, nil
}

// Poll starts ticker loop for each token - calls handler on price update.
func (w *MarketWatcher) Poll(ctx context.Context, tokens []string, handler func(token string, price float64)) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, tok := range tokens {
				price, err := w.GetPrice(ctx, tok)
				if err != nil {
					log.Warn().Err(err).Str("token", tok).Msg("poll price failed")
					continue
				}
				handler(tok, price)
			}
		}
	}
}
