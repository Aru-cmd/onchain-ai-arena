package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/arena"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/db"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
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
