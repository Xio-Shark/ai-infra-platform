package gateway

import (
	"sync"
	"time"
)

// TokenBucket implements a simple token-bucket rate limiter.
type TokenBucket struct {
	rate       float64 // tokens per second
	capacity   float64
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a limiter with the given requests-per-second.
func NewTokenBucket(rps int) *TokenBucket {
	cap := float64(rps)
	if cap < 1 {
		cap = 1
	}
	return &TokenBucket{
		rate:       cap,
		capacity:   cap * 2, // burst = 2x steady state
		tokens:     cap,
		lastRefill: time.Now(),
	}
}

// Allow returns true if a request is permitted, consuming one token.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now
}

// RateLimiterRegistry manages per-model rate limiters.
type RateLimiterRegistry struct {
	mu         sync.RWMutex
	limiters   map[string]*TokenBucket
	defaultRPS int
	modelRPS   map[string]int
}

// NewRateLimiterRegistry builds the registry from config.
func NewRateLimiterRegistry(defaultRPS int, modelRPS map[string]int) *RateLimiterRegistry {
	return &RateLimiterRegistry{
		limiters:   make(map[string]*TokenBucket),
		defaultRPS: defaultRPS,
		modelRPS:   modelRPS,
	}
}

// Allow checks rate limit for the given model. Thread-safe.
func (r *RateLimiterRegistry) Allow(modelName string) bool {
	r.mu.RLock()
	limiter, ok := r.limiters[modelName]
	r.mu.RUnlock()

	if !ok {
		r.mu.Lock()
		// double check
		if limiter, ok = r.limiters[modelName]; !ok {
			rps := r.defaultRPS
			if custom, found := r.modelRPS[modelName]; found {
				rps = custom
			}
			limiter = NewTokenBucket(rps)
			r.limiters[modelName] = limiter
		}
		r.mu.Unlock()
	}
	return limiter.Allow()
}
