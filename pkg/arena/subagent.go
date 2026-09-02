package arena

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SubagentManager handles async/sync subagent spawning with whitelist enforcement.
type SubagentTask struct {
	ID        string
	Task      string
	Label     string
	AgentID   string
	Status    string // running, completed, failed, canceled
	Result    string
	Created   int64
}

type SubagentManager struct {
	registry *AgentRegistry
	tasks    map[string]*SubagentTask
	mu       sync.RWMutex
	nextID   int
}

func NewSubagentManager(registry *AgentRegistry) *SubagentManager {
	return &SubagentManager{
		registry: registry,
		tasks:    make(map[string]*SubagentTask),
		nextID:   1,
	}
}

// Spawn creates async subagent if allowed. Returns taskID or error.
func (sm *SubagentManager) Spawn(
	ctx context.Context,
	parentAgentID, targetAgentID, task, label string,
	exec func(ctx context.Context, agent *AgentInstance, task string) (string, error),
) (string, error) {
	if !sm.registry.CanSpawnSubagent(parentAgentID, targetAgentID) {
		return "", fmt.Errorf("agent %q not allowed to spawn %q (check allow_agents)", parentAgentID, targetAgentID)
	}
	agent, ok := sm.registry.GetAgent(targetAgentID)
	if !ok {
		return "", fmt.Errorf("target agent %q not found", targetAgentID)
	}

	sm.mu.Lock()
	taskID := fmt.Sprintf("subagent-%d", sm.nextID)
	sm.nextID++
	t := &SubagentTask{
		ID:      taskID,
		Task:    task,
		Label:   label,
		AgentID: targetAgentID,
		Status:  "running",
		Created: time.Now().UnixMilli(),
	}
	sm.tasks[taskID] = t
	sm.mu.Unlock()

	go func() {
		result, err := exec(ctx, agent, task)
		sm.mu.Lock()
		defer sm.mu.Unlock()
		if err != nil {
			t.Status = "failed"
			t.Result = err.Error()
			if ctx.Err() != nil {
				t.Status = "canceled"
			}
			return
		}
		t.Status = "completed"
		t.Result = result
	}()

	return taskID, nil
}

// SpawnSync executes subagent synchronously (blocking)
func (sm *SubagentManager) SpawnSync(
	ctx context.Context,
	parentAgentID, targetAgentID, task string,
	exec func(ctx context.Context, agent *AgentInstance, task string) (string, error),
) (string, error) {
	if !sm.registry.CanSpawnSubagent(parentAgentID, targetAgentID) {
		return "", fmt.Errorf("agent %q not allowed to spawn %q", parentAgentID, targetAgentID)
	}
	agent, ok := sm.registry.GetAgent(targetAgentID)
	if !ok {
		return "", fmt.Errorf("target agent %q not found", targetAgentID)
	}
	return exec(ctx, agent, task)
}

func (sm *SubagentManager) GetTask(taskID string) (*SubagentTask, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	t, ok := sm.tasks[taskID]
	return t, ok
}

func (sm *SubagentManager) ListTasks() []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*SubagentTask, 0, len(sm.tasks))
	for _, t := range sm.tasks {
		out = append(out, t)
	}
	return out
}
