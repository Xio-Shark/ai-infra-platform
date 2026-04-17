// Package ratelimit implements a token-bucket rate limiter for the gateway pipeline.
//
// V2: Per-signal rate limiting with configurable burst.
// When the bucket is empty, payloads are rejected and gateway_rejected_spans_total is incremented.
package ratelimit

import (
	"fmt"
	"log"
	"sync"
	"time"

	"ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// Config holds rate limiter settings.
type Config struct {
	// TraceRatePerSec is the sustained trace throughput (spans/sec).
	TraceRatePerSec float64 `yaml:"trace_rate_per_sec"`
	// MetricRatePerSec is the sustained metric throughput (datapoints/sec).
	MetricRatePerSec float64 `yaml:"metric_rate_per_sec"`
	// BurstMultiplier allows short bursts above the sustained rate.
	BurstMultiplier float64 `yaml:"burst_multiplier"`
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
	return Config{
		TraceRatePerSec:  10000,
		MetricRatePerSec: 50000,
		BurstMultiplier:  2.0,
	}
}

// tokenBucket implements the token bucket algorithm.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	maxTokens float64
	rate     float64 // tokens per second
	lastTime time.Time
}

func newBucket(ratePerSec, burstMultiplier float64) *tokenBucket {
	max := ratePerSec * burstMultiplier
	return &tokenBucket{
		tokens:    max,
		maxTokens: max,
		rate:      ratePerSec,
		lastTime:  time.Now(),
	}
}

// tryConsume attempts to consume n tokens. Returns true if allowed.
func (b *tokenBucket) tryConsume(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	// Refill tokens
	b.tokens += elapsed * b.rate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}

	cost := float64(n)
	if b.tokens < cost {
		return false
	}
	b.tokens -= cost
	return true
}

// Processor applies rate limiting to both trace and metric pipelines.
type Processor struct {
	traceBucket  *tokenBucket
	metricBucket *tokenBucket
}

// New creates a rate limiter processor.
func New(cfg Config) *Processor {
	p := &Processor{
		traceBucket:  newBucket(cfg.TraceRatePerSec, cfg.BurstMultiplier),
		metricBucket: newBucket(cfg.MetricRatePerSec, cfg.BurstMultiplier),
	}
	log.Printf("[ratelimit] trace=%.0f/s metric=%.0f/s burst=%.1fx",
		cfg.TraceRatePerSec, cfg.MetricRatePerSec, cfg.BurstMultiplier)
	return p
}

// UpdateConfig dynamically updates rate limit parameters (SIGHUP hot reload).
// Creates new token buckets with updated rates; in-flight tokens are reset.
func (p *Processor) UpdateConfig(cfg Config) {
	p.traceBucket = newBucket(cfg.TraceRatePerSec, cfg.BurstMultiplier)
	p.metricBucket = newBucket(cfg.MetricRatePerSec, cfg.BurstMultiplier)
	log.Printf("[ratelimit] config updated: trace=%.0f/s metric=%.0f/s burst=%.1fx",
		cfg.TraceRatePerSec, cfg.MetricRatePerSec, cfg.BurstMultiplier)
}

func (p *Processor) Name() string { return "ratelimit" }

// ProcessTraces rejects traces when the rate limit is exceeded.
func (p *Processor) ProcessTraces(payload *model.TracePayload) (*model.TracePayload, error) {
	if !p.traceBucket.tryConsume(payload.SpanCount) {
		metrics.RejectedSpansTotal.Add(float64(payload.SpanCount))
		return nil, fmt.Errorf("ratelimit: trace rate exceeded, rejecting %d spans", payload.SpanCount)
	}
	return payload, nil
}

// ProcessMetrics rejects metrics when the rate limit is exceeded.
func (p *Processor) ProcessMetrics(payload *model.MetricPayload) (*model.MetricPayload, error) {
	if !p.metricBucket.tryConsume(payload.DataPointCount) {
		return nil, fmt.Errorf("ratelimit: metric rate exceeded, rejecting %d datapoints", payload.DataPointCount)
	}
	return payload, nil
}
