package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/arena"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/db"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/risk"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/trading"
)

// Manager runs 3 bots (konservatif/degen/fomo) sharing SQLite DB + roast manager.
// Orchestrator is DB supervisor: gives each bot fake testnet USD.
type Manager struct {
	cfg      *config.Config
	db       *db.DB
	registry *arena.Registry
	roastMgr *roast.Manager
	bots     map[string]*Bot // agentID -> Bot
	mu       sync.Mutex
}

// NewManager creates 3-bot manager sharing DB and roast state.
func NewManager(cfg *config.Config) (*Manager, error) {
	dbPath := cfg.DB.GetPath()
	initialUSD := cfg.DB.GetInitialUSD()

	sqlite, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db %q: %w", dbPath, err)
	}
	registry := arena.NewAgentRegistry(cfg)
	roastMgr := roast.NewManager(roast.Config{
		GlobalCooldownMinutes: cfg.Roast.GlobalCooldownMinutes,
		TTLHoursMin:           cfg.Roast.UserTTLHoursMin,
		TTLHoursMax:           cfg.Roast.UserTTLHoursMax,
		RandomChance:          cfg.Roast.RandomChance,
		MentionOnly:           cfg.Roast.MentionOnly,
	})

	// ensure portfolios
	for _, id := range registry.ListAgentIDs() {
		if id == "orchestrator" {
			continue
		}
		if err := sqlite.EnsureAgent(id, initialUSD); err != nil {
			_ = sqlite.Close()
			return nil, fmt.Errorf("ensure agent %q: %w", id, err)
		}
	}

	bots := make(map[string]*Bot)
	// create per-agent bot using shared DB + shared roastMgr
	for _, agentID := range []string{"konservatif", "degen", "fomo"} {
		token := cfg.Telegram.GetToken(agentID)
		if token == "" {
			log.Warn().Str("agent", agentID).Msg("telegram token empty, skipping bot (set telegram.tokens." + agentID + ")")
			continue
		}
		// check agent exists in registry
		if _, ok := registry.GetAgent(agentID); !ok {
			log.Warn().Str("agent", agentID).Msg("agent not in registry, skip")
			continue
		}
		bot, err := NewWithDB(cfg, agentID, token, sqlite, registry, roastMgr)
		if err != nil {
			_ = sqlite.Close()
			return nil, fmt.Errorf("create bot %q: %w", agentID, err)
		}
		bots[agentID] = bot
	}
	if len(bots) == 0 {
		_ = sqlite.Close()
		return nil, fmt.Errorf("no bots created: set telegram.tokens for konservatif/degen/fomo")
	}

	return &Manager{
		cfg:      cfg,
		db:       sqlite,
		registry: registry,
		roastMgr: roastMgr,
		bots:     bots,
	}, nil
}

// NewWithDB creates per-agent bot sharing DB and roastMgr.
func NewWithDB(cfg *config.Config, agentID, token string, sqlite *db.DB, registry *arena.Registry, roastMgr *roast.Manager) (*Bot, error) {
	bot, err := New(cfg, token)
	if err != nil {
		return nil, err
	}
	// override to use shared DB + shared managers
	bot.agentID = agentID
	bot.db = sqlite
	bot.registry = registry
	bot.roastMgr = roastMgr
	// replace traders map with DB-backed: keep for compat but not used
	// trading now via DB
	bot.traders = nil
	return bot, nil
}

// Start starts all 3 bots concurrently.
func (m *Manager) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(m.bots))
	for id, bot := range m.bots {
		wg.Add(1)
		go func(agentID string, b *Bot) {
			defer wg.Done()
			log.Info().Str("agent", agentID).Msg("starting telegram bot")
			if err := b.Start(ctx); err != nil {
				errCh <- fmt.Errorf("bot %s failed: %w", agentID, err)
			}
		}(id, bot)
	}
	// wait for any error or ctx done
	go func() {
		wg.Wait()
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		m.Stop()
		return nil
	case err := <-errCh:
		m.Stop()
		return err
	}
}

// Stop stops all bots and closes DB.
func (m *Manager) Stop() {
	for _, bot := range m.bots {
		bot.Stop()
	}
	if m.db != nil {
		_ = m.db.Close()
	}
}

// Bots returns map of running bots.
func (m *Manager) Bots() map[string]*Bot { return m.bots }

// DB returns sqlite handle (orchestrator supervisor).
func (m *Manager) DB() *db.DB { return m.db }

// LeaderboardText delegates to DB.
func (m *Manager) LeaderboardText() string {
	prices := map[string]float64{"So11111111111111111111111111111111111111112": 150, "PEPE": 0.00001, "BTC": 65000, "ETH": 3500, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 1}
	board, err := m.db.Leaderboard(prices)
	if err != nil {
		return fmt.Sprintf("leaderboard error: %v", err)
	}
	out := "=== LEADERBOARD (SQLite Fake Testnet) ===\n"
	for id, total := range board {
		usd, _, _, _ := m.db.GetPortfolio(id)
		out += fmt.Sprintf("%-12s USD:%.2f Total:%.2f PnL:%+.2f\n", id, usd, total, total-100)
	}
	return out
}

// StartLoops starts auto-trading loop per agent via orchestrator cron.
// Each agent ticks at its own interval (konservatif 60m, degen/fomo 15m from strategy.params).
// Broadcasts trade results to arena channel.
func (m *Manager) StartLoops(ctx context.Context, watcher *trading.MarketWatcher) {
	loops := []arena.LoopConfig{}
	for _, agentID := range []string{"konservatif", "degen", "fomo"} {
		agent, ok := findAgentConfig(m.cfg, agentID)
		if !ok {
			continue
		}
		interval := arena.ParsePollInterval(agent.Strategy.Params, 15*time.Minute)
		if agentID == "konservatif" && interval == 15*time.Minute {
			// default konservatif slower
			if _, has := agent.Strategy.Params["poll_interval"]; !has {
				interval = 60 * time.Minute
			}
		}
		agentIDCopy := agentID
		loops = append(loops, arena.LoopConfig{
			AgentID:      agentIDCopy,
			PollInterval: interval,
				MarketDataFn: func(c context.Context) (string, error) {
				if watcher != nil {
					if snap, err := watcher.Snapshot(c); err == nil && snap != "" {
						return fmt.Sprintf("%s (agent:%s)", snap, agentIDCopy), nil
					} else if err != nil {
						log.Warn().Err(err).Str("agent", agentIDCopy).Msg("watcher snapshot failed, fallback")
					}
				}
				return fmt.Sprintf("BTC $65000 RSI 28, ETH $3500 RSI 45, SOL $150, PEPE +120%% vol (agent:%s) [fallback]", agentIDCopy), nil
			},
			TradeFn: func(c context.Context, marketData string) error {
				return m.runAgentTrade(c, agentIDCopy, marketData)
			},
		})
	}
	if len(loops) == 0 {
		log.Warn().Msg("no loops to start")
		return
	}
	go arena.StartLoops(ctx, loops)
}

func (m *Manager) runAgentTrade(ctx context.Context, agentID, marketData string) error {
	// Risk: check stop-loss before LLM decision
	if err := m.checkStopLoss(ctx, agentID); err != nil {
		log.Warn().Err(err).Str("agent", agentID).Msg("stop-loss check failed")
	}
	sig, err := arena.DecideTrade(ctx, m.cfg, agentID, marketData)
	if err != nil {
		return err
	}
	bot, ok := m.bots[agentID]
	if !ok {
		return fmt.Errorf("bot %q not found", agentID)
	}
	channelID := strings.TrimSpace(m.cfg.Telegram.ChannelID)
	if channelID == "" {
		log.Warn().Str("agent", agentID).Msg("telegram.channel_id empty, skip broadcast (set TELEGRAM_CHANNEL_ID)")
		return nil
	}
	var chatID telego.ChatID
	chatID = telego.ChatID{ID: 0}
	if strings.HasPrefix(channelID, "@") {
		chatID.Username = channelID
	} else {
		var id int64
		_, _ = fmt.Sscan(channelID, &id)
		chatID.ID = id
	}

	// Risk config (orchestrator caps)
	riskCfg := risk.DefaultConfig()
	// Execute via DB (orchestrator supervisor) with risk checks
	price := map[string]float64{"PEPE": 0.00001, "BTC": 65000, "ETH": 3500, "So11111111111111111111111111111111111111112": 150}[sig.Token]
	if price == 0 {
		price = 0.00001
	}
	// Apply slippage to price
	switch sig.Action {
	case "BUY":
		amt := sig.AmountUSD
		if amt == 0 {
			amt = 10
		}
		token := sig.Token
		if token == "" {
			token = "PEPE"
		}
		// Risk: cap to 10% per trade
		usd, holdings, _, _ := m.db.GetPortfolio(agentID)
		capped, err := risk.ValidateBuy(amt, usd, len(holdings), riskCfg)
		if err != nil {
			_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] BUY %s blocked by risk: %v", agentID, token, err)})
			return err
		}
		amt = capped
		effPrice := risk.EstimateSlippage(price, true, riskCfg)
		txHash := fmt.Sprintf("loop-buy-%s-%.2f-%d", token, amt, time.Now().UnixMilli())
		if err := m.db.Buy(agentID, token, amt, effPrice, txHash, sig.Reason); err != nil {
			_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] BUY %s failed: %v", agentID, token, err)})
			return err
		}
		text := fmt.Sprintf("[%s] BUY %s $%.2f @ %.6f (slip %.1f%%) tx:%s reason:%s", agentID, token, amt, effPrice, riskCfg.MaxSlippagePct*100, txHash, sig.Reason)
		if trash, err := arena.GenerateTrashTalk(ctx, m.cfg, agentID, text); err == nil && trash != "" {
			text += "\n" + trash
		}
		_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: text})
	case "SELL":
		token := sig.Token
		if token == "" {
			token = "PEPE"
		}
		_, holdings, _, _ := m.db.GetPortfolio(agentID)
		amt := holdings[token] / 2
		if amt <= 0 {
			_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] HOLD %s (no holdings) reason:%s", agentID, token, sig.Reason)})
			return nil
		}
		if err := risk.ValidateSell(token, amt, holdings[token], riskCfg); err != nil {
			_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] SELL %s blocked by risk: %v", agentID, token, err)})
			return err
		}
		effPrice := risk.EstimateSlippage(price, false, riskCfg)
		txHash := fmt.Sprintf("loop-sell-%s-%.2f-%d", token, amt, time.Now().UnixMilli())
		if err := m.db.Sell(agentID, token, amt, effPrice, txHash, sig.Reason); err != nil {
			_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] SELL %s failed: %v", agentID, token, err)})
			return err
		}
		_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] SELL %s %.2f @ %.6f (slip %.1f%%) tx:%s reason:%s", agentID, token, amt, effPrice, riskCfg.MaxSlippagePct*100, txHash, sig.Reason)})
	default:
		_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{ChatID: chatID, Text: fmt.Sprintf("[%s] HOLD %s reason:%s", agentID, sig.Token, sig.Reason)})
	}
	return nil
}

func (m *Manager) checkStopLoss(ctx context.Context, agentID string) error {
	_, holdings, avg, err := m.db.GetPortfolio(agentID)
	if err != nil || len(holdings) == 0 {
		return nil
	}
	// current prices mock - in real would use watcher.Snapshot
	current := map[string]float64{"PEPE": 0.000008, "BTC": 65000, "ETH": 3500, "So11111111111111111111111111111111111111112": 150}
	toSell := risk.CheckStopLoss(holdings, avg, current, risk.DefaultConfig())
	for _, token := range toSell {
		amt := holdings[token]
		price := current[token]
		txHash := fmt.Sprintf("stop-loss-%s-%.2f-%d", token, amt, time.Now().UnixMilli())
		if err := m.db.Sell(agentID, token, amt, price, txHash, "stop-loss -15%"); err == nil {
			// broadcast stop-loss
			if bot, ok := m.bots[agentID]; ok {
				channelID := strings.TrimSpace(m.cfg.Telegram.ChannelID)
				if channelID != "" {
					var chatID telego.ChatID
					if strings.HasPrefix(channelID, "@") {
						chatID.Username = channelID
					} else {
						var id int64
						_, _ = fmt.Sscan(channelID, &id)
						chatID.ID = id
					}
					_, _ = bot.bot.SendMessage(ctx, &telego.SendMessageParams{
						ChatID: chatID,
						Text:   fmt.Sprintf("[%s] STOP-LOSS SELL %s %.2f @ %.6f (auto -15%%)", agentID, token, amt, price),
					})
				}
			}
		}
	}
	return nil
}

func findAgentConfig(cfg *config.Config, id string) (*config.AgentConfig, bool) {
	norm := arena.NormalizeAgentID(id)
	for i := range cfg.Agents.List {
		if arena.NormalizeAgentID(cfg.Agents.List[i].ID) == norm {
			return &cfg.Agents.List[i], true
		}
	}
	return nil, false
}
