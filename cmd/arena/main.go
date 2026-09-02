package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/arena"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
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
		Short: "Run arena orchestrator (poll market + simulate trading)",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			registry := arena.NewAgentRegistry(cfg)
			roastMgr := roast.NewManager(roast.Config{
				GlobalCooldownMinutes: cfg.Roast.GlobalCooldownMinutes,
				TTLHoursMin:           cfg.Roast.UserTTLHoursMin,
				TTLHoursMax:           cfg.Roast.UserTTLHoursMax,
				RandomChance:          cfg.Roast.RandomChance,
			})
			_ = roastMgr
			_ = trading.NewMarketWatcher(cfg.Chain.JupiterAPI, cfg.Chain.DexScreenerAPI, 0)

			fmt.Printf("Arena started with %d agents: %v\n", len(registry.ListAgentIDs()), registry.ListAgentIDs())
			for _, id := range registry.ListAgentIDs() {
				ag, _ := registry.GetAgent(id)
				fmt.Printf(" - %s (%s) persona=%s strategy=%s\n", ag.ID, ag.Name, ag.Persona, ag.Strategy.Type)
			}
			fmt.Println("Mode:", cfg.Chain.Mode, "| Chain:", cfg.Chain.Active, "| Simulation:", cfg.Market.EnableSimulation)
			fmt.Println("Use Ctrl+C to stop. (MVP: simulation only, on-chain wiring next)")
			select {}
		},
	}
}

func cmdLeaderboard() *cobra.Command {
	return &cobra.Command{
		Use:   "leaderboard",
		Short: "Show PnL leaderboard (simulated)",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			registry := arena.NewAgentRegistry(cfg)
			// demo: create simulated traders per agent
			fmt.Println("=== LEADERBOARD (Simulation) ===")
			for _, id := range registry.ListAgentIDs() {
				tr := trading.NewSimulatedTrader(cfg.Chain.Active, id, 100)
				// mock prices for demo
				tr.SetPrice("So11111111111111111111111111111111111111112", 150)
				tr.SetPrice("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 1) // USDC
				bal, _ := tr.GetBalance(cmd.Context())
				fmt.Printf("%-12s | USD: %6.2f | Holdings: %v\n", id, bal, tr.GetPortfolio().Holdings)
			}
		},
	}
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("onchain-ai-arena v0.1.0 (alpha)")
		},
	}
}
