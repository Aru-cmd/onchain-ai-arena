package arena

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

// AgentInstance represents a fully configured trading agent with its own
// workspace, strategy persona, and subagent permissions.
type AgentInstance struct {
	ID        string
	Name      string
	Persona   string // konservatif, degen, fomo, orchestrator
	Workspace string
	Model     string
	Fallbacks []string

	Strategy  config.StrategyConfig
	Subagents *config.SubagentsConfig

	// Trading specific
	InitialBalance float64 // for simulation
}

// NewAgentInstance creates agent instance from config.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
) *AgentInstance {
	workspace := resolveWorkspace(agentCfg, defaults)
	_ = os.MkdirAll(workspace, 0o755)

	model := resolveModel(agentCfg, defaults)
	fallbacks := resolveFallbacks(agentCfg, defaults)

	id := NormalizeAgentID(agentCfg.ID)
	name := agentCfg.Name
	if name == "" {
		name = id
	}
	persona := agentCfg.Persona
	if persona == "" {
		persona = agentCfg.Strategy.Type
	}

	return &AgentInstance{
		ID:        id,
		Name:      name,
		Persona:   persona,
		Workspace: workspace,
		Model:     model,
		Fallbacks: fallbacks,
		Strategy:  agentCfg.Strategy,
		Subagents: agentCfg.Subagents,
	}
}

func resolveWorkspace(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && strings.TrimSpace(agentCfg.Workspace) != "" {
		return expandHome(strings.TrimSpace(agentCfg.Workspace))
	}
	if agentCfg == nil || agentCfg.ID == "" || NormalizeAgentID(agentCfg.ID) == "main" {
		if defaults.Workspace != "" {
			return expandHome(defaults.Workspace)
		}
		return "./workspace-main"
	}
	id := NormalizeAgentID(agentCfg.ID)
	base := defaults.Workspace
	if base == "" {
		base = "./workspace"
	}
	base = expandHome(base)
	// workspace-<id> sibling pattern
	return filepath.Join(filepath.Dir(base), "workspace-"+id)
}

func resolveModel(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && agentCfg.Model != nil && strings.TrimSpace(agentCfg.Model.Primary) != "" {
		return strings.TrimSpace(agentCfg.Model.Primary)
	}
	return defaults.GetModelName()
}

func resolveFallbacks(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) []string {
	if agentCfg != nil && agentCfg.Model != nil && agentCfg.Model.Fallbacks != nil {
		return agentCfg.Model.Fallbacks
	}
	return defaults.ModelFallbacks
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	return path
}
