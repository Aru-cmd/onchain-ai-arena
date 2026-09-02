package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/arena"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/db"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/llm"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/telegram"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/trading"
)

var (
	cfgPath string
)

func main() {
	root := &cobra.Command{
		Use:   "arena",
		Short: "On-Chain AI Agent Trading Arena - AI vs AI tanding trading",
		Long: `On-Chain AI Agent Trading Arena
3 AI Agent (Konservatif/Degen/FOMO) tanding trading di DEX.
Support simulation (Rp 0) & on-chain testnet (Solana/EVM).`,
	}

	root.PersistentFlags().StringVarP(&cfgPath, "config", "c", "config/config.json", "path to config file")

	root.AddCommand(cmdRun())
	root.AddCommand(cmdTelegram())
	root.AddCommand(cmdChat())
	root.AddCommand(cmdLeaderboard())
	root.AddCommand(cmdVersion())

	if err := root.Execute(); err != nil {
		log.Fatal().Err(err).Msg("arena failed")
	}
}

func loadConfig() *config.Config {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfgPath).Msg("failed to read config")
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatal().Err(err).Msg("invalid config json")
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("config validation failed")
	}
	return &cfg
}

func cmdRun() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run arena orchestrator (poll market + simulate trading + LLM wiring)",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			registry := arena.NewAgentRegistry(cfg)
			roastMgr := roast.NewManager(roast.Config{
				GlobalCooldownMinutes: cfg.Roast.GlobalCooldownMinutes,
				TTLHoursMin:           cfg.Roast.UserTTLHoursMin,
				TTLHoursMax:           cfg.Roast.UserTTLHoursMax,
				RandomChance:          cfg.Roast.RandomChance,
			})
			watcher := trading.NewMarketWatcher(cfg.Chain.JupiterAPI, cfg.Chain.DexScreenerAPI, time.Duration(cfg.Market.PollIntervalSeconds)*time.Second)

			fmt.Printf("Arena started with %d agents: %v\n", len(registry.ListAgentIDs()), registry.ListAgentIDs())
			for _, id := range registry.ListAgentIDs() {
				ag, _ := registry.GetAgent(id)
				fmt.Printf(" - %s (%s) persona=%s strategy=%s model=%s\n", ag.ID, ag.Name, ag.Persona, ag.Strategy.Type, ag.Model)
			}
			fmt.Printf("Mode: %s | Chain: %s | Simulation: %v\n", cfg.Chain.Mode, cfg.Chain.Active, cfg.Market.EnableSimulation)
			fmt.Printf("LLM providers: %d (openai/openrouter/groq/aistudio via openai-compatible)\n", len(cfg.GetModelList()))
			for _, m := range cfg.GetModelList() {
				fmt.Printf(" - %s -> %s (%s)\n", m.ModelName, m.Model, m.APIBase)
			}
			fmt.Println("Watcher + LLM wiring ready. Demo single tick (no loop) ...")

			// Demo single tick: each trader agent decides via LLM (if API key set)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			_ = watcher
			_ = roastMgr
			for _, id := range registry.ListAgentIDs() {
				if id == "orchestrator" {
					continue
				}
				marketData := "BTC: $65000 RSI 28, ETH: $3500 RSI 45, PEPE trending +120% vol 80k, SOL $150"
				sig, err := arena.DecideTrade(ctx, cfg, id, marketData)
				if err != nil {
					fmt.Printf("[%s] LLM decide skipped (no key?): %v\n", id, err)
					continue
				}
				fmt.Printf("[%s] LLM signal: %s %s $%.2f reason:%s conf:%.2f\n", id, sig.Action, sig.Token, sig.AmountUSD, sig.Reason, sig.Confidence)
			}

			// Demo roast wiring
			if should, _ := roastMgr.ShouldRoast("demo-user", false, true); should {
				txt, err := arena.GenerateRoast(ctx, cfg, "degen", "Budi", "btc scam?")
				if err == nil {
					fmt.Printf("[roast demo] %s\n", txt)
				}
			}
			// Show llm client resolution
			if c, err := llm.ResolveModelForAgent(cfg, "degen"); err == nil {
				fmt.Printf("LLM client for degen: model=%s base=%s\n", c.Model, c.Config.APIBase)
			}
			fmt.Println("Use Ctrl+C to stop. (MVP: single tick demo, loop + Telegram next)")
			select {}
		},
	}
}

func cmdChat() *cobra.Command {
	return &cobra.Command{
		Use:   "chat [agent] [message]",
		Short: "Test LLM wiring for agent (telegram arena without Telegram)",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			agentID := args[0]
			msg := "hello"
			if len(args) > 1 {
				msg = args[1]
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()
			client, err := llm.ResolveModelForAgent(cfg, agentID)
			if err != nil {
				log.Fatal().Err(err).Msg("resolve model")
			}
			ag, _ := cfg.GetModelConfig(client.Config.ModelName)
			fmt.Printf("Using provider %s (%s) for agent %s\n", ag.ModelName, ag.APIBase, agentID)
			text, err := llm.Chat(ctx, client, "Kamu adalah "+agentID+" trader AI, jawab singkat 1 baris.", msg, nil, nil)
			if err != nil {
				log.Fatal().Err(err).Msg("chat failed - check API key env (GEMINI_API_KEY/OPENAI_API_KEY etc)")
			}
			fmt.Printf("[%s] %s\n", agentID, text)
		},
	}
}

func cmdLeaderboard() *cobra.Command {
	return &cobra.Command{
		Use:   "leaderboard",
		Short: "Show PnL leaderboard (SQLite fake testnet if DB exists, else simulation)",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			// Try SQLite first (3-bot mode)
			if cfg.DB.GetPath() != "" {
				if db, err := loadDB(cfg); err == nil {
					defer db.Close()
					prices := map[string]float64{"So11111111111111111111111111111111111111112": 150, "PEPE": 0.00001, "BTC": 65000, "ETH": 3500, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 1}
					board, _ := db.Leaderboard(prices)
					fmt.Println("=== LEADERBOARD (SQLite Fake Testnet) ===")
					for id, total := range board {
						usd, _, _, _ := db.GetPortfolio(id)
						fmt.Printf("%-12s USD:%.2f Total:%.2f PnL:%+.2f\n", id, usd, total, total-cfg.DB.GetInitialUSD())
					}
					if len(board) > 0 {
						return
					}
				}
			}
			registry := arena.NewAgentRegistry(cfg)
			fmt.Println("=== LEADERBOARD (Simulation in-memory) ===")
			for _, id := range registry.ListAgentIDs() {
				if id == "orchestrator" {
					continue
				}
				tr := trading.NewSimulatedTrader(cfg.Chain.Active, id, cfg.DB.GetInitialUSD())
				tr.SetPrice("So11111111111111111111111111111111111111112", 150)
				tr.SetPrice("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 1)
				bal, _ := tr.GetBalance(cmd.Context())
				fmt.Printf("%-12s | USD: %6.2f | Holdings: %v\n", id, bal, tr.GetPortfolio().Holdings)
			}
		},
	}
}

func openDB(path string) (*db.DB, error) {
	return db.Open(path)
}

func loadDB(cfg *config.Config) (*db.DB, error) {
	return openDB(cfg.DB.GetPath())
}

func cmdTelegram() *cobra.Command {
	return &cobra.Command{
		Use:   "telegram",
		Short: "Run Telegram arena with 3 bots + SQLite + auto-trading loop",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			// Try 3-bot manager first (tokens per agent in telegram.tokens)
			if len(cfg.Telegram.Tokens) > 0 {
				mgr, err := telegram.NewManager(cfg)
				if err != nil {
					log.Fatal().Err(err).Msg("telegram manager init failed (check telegram.tokens + DB path)")
				}
				fmt.Printf("Telegram arena 3-bot + SQLite: %d bots | DB:%s | providers:%d\n", len(mgr.Bots()), cfg.DB.GetPath(), len(cfg.GetModelList()))
				for id := range mgr.Bots() {
					fmt.Printf(" - bot @%s\n", id)
				}
				fmt.Println(mgr.LeaderboardText())
				ctx := cmd.Context()
				// orchestrator auto-trading loop: 15m degen/fomo, 60m konservatif
				watcher := trading.NewMarketWatcher(cfg.Chain.JupiterAPI, cfg.Chain.DexScreenerAPI, time.Duration(cfg.Market.PollIntervalSeconds)*time.Second)
				go mgr.StartLoops(ctx, watcher)
				fmt.Printf("Auto-trading loop started (konservatif 60m, degen/fomo 15m) broadcasting to %s\n", cfg.Telegram.ChannelID)
				if err := mgr.Start(ctx); err != nil {
					log.Fatal().Err(err).Msg("telegram manager failed")
				}
				<-ctx.Done()
				mgr.Stop()
				return
			}
			// Fallback single bot (legacy)
			token := os.Getenv("TELEGRAM_BOT_TOKEN")
			if token == "" {
				log.Fatal().Msg("TELEGRAM_BOT_TOKEN empty and telegram.tokens empty - set 3 tokens in config telegram.tokens")
			}
			bot, err := telegram.New(cfg, token)
			if err != nil {
				log.Fatal().Err(err).Msg("telegram init failed")
			}
			fmt.Printf("Telegram arena single-bot: %v | providers: %d\n", bot, len(cfg.GetModelList()))
			ctx := cmd.Context()
			if err := bot.Start(ctx); err != nil {
				log.Fatal().Err(err).Msg("telegram start failed")
			}
			<-ctx.Done()
			bot.Stop()
		},
	}
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("onchain-ai-arena v0.6.0-alpha")
		},
	}
}
