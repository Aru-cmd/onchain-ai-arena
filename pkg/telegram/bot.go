package telegram

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog/log"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/arena"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/bus"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/db"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/trading"
)

// Bot is Telegram arena bot: can be single (all traders) or per-agent (with DB).
// In 3-bot mode, each Bot has agentID and shares DB + roastMgr via Manager.
type Bot struct {
	agentID  string // empty = single-bot multi-persona, else konservatif/degen/fomo
	cfg      *config.Config
	registry *arena.Registry
	roastMgr *roast.Manager
	bot      *telego.Bot
	handler  *th.BotHandler
	traders  map[string]*trading.SimulatedTrader // legacy single-bot mode
	db       *db.DB                               // 3-bot mode: orchestrator SQLite supervisor
}

// New creates bot from config. Token from ${TELEGRAM_BOT_TOKEN} or config.
func New(cfg *config.Config, token string) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	// Also try expand from config if set
	token = os.ExpandEnv(token)
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("telegram token empty: set TELEGRAM_BOT_TOKEN or config")
	}
	bot, err := telego.NewBot(token, telego.WithDiscardLogger())
	if err != nil {
		return nil, err
	}
	registry := arena.NewAgentRegistry(cfg)
	roastMgr := roast.NewManager(roast.Config{
		GlobalCooldownMinutes: cfg.Roast.GlobalCooldownMinutes,
		TTLHoursMin:           cfg.Roast.UserTTLHoursMin,
		TTLHoursMax:           cfg.Roast.UserTTLHoursMax,
		RandomChance:          cfg.Roast.RandomChance,
		MentionOnly:           cfg.Roast.MentionOnly,
	})
	traders := make(map[string]*trading.SimulatedTrader)
	for _, id := range registry.ListAgentIDs() {
		if id == "orchestrator" {
			continue
		}
		traders[id] = trading.NewSimulatedTrader(cfg.Chain.Active, id, 100)
		// seed prices for simulation
		traders[id].SetPrice("So11111111111111111111111111111111111111112", 150)
		traders[id].SetPrice("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 1)
		traders[id].SetPrice("PEPE", 0.00001)
		traders[id].SetPrice("BTC", 65000)
		traders[id].SetPrice("ETH", 3500)
	}
	return &Bot{
		cfg:      cfg,
		registry: registry,
		roastMgr: roastMgr,
		bot:      bot,
		traders:  traders,
	}, nil
}

// Start starts long polling and handlers.
func (b *Bot) Start(ctx context.Context) error {
	updates, err := b.bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return err
	}
	bh, err := th.NewBotHandler(b.bot, updates)
	if err != nil {
		return err
	}
	b.handler = bh

	// /start
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		if update.Message == nil {
			return
		}
		chatID := update.Message.Chat.ID
		_, _ = bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   fmt.Sprintf("Arena ready: %v | Mode:%s Chain:%s\nCommands: /leaderboard /trade /roastme", b.registry.ListAgentIDs(), b.cfg.Chain.Mode, b.cfg.Chain.Active),
		})
	}, th.CommandEqual("start"))

	// /leaderboard
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		if update.Message == nil {
			return
		}
		chatID := update.Message.Chat.ID
		text := b.LeaderboardText()
		_, _ = bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   text,
		})
	}, th.CommandEqual("leaderboard"))

	// /trade - manual trigger tick for demo
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		if update.Message == nil {
			return
		}
		chatID := update.Message.Chat.ID
		b.runTradingTick(ctx, chatID)
	}, th.CommandEqual("trade"))

	// All messages (group)
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		if update.Message == nil {
			return
		}
		b.handleMessage(ctx, bot, update)
	}, th.AnyMessageWithText())

	log.Info().Msg("telegram arena polling started")
	return bh.Start()
}

// Stop gracefully.
func (b *Bot) Stop() {
	if b.handler != nil {
		b.handler.Stop()
	}
	if b.bot != nil {
		_ = b.bot.StopLongPolling()
	}
}

// handleMessage routes human messages, decides roast.
func (b *Bot) handleMessage(ctx context.Context, bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil {
		return
	}
	// Ignore bot messages to avoid loop
	if msg.From.IsBot {
		return
	}
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	// Build bus context
	in := bus.InboundContext{
		Channel:   "telegram",
		ChatID:    fmt.Sprintf("%d", chatID),
		ChatType:  strings.ToLower(string(msg.Chat.Type)),
		SenderID:  fmt.Sprintf("%d", msg.From.ID),
		MessageID: fmt.Sprintf("%d", msg.MessageID),
		Mentioned: strings.Contains(text, "@") || msg.ReplyToMessage != nil,
	}
	// Check if message is reply to bot or mentions bot
	// telego doesn't give bot username here, we treat any @ as mention for now
	hasKeyword := containsKeyword(text)
	userID := in.SenderID

	// Try roast - per-agent bot uses its own persona, single-bot picks random
	if should, _ := b.roastMgr.ShouldRoast(userID, in.Mentioned, hasKeyword); should {
		persona := b.agentID
		if persona == "" {
			personas := []string{"konservatif", "degen", "fomo"}
			persona = personas[time.Now().UnixNano()%int64(len(personas))]
		}
		name := msg.From.FirstName
		if name == "" {
			name = msg.From.Username
		}
		roastText, err := arena.GenerateRoast(ctx, b.cfg, persona, name, text)
		if err != nil {
			roastText = fmt.Sprintf("[%s] %s, diem dulu — ini arena AI 😏", persona, name)
		}
		_, _ = bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   roastText,
		})
		log.Info().Str("persona", persona).Str("user", userID).Msg("roasted human")
		return
	}

	// Also handle dispatch routing (for future: orchestrator)
	_ = b.registry.ResolveRoute(in)
}

// runTradingTick triggers each trader agent to decide via LLM and simulate trade.
// If DB is set (3-bot mode), uses SQLite via orchestrator supervisor.
func (b *Bot) runTradingTick(ctx context.Context, chatID int64) {
	marketData := "BTC $65000 RSI 28, ETH $3500 RSI 45, SOL $150, PEPE +120% vol 80k trending"
	// 3-bot mode: single agent tick
	if b.db != nil && b.agentID != "" {
		b.runSingleAgentTick(ctx, chatID, b.agentID, marketData)
		_, _ = b.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   b.LeaderboardText(),
		})
		return
	}
	// single-bot legacy: tick all traders
	for id, trader := range b.traders {
		sig, err := arena.DecideTrade(ctx, b.cfg, id, marketData)
		if err != nil {
			log.Warn().Err(err).Str("agent", id).Msg("decide failed")
			continue
		}
		var result string
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
			tx, price, err := trader.Buy(ctx, token, amt)
			if err != nil {
				result = fmt.Sprintf("[%s] BUY %s failed: %v reason:%s", id, token, err, sig.Reason)
			} else {
				result = fmt.Sprintf("[%s] BUY %s $%.2f @ %.6f tx:%s reason:%s", id, token, amt, price, tx, sig.Reason)
				if trash, err := arena.GenerateTrashTalk(ctx, b.cfg, id, result); err == nil && trash != "" {
					result += "\n" + trash
				}
			}
		case "SELL":
			token := sig.Token
			if token == "" {
				token = "PEPE"
			}
			p := trader.GetPortfolio()
			amt := p.Holdings[token] / 2
			if amt > 0 {
				tx, price, err := trader.Sell(ctx, token, amt)
				if err != nil {
					result = fmt.Sprintf("[%s] SELL %s failed: %v", id, token, err)
				} else {
					result = fmt.Sprintf("[%s] SELL %s %.2f @ %.6f tx:%s reason:%s", id, token, amt, price, tx, sig.Reason)
				}
			} else {
				result = fmt.Sprintf("[%s] HOLD %s (no holdings) reason:%s", id, token, sig.Reason)
			}
		default:
			result = fmt.Sprintf("[%s] HOLD %s reason:%s", id, sig.Token, sig.Reason)
		}
		_, _ = b.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   result,
		})
	}
	_, _ = b.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   b.LeaderboardText(),
	})
}

func (b *Bot) runSingleAgentTick(ctx context.Context, chatID int64, agentID, marketData string) {
	sig, err := arena.DecideTrade(ctx, b.cfg, agentID, marketData)
	if err != nil {
		log.Warn().Err(err).Str("agent", agentID).Msg("decide failed")
		return
	}
	var result string
	// prices for DB execution (mock)
	prices := map[string]float64{"PEPE": 0.00001, "BTC": 65000, "ETH": 3500, "So11111111111111111111111111111111111111112": 150, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 1}
	price := prices[sig.Token]
	if price == 0 {
		price = 0.00001
	}
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
		if b.db != nil {
			txHash := fmt.Sprintf("db-buy-%s-%.2f-%d", token, amt, time.Now().UnixMilli())
			if err := b.db.Buy(agentID, token, amt, price, txHash, sig.Reason); err != nil {
				result = fmt.Sprintf("[%s] BUY %s failed: %v reason:%s", agentID, token, err, sig.Reason)
			} else {
				result = fmt.Sprintf("[%s] BUY %s $%.2f @ %.6f tx:%s reason:%s", agentID, token, amt, price, txHash, sig.Reason)
				if trash, err := arena.GenerateTrashTalk(ctx, b.cfg, agentID, result); err == nil && trash != "" {
					result += "\n" + trash
				}
			}
		}
	case "SELL":
		token := sig.Token
		if token == "" {
			token = "PEPE"
		}
		_, holdings, _, _ := b.db.GetPortfolio(agentID)
		amt := holdings[token] / 2
		if amt > 0 {
			txHash := fmt.Sprintf("db-sell-%s-%.2f-%d", token, amt, time.Now().UnixMilli())
			if err := b.db.Sell(agentID, token, amt, price, txHash, sig.Reason); err != nil {
				result = fmt.Sprintf("[%s] SELL %s failed: %v", agentID, token, err)
			} else {
				result = fmt.Sprintf("[%s] SELL %s %.2f @ %.6f tx:%s reason:%s", agentID, token, amt, price, txHash, sig.Reason)
			}
		} else {
			result = fmt.Sprintf("[%s] HOLD %s (no holdings) reason:%s", agentID, token, sig.Reason)
		}
	default:
		result = fmt.Sprintf("[%s] HOLD %s reason:%s", agentID, sig.Token, sig.Reason)
	}
	_, _ = b.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   result,
	})
}

// LeaderboardText returns PnL leaderboard (DB if available, else legacy traders).
func (b *Bot) LeaderboardText() string {
	prices := map[string]float64{"So11111111111111111111111111111111111111112": 150, "PEPE": 0.00001, "BTC": 65000, "ETH": 3500, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 1}
	if b.db != nil {
		board, err := b.db.Leaderboard(prices)
		if err == nil {
			var sb strings.Builder
			sb.WriteString("=== LEADERBOARD (SQLite Fake Testnet) ===\n")
			for id, total := range board {
				usd, _, _, _ := b.db.GetPortfolio(id)
				sb.WriteString(fmt.Sprintf("%-12s USD:%.2f Total:%.2f PnL:%+.2f\n", id, usd, total, total-100))
			}
			return sb.String()
		}
	}
	var sb strings.Builder
	sb.WriteString("=== LEADERBOARD (Simulation) ===\n")
	for id, tr := range b.traders {
		p := tr.GetPortfolio()
		total := p.Value(prices)
		pnl := total - 100
		sb.WriteString(fmt.Sprintf("%-12s USD:%.2f Holdings:%v PnL:%+.2f\n", id, p.USD, p.Holdings, pnl))
	}
	return sb.String()
}

func containsKeyword(text string) bool {
	low := strings.ToLower(text)
	keywords := []string{"btc", "eth", "sol", "pepe", "rugi", "cuan", "scam", "ai", "bot", "trade"}
	for _, k := range keywords {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}
