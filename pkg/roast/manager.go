package roast

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Manager handles human roast logic with TTL 6-12 jam + global cooldown + probability.
// Adapted from requirement: if human joins channel, bot can roast with TTL.
// No Redis required for MVP - in-memory with expiry. Swap to Redis later.

type Manager struct {
	mu sync.RWMutex

	// per-user cooldown expiry
	userCooldown map[string]time.Time
	// global cooldown expiry
	globalCooldown time.Time

	// config
	globalCooldownDur time.Duration
	ttlMin            time.Duration
	ttlMax            time.Duration
	randomChance      float64 // 0.15 = 15%
	mentionOnly       bool

	rnd *rand.Rand
}

type Config struct {
	GlobalCooldownMinutes int
	TTLHoursMin           int
	TTLHoursMax           int
	RandomChance          float64
	MentionOnly           bool
}

func NewManager(cfg Config) *Manager {
	if cfg.GlobalCooldownMinutes == 0 {
		cfg.GlobalCooldownMinutes = 30
	}
	if cfg.TTLHoursMin == 0 {
		cfg.TTLHoursMin = 6
	}
	if cfg.TTLHoursMax == 0 {
		cfg.TTLHoursMax = 12
	}
	if cfg.RandomChance == 0 {
		cfg.RandomChance = 0.15
	}
	return &Manager{
		userCooldown:      make(map[string]time.Time),
		globalCooldownDur: time.Duration(cfg.GlobalCooldownMinutes) * time.Minute,
		ttlMin:            time.Duration(cfg.TTLHoursMin) * time.Hour,
		ttlMax:            time.Duration(cfg.TTLHoursMax) * time.Hour,
		randomChance:      cfg.RandomChance,
		mentionOnly:       cfg.MentionOnly,
		rnd:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldRoast decides if bot should roast human message.
// Returns (shouldRoast, chosenAgentID, reason)
func (m *Manager) ShouldRoast(userID string, mentioned bool, hasKeyword bool) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 1. Global cooldown check
	if now.Before(m.globalCooldown) {
		return false, ""
	}
	// 2. Per-user TTL check
	if exp, ok := m.userCooldown[userID]; ok && now.Before(exp) {
		return false, ""
	}
	// 3. MentionOnly filter
	if m.mentionOnly && !mentioned {
		return false, ""
	}
	// 4. Probability + keyword logic
	should := false
	if mentioned {
		should = true // 100% if mentioned
	} else if hasKeyword {
		should = m.rnd.Float64() < m.randomChance*2 // boost if keyword
	} else {
		should = m.rnd.Float64() < m.randomChance
	}
	if !should {
		return false, ""
	}
	// 5. Set cooldowns
	ttl := m.randomTTL()
	m.userCooldown[userID] = now.Add(ttl)
	m.globalCooldown = now.Add(m.globalCooldownDur)

	// cleanup expired
	for k, exp := range m.userCooldown {
		if now.After(exp) {
			delete(m.userCooldown, k)
		}
	}
	return true, ""
}

func (m *Manager) randomTTL() time.Duration {
	if m.ttlMin == m.ttlMax {
		return m.ttlMin
	}
	delta := m.ttlMax - m.ttlMin
	return m.ttlMin + time.Duration(m.rnd.Int63n(int64(delta)))
}

// ForceCooldown sets cooldown for user (for testing or manual)
func (m *Manager) ForceCooldown(userID string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userCooldown[userID] = time.Now().Add(d)
}

// IsOnCooldown checks if user is on cooldown
func (m *Manager) IsOnCooldown(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, ok := m.userCooldown[userID]
	return ok && time.Now().Before(exp)
}

// Reset clears all cooldowns
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userCooldown = make(map[string]time.Time)
	m.globalCooldown = time.Time{}
}

// PersonaRoast generates roast prompt for LLM based on persona.
// This is template, LLM call is outside manager to keep testable.
func PersonaRoast(persona, humanName, humanMsg string) string {
	templates := map[string]string{
		"konservatif": "Manusia %s berisik: \"%s\". Ejek dengan gaya boomer sinis: trading pakai feeling, kami pakai RSI. 1 baris, pedas lucu, jangan SARA.",
		"degen":       "Woi manusia %s bilang \"%s\". Balas ala degen: suruh all-in memecoin daripada bacot. 1 baris, wkwk style.",
		"fomo":        "Manusia %s telat: \"%s\". Ejek gaya FOMO panikan: kami udah TP, kamu baru entry. 1 baris.",
	}
	tmpl, ok := templates[persona]
	if !ok {
		tmpl = "Ejek manusia %s yang bilang \"%s\" dengan gaya trader AI sombong, 1 baris."
	}
	return fmt.Sprintf(tmpl, humanName, humanMsg)
}

func fmtSprintf(format string, args ...any) string {
	// tiny helper to avoid import fmt in caller
	return fmt.Sprintf(format, args...)
}
