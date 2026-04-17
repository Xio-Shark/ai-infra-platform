// Package memlimit implements a memory-aware processor that applies backpressure
// when heap usage exceeds configurable watermarks.
//
// V2: Two-level watermark (high = slow-down, critical = reject).
// Polls runtime.MemStats on a configurable interval and updates gateway_backpressure_events_total.
package memlimit

import (
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// Level represents the current memory pressure level.
type Level int32

const (
	LevelNormal   Level = 0
	LevelHigh     Level = 1
	LevelCritical Level = 2
)

func (l Level) String() string {
	switch l {
	case LevelNormal:
		return "normal"
	case LevelHigh:
		return "high"
	case LevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Config holds memory limiter settings.
type Config struct {
	// HighWatermarkMB triggers backpressure (slow path).
	HighWatermarkMB uint64 `yaml:"high_watermark_mb"`
	// CriticalWatermarkMB starts rejecting new data.
	CriticalWatermarkMB uint64 `yaml:"critical_watermark_mb"`
	// CheckInterval controls polling frequency of runtime.MemStats.
	CheckInterval time.Duration `yaml:"check_interval"`
}

// DefaultConfig returns sensible defaults for a 512MB container.
func DefaultConfig() Config {
	return Config{
		HighWatermarkMB:     384, // 75% of 512MB
		CriticalWatermarkMB: 460, // 90% of 512MB
		CheckInterval:       time.Second,
	}
}

// Processor tracks heap usage and rejects payloads when memory is critically high.
type Processor struct {
	cfg           Config
	level         atomic.Int32
	stopCh        chan struct{}
	onLevelChange func(Level) // optional callback when level transitions
}

// New creates a memory limiter processor and starts the background monitor.
// onLevelChange is called whenever the level transitions (may be nil).
func New(cfg Config, onLevelChange func(Level)) *Processor {
	p := &Processor{
		cfg:           cfg,
		stopCh:        make(chan struct{}),
		onLevelChange: onLevelChange,
	}
	go p.monitor()
	log.Printf("[memlimit] high=%dMB critical=%dMB interval=%s",
		cfg.HighWatermarkMB, cfg.CriticalWatermarkMB, cfg.CheckInterval)
	return p
}

func (p *Processor) Name() string { return "memlimit" }

// Level returns the current memory pressure level.
func (p *Processor) Level() Level {
	return Level(p.level.Load())
}

// ProcessTraces rejects traces when memory is critically high.
func (p *Processor) ProcessTraces(payload *model.TracePayload) (*model.TracePayload, error) {
	if Level(p.level.Load()) == LevelCritical {
		metrics.BackpressureEventsTotal.Inc()
		metrics.RejectedSpansTotal.Add(float64(payload.SpanCount))
		return nil, fmt.Errorf("memlimit: heap above critical watermark, rejecting %d spans", payload.SpanCount)
	}
	return payload, nil
}

// ProcessMetrics rejects metrics when memory is critically high.
func (p *Processor) ProcessMetrics(payload *model.MetricPayload) (*model.MetricPayload, error) {
	if Level(p.level.Load()) == LevelCritical {
		metrics.BackpressureEventsTotal.Inc()
		return nil, fmt.Errorf("memlimit: heap above critical watermark, rejecting %d datapoints", payload.DataPointCount)
	}
	return payload, nil
}

// Shutdown stops the background memory monitor.
func (p *Processor) Shutdown() {
	close(p.stopCh)
}

func (p *Processor) monitor() {
	ticker := time.NewTicker(p.cfg.CheckInterval)
	defer ticker.Stop()

	highBytes := p.cfg.HighWatermarkMB * 1024 * 1024
	critBytes := p.cfg.CriticalWatermarkMB * 1024 * 1024

	var stats runtime.MemStats
	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&stats)
			heapMB := stats.HeapAlloc / (1024 * 1024)

			var newLevel Level
			switch {
			case stats.HeapAlloc >= critBytes:
				newLevel = LevelCritical
			case stats.HeapAlloc >= highBytes:
				newLevel = LevelHigh
			default:
				newLevel = LevelNormal
			}

			old := Level(p.level.Swap(int32(newLevel)))
			if old != newLevel {
				log.Printf("[memlimit] level changed %s -> %s (heap=%dMB)", old, newLevel, heapMB)
				metrics.DegradeMode.Set(float64(newLevel))
				if p.onLevelChange != nil {
					p.onLevelChange(newLevel)
				}
			}

		case <-p.stopCh:
			return
		}
	}
}
