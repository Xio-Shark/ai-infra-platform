// Package degrade implements adaptive degradation for the gateway pipeline.
//
// V2: Under high memory/queue pressure, automatically reduce sampling rate
// and prioritize metrics over traces to preserve critical observability data.
package degrade

import (
	"log"
	"math/rand/v2"
	"sync/atomic"

	"ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// Mode represents the degradation level.
type Mode int32

const (
	ModeNormal   Mode = 0 // Full fidelity
	ModeDegraded Mode = 1 // Reduced trace sampling
	ModeCritical Mode = 2 // Traces rejected, only metrics pass
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeDegraded:
		return "degraded"
	case ModeCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Config holds degradation thresholds.
type Config struct {
	// DegradedSamplingRate is applied to traces in degraded mode (0.0-1.0).
	DegradedSamplingRate float64 `yaml:"degraded_sampling_rate"`
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		DegradedSamplingRate: 0.1, // Keep 10% of traces in degraded mode
	}
}

// Processor adapts processing behavior based on the current system pressure.
type Processor struct {
	cfg  Config
	mode atomic.Int32
}

// New creates a degradation processor.
func New(cfg Config) *Processor {
	p := &Processor{cfg: cfg}
	log.Printf("[degrade] degraded_sampling_rate=%.2f", cfg.DegradedSamplingRate)
	return p
}

// UpdateConfig dynamically updates degradation parameters (SIGHUP hot reload).
func (p *Processor) UpdateConfig(cfg Config) {
	p.cfg = cfg
	log.Printf("[degrade] config updated: degraded_sampling_rate=%.2f", cfg.DegradedSamplingRate)
}

func (p *Processor) Name() string { return "degrade" }

// SetMode updates the degradation mode. Typically called by memlimit or a supervisor.
func (p *Processor) SetMode(m Mode) {
	old := Mode(p.mode.Swap(int32(m)))
	if old != m {
		log.Printf("[degrade] mode changed %s -> %s", old, m)
		metrics.DegradeMode.Set(float64(m))
		switch m {
		case ModeNormal:
			metrics.SamplingRatio.Set(1.0)
		case ModeDegraded:
			metrics.SamplingRatio.Set(p.cfg.DegradedSamplingRate)
		case ModeCritical:
			metrics.SamplingRatio.Set(0.0)
		}
	}
}

// Mode returns the current degradation mode.
func (p *Processor) Mode() Mode {
	return Mode(p.mode.Load())
}

// ProcessTraces applies degradation rules to traces.
func (p *Processor) ProcessTraces(payload *model.TracePayload) (*model.TracePayload, error) {
	switch Mode(p.mode.Load()) {
	case ModeCritical:
		// Reject all traces in critical mode
		metrics.RejectedSpansTotal.Add(float64(payload.SpanCount))
		return nil, nil // silently drop
	case ModeDegraded:
		// Probabilistic sampling in degraded mode
		if rand.Float64() >= p.cfg.DegradedSamplingRate {
			metrics.RejectedSpansTotal.Add(float64(payload.SpanCount))
			return nil, nil
		}
	}
	return payload, nil
}

// ProcessMetrics always passes metrics through — metrics are higher priority.
func (p *Processor) ProcessMetrics(payload *model.MetricPayload) (*model.MetricPayload, error) {
	return payload, nil
}
