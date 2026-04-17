// Package retry implements exponential backoff retry for gateway exporters.
//
// V2: Wraps any TraceExporter/MetricExporter with retry logic.
// Uses exponential backoff + jitter to avoid thundering herd on backend recovery.
package retry

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// Config holds retry settings.
type Config struct {
	MaxAttempts   int           `yaml:"max_attempts"`
	InitialDelay  time.Duration `yaml:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	JitterFraction float64      `yaml:"jitter_fraction"` // 0.0-1.0
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:    5,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		JitterFraction: 0.2,
	}
}

// backoff calculates the delay for the given attempt with jitter.
func backoff(cfg Config, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(2, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	// Add jitter
	jitter := delay * cfg.JitterFraction * (rand.Float64()*2 - 1) // [-jitter, +jitter]
	delay += jitter
	if delay < 0 {
		delay = float64(cfg.InitialDelay)
	}
	return time.Duration(delay)
}

// TraceExporter wraps a TraceExporter with retry logic.
type TraceExporter struct {
	inner model.TraceExporter
	cfg   Config
	name  string
}

// NewTraceExporter creates a retry-wrapped trace exporter.
func NewTraceExporter(inner model.TraceExporter, name string, cfg Config) *TraceExporter {
	log.Printf("[retry/%s] max_attempts=%d initial_delay=%s max_delay=%s",
		name, cfg.MaxAttempts, cfg.InitialDelay, cfg.MaxDelay)
	return &TraceExporter{inner: inner, cfg: cfg, name: name}
}

// ExportTraces attempts export with retries on failure.
func (r *TraceExporter) ExportTraces(payload *model.TracePayload) error {
	var lastErr error
	for attempt := 0; attempt < r.cfg.MaxAttempts; attempt++ {
		metrics.ExportAttemptTotal.WithLabelValues(r.name).Inc()

		if err := r.inner.ExportTraces(payload); err != nil {
			lastErr = err
			metrics.ExportRetryTotal.WithLabelValues(r.name).Inc()

			delay := backoff(r.cfg, attempt)
			log.Printf("[retry/%s] attempt %d/%d failed: %v, retrying in %s",
				r.name, attempt+1, r.cfg.MaxAttempts, err, delay)
			time.Sleep(delay)
			continue
		}
		return nil
	}
	return fmt.Errorf("retry/%s: exhausted %d attempts, last error: %w",
		r.name, r.cfg.MaxAttempts, lastErr)
}

// Shutdown delegates to the inner exporter.
func (r *TraceExporter) Shutdown() error {
	return r.inner.Shutdown()
}

// MetricExporter wraps a MetricExporter with retry logic.
type MetricExporter struct {
	inner model.MetricExporter
	cfg   Config
	name  string
}

// NewMetricExporter creates a retry-wrapped metric exporter.
func NewMetricExporter(inner model.MetricExporter, name string, cfg Config) *MetricExporter {
	log.Printf("[retry/%s] max_attempts=%d initial_delay=%s max_delay=%s",
		name, cfg.MaxAttempts, cfg.InitialDelay, cfg.MaxDelay)
	return &MetricExporter{inner: inner, cfg: cfg, name: name}
}

// ExportMetrics attempts export with retries on failure.
func (r *MetricExporter) ExportMetrics(payload *model.MetricPayload) error {
	var lastErr error
	for attempt := 0; attempt < r.cfg.MaxAttempts; attempt++ {
		metrics.ExportAttemptTotal.WithLabelValues(r.name).Inc()

		if err := r.inner.ExportMetrics(payload); err != nil {
			lastErr = err
			metrics.ExportRetryTotal.WithLabelValues(r.name).Inc()

			delay := backoff(r.cfg, attempt)
			log.Printf("[retry/%s] attempt %d/%d failed: %v, retrying in %s",
				r.name, attempt+1, r.cfg.MaxAttempts, err, delay)
			time.Sleep(delay)
			continue
		}
		return nil
	}
	return fmt.Errorf("retry/%s: exhausted %d attempts, last error: %w",
		r.name, r.cfg.MaxAttempts, lastErr)
}

// Shutdown delegates to the inner exporter.
func (r *MetricExporter) Shutdown() error {
	return r.inner.Shutdown()
}

// WithContext is a helper for context-aware retry (cancels on context done).
func WithContext(ctx context.Context, fn func() error, cfg Config) error {
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retry: context cancelled: %w", err)
		}

		if err := fn(); err != nil {
			lastErr = err
			delay := backoff(cfg, attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("retry: context cancelled during backoff: %w", ctx.Err())
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("retry: exhausted %d attempts, last error: %w", cfg.MaxAttempts, lastErr)
}
