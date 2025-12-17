package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	l := NewLimiter()

	// 1. Basic Allow
	key := "test-route"
	// 1 RPS, Burst 1
	if !l.Allow(key, 1, 1) {
		t.Errorf("expected Allow to return true for initial request")
	}

	// 2. Burst exceeded
	// We just consumed 1. Next one should fail immediately.
	if l.Allow(key, 1, 1) {
		t.Errorf("expected Allow to return false when burst exceeded")
	}

	// 3. Dynamic config update (increase rate)
	// Change to 100 RPS (1 token per 10ms). Burst 5.
	// We call Allow, which updates the config. It will try to consume 1.
	// Since we are at 0 tokens, and just increased rate, we still need to wait for 1 token to generate.
	// 1 token at 100 RPS takes 10ms.

	// Let's just update the config by calling Allow (which will fail, but update state)
	if l.Allow(key, 100, 5) {
		// It MIGHT pass if enough time elapsed between steps, but unlikely in unit test speed.
		// If it passes, that's fine too (tokens generated).
	} else {
		// If it failed, wait 20ms and try again. It SHOULD pass now.
		time.Sleep(20 * time.Millisecond)
		if !l.Allow(key, 100, 5) {
			t.Errorf("expected Allow to return true after increasing rate and waiting")
		}
	}
}

func TestLimiter_DifferentKeys(t *testing.T) {
	l := NewLimiter()

	// Key A
	if !l.Allow("A", 1, 1) {
		t.Error("A should be allowed")
	}
	if l.Allow("A", 1, 1) {
		t.Error("A should be blocked")
	}

	// Key B (independent)
	if !l.Allow("B", 1, 1) {
		t.Error("B should be allowed (independent of A)")
	}
}
