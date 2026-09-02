package roast

import (
	"testing"
	"time"
)

func TestManager_ShouldRoast_MentionAlways(t *testing.T) {
	m := NewManager(Config{GlobalCooldownMinutes: 30, TTLHoursMin: 6, TTLHoursMax: 6, RandomChance: 0})
	should, _ := m.ShouldRoast("user1", true, false)
	if !should {
		t.Error("mention should always roast when no cooldown")
	}
	// second call immediately should be blocked by global cooldown
	should2, _ := m.ShouldRoast("user2", true, false)
	if should2 {
		t.Error("should be blocked by global cooldown")
	}
}

func TestManager_TTLPerUser(t *testing.T) {
	m := NewManager(Config{GlobalCooldownMinutes: 0, TTLHoursMin: 6, TTLHoursMax: 6, RandomChance: 1.0})
	m.ShouldRoast("budi", false, true)
	if !m.IsOnCooldown("budi") {
		t.Error("budi should be on cooldown after roast")
	}
	if m.IsOnCooldown("andi") {
		t.Error("andi should not be on cooldown")
	}
}

func TestManager_MentionOnly(t *testing.T) {
	m := NewManager(Config{GlobalCooldownMinutes: 0, TTLHoursMin: 6, TTLHoursMax: 6, RandomChance: 1.0, MentionOnly: true})
	should, _ := m.ShouldRoast("u1", false, true)
	if should {
		t.Error("mentionOnly should block non-mention")
	}
	should2, _ := m.ShouldRoast("u1", true, false)
	if !should2 {
		t.Error("mentionOnly should allow mention")
	}
}

func TestManager_Reset(t *testing.T) {
	m := NewManager(Config{GlobalCooldownMinutes: 30, TTLHoursMin: 6, TTLHoursMax: 6, RandomChance: 1.0})
	m.ShouldRoast("x", true, false)
	m.Reset()
	if m.IsOnCooldown("x") {
		t.Error("reset should clear cooldown")
	}
}

func TestManager_ForceCooldown(t *testing.T) {
	m := NewManager(Config{})
	m.ForceCooldown("y", 1*time.Hour)
	if !m.IsOnCooldown("y") {
		t.Error("force cooldown failed")
	}
}

func TestPersonaRoast(t *testing.T) {
	s := PersonaRoast("konservatif", "Budi", "btc to the moon")
	if s == "" {
		t.Error("persona roast empty")
	}
	s2 := PersonaRoast("unknown", "Budi", "hello")
	if s2 == "" {
		t.Error("unknown persona should fallback")
	}
}
