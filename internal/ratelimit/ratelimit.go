package ratelimit

import (
	"sync"

	ratelib "golang.org/x/time/rate"
)

// Limiter manages a collection of token bucket rate limiters.
type Limiter struct {
	// mu protects the limiters map.
	mu sync.RWMutex
	// limiters stores rate.Limiter instances, keyed by a string identifier.
	limiters map[string]*ratelib.Limiter
}

// Config defines the parameters for a token bucket rate limiter.
type Config struct {
	// RequestsPerSecond is the average number of requests per second allowed.
	RequestsPerSecond float64
	// Burst is the maximum number of requests that can exceed the rate limit instantaneously.
	Burst int
}

// NewLimiter creates and returns a new Limiter.
func NewLimiter() *Limiter {
	return &Limiter{
		limiters: make(map[string]*ratelib.Limiter),
	}
}

// Allow checks if a request is allowed for the given key, updating the limiter's
// configuration (rps/burst) if it has changed.
func (l *Limiter) Allow(key string, rps float64, burst int) bool {
	l.mu.RLock()
	lim, ok := l.limiters[key]
	l.mu.RUnlock()

	if !ok {
		l.mu.Lock()
		// Double-check
		lim, ok = l.limiters[key]
		if !ok {
			lim = ratelib.NewLimiter(ratelib.Limit(rps), burst)
			l.limiters[key] = lim
		}
		l.mu.Unlock()
	}

	// Update limit if changed (e.g. hot reload)
	// Note: checking float equality with == is usually bad, but here we want exact config match.
	// ratelib.Limit is float64.
	if lim.Limit() != ratelib.Limit(rps) {
		lim.SetLimit(ratelib.Limit(rps))
	}
	if lim.Burst() != burst {
		lim.SetBurst(burst)
	}

	return lim.Allow()
}

// Remove removes the limiter for the given key.
// Useful for cleanup if needed.
func (l *Limiter) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.limiters, key)
}

// Prune could be added to remove unused limiters, but requires tracking last access time.
// For now, we assume the set of routes is relatively bounded.
