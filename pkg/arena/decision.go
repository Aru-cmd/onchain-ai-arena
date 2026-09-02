package arena

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/llm"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/roast"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/trading"
)

// PersonaPrompts defines system prompts per trading persona.
var PersonaPrompts = map[string]string{
	"konservatif": `Kamu adalah Agent Konservatif (Boomer RSI). Hanya beli BTC/ETH kalau RSI < 30 dan di support MA200. Risk kecil, hold lama. Jawab HANYA JSON: {"action":"BUY/SELL/HOLD","token":"...","amount_usd":10,"reason":"...","confidence":0.0-1.0}`,
	"degen":       `Kamu adalah Agent Degen Memecoin. Hajar koin baru viral di DexScreener/Twitter kalau volume 24h naik >100% dan umur koin <7 hari. High risk high return. Jawab HANYA JSON: {"action":"BUY/SELL/HOLD","token":"...","amount_usd":10,"reason":"...","confidence":0.0-1.0}`,
	"fomo":        `Kamu adalah Agent FOMO Chaser. Suka serobot koin yang lagi pump >20% dalam 1 jam dengan volume spike >2x. Jawab HANYA JSON: {"action":"BUY/SELL/HOLD","token":"...","amount_usd":10,"reason":"...","confidence":0.0-1.0}`,
	"orchestrator": `Kamu adalah Orchestrator Arena. Tugasmu ringkas hasil trading dan buat trash-talk antar agent.`,
}

// DecideTrade calls LLM for agent persona to decide trade.
// marketData is free-form text: prices, RSI, trending, etc.
func DecideTrade(ctx context.Context, cfg *config.Config, agentID string, marketData string) (trading.TradeSignal, error) {
	client, err := llm.ResolveModelForAgent(cfg, agentID)
	if err != nil {
		return trading.TradeSignal{}, err
	}
	agent, ok := findAgent(cfg, agentID)
	if !ok {
		return trading.TradeSignal{}, fmt.Errorf("agent %q not found", agentID)
	}
	persona := strings.ToLower(agent.Persona)
	if persona == "" {
		persona = strings.ToLower(agent.Strategy.Type)
	}
	sys, ok := PersonaPrompts[persona]
	if !ok {
		sys = PersonaPrompts["konservatif"]
	}
	// Temperature from agent defaults or config
	var temp *float64
	if cfg.Agents.Defaults.Temperature != nil {
		temp = cfg.Agents.Defaults.Temperature
	}
	// Build user prompt with market data
	user := fmt.Sprintf("Market data:\n%s\n\nAgent: %s (%s)\nPortfolio: initial $100, holdings unknown (simulation)\nTugas: decide BUY/SELL/HOLD sekarang. Balas JSON saja.", marketData, agent.Name, agent.ID)

	// Use llm.ChatJSON
	var sig trading.TradeSignal
	// Need to bypass temperature via chat helper - use llm.ChatJSON which ignores temp for now
	// For custom temp, use llm.Chat directly then parse
	if temp != nil {
		// use chat with temp
		text, err := llm.Chat(ctx, client, sys, user, temp, nil)
		if err != nil {
			return trading.TradeSignal{}, err
		}
		// parse json
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		if err := parseSignal(text, &sig); err != nil {
			return trading.TradeSignal{}, err
		}
		return sig, nil
	}
	if err := llm.ChatJSON(ctx, client, sys, user, &sig); err != nil {
		return trading.TradeSignal{}, err
	}
	return sig, nil
}

// GenerateRoast calls LLM to roast human or other agent.
func GenerateRoast(ctx context.Context, cfg *config.Config, persona, humanName, humanMsg string) (string, error) {
	// Pick model for persona or fallback to default
	client, err := llm.ResolveModelForAgent(cfg, persona)
	if err != nil {
		// fallback to orchestrator or default
		client, err = llm.NewClientFromConfig(cfg, "")
		if err != nil {
			return "", err
		}
	}
	prompt := roast.PersonaRoast(persona, humanName, humanMsg)
	sys := "Kamu adalah AI trader yang suka nge-roast. Balas 1 baris, pedas tapi lucu, jangan SARA/toxic berat. Pakai bahasa Indonesia santai."
	text, err := llm.Chat(ctx, client, sys, prompt, nil, nil)
	if err != nil {
		return "", err
	}
	// Fallback to template if LLM empty
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf("[%s] %s, diem dulu — ini arena AI 😏", persona, humanName), nil
	}
	return text, nil
}

// GenerateTrashTalk makes agent trash-talk after trade.
func GenerateTrashTalk(ctx context.Context, cfg *config.Config, fromPersona, tradeResult string) (string, error) {
	client, err := llm.ResolveModelForAgent(cfg, fromPersona)
	if err != nil {
		client, err = llm.NewClientFromConfig(cfg, "")
		if err != nil {
			return "", err
		}
	}
	sys := "Kamu adalah trader AI yang baru profit. Ejek agent lain dengan gaya persona kamu, 1 baris, lucu."
	user := fmt.Sprintf("Hasil trade kamu: %s. Buat 1 kalimat trash-talk ke agent lain.", tradeResult)
	return llm.Chat(ctx, client, sys, user, nil, nil)
}

func findAgent(cfg *config.Config, id string) (*config.AgentConfig, bool) {
	norm := NormalizeAgentID(id)
	for i := range cfg.Agents.List {
		if NormalizeAgentID(cfg.Agents.List[i].ID) == norm {
			return &cfg.Agents.List[i], true
		}
	}
	return nil, false
}

func parseSignal(text string, out *trading.TradeSignal) error {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}
	return json.Unmarshal([]byte(text), out)
}
