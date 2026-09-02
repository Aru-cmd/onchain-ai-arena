package arena

import (
	"sync"

	"github.com/Aru-cmd/onchain-ai-arena/pkg/bus"
	"github.com/Aru-cmd/onchain-ai-arena/pkg/config"
)

// AgentRegistry manages multiple agent instances and routes messages.
type AgentRegistry struct {
	agents map[string]*AgentInstance
	cfg    *config.Config
	mu     sync.RWMutex
}

// NewAgentRegistry creates registry from config, instantiating all agents.
func NewAgentRegistry(cfg *config.Config) *AgentRegistry {
	r := &AgentRegistry{
		agents: make(map[string]*AgentInstance),
		cfg:    cfg,
	}
	list := cfg.Agents.List
	if len(list) == 0 {
		implicit := &config.AgentConfig{ID: "main", Default: true}
		inst := NewAgentInstance(implicit, &cfg.Agents.Defaults)
		r.agents["main"] = inst
		return r
	}
	for i := range list {
		ac := &list[i]
		id := NormalizeAgentID(ac.ID)
		inst := NewAgentInstance(ac, &cfg.Agents.Defaults)
		r.agents[id] = inst
	}
	return r
}

func (r *AgentRegistry) GetAgent(agentID string) (*AgentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id := NormalizeAgentID(agentID)
	a, ok := r.agents[id]
	return a, ok
}

func (r *AgentRegistry) ListAgentIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	return ids
}

func (r *AgentRegistry) GetDefaultAgent() *AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.agents["main"]; ok {
		return a
	}
	for _, a := range r.agents {
		return a
	}
	return nil
}

// CanSpawnSubagent checks if parent is allowed to spawn target via whitelist.
func (r *AgentRegistry) CanSpawnSubagent(parentAgentID, targetAgentID string) bool {
	parent, ok := r.GetAgent(parentAgentID)
	if !ok {
		return false
	}
	if parent.Subagents == nil || parent.Subagents.AllowAgents == nil {
		return false
	}
	targetNorm := NormalizeAgentID(targetAgentID)
	for _, allowed := range parent.Subagents.AllowAgents {
		if allowed == "*" {
			return true
		}
		if NormalizeAgentID(allowed) == targetNorm {
			return true
		}
	}
	return false
}

// ResolveRoute determines which agent handles inbound context.
func (r *AgentRegistry) ResolveRoute(inbound bus.InboundContext) string {
	// Check dispatch rules first
	if r.cfg.Agents.Dispatch != nil {
		for _, rule := range r.cfg.Agents.Dispatch.Rules {
			if matchesRule(rule, inbound) {
				return NormalizeAgentID(rule.Agent)
			}
		}
	}
	// Fallback to default agent
	for _, a := range r.cfg.Agents.List {
		if a.Default {
			return NormalizeAgentID(a.ID)
		}
	}
	if len(r.cfg.Agents.List) > 0 {
		return NormalizeAgentID(r.cfg.Agents.List[0].ID)
	}
	return "main"
}

func matchesRule(rule config.DispatchRule, in bus.InboundContext) bool {
	w := rule.When
	if w.Channel != "" && w.Channel != in.Channel {
		return false
	}
	if w.Chat != "" && w.Chat != in.ChatID {
		return false
	}
	if w.Sender != "" && w.Sender != in.SenderID {
		return false
	}
	if w.Mentioned != nil && *w.Mentioned != in.Mentioned {
		return false
	}
	// at least one constraint must exist
	if w.Channel == "" && w.Chat == "" && w.Sender == "" && w.Mentioned == nil {
		return false
	}
	return true
}
