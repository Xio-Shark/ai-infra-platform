// Package sampler implements probabilistic head-based sampling for traces.
package sampler

import (
	"hash/fnv"
	"log"
	"sync/atomic"

	"ai-infra-platform/internal/otelgateway/model"
)

// Processor performs probabilistic head-based sampling on trace payloads.
// The sampling rate can be updated at runtime via SetRate (hot reload).
type Processor struct {
	rate atomic.Value // stores float64, 0.0 ~ 1.0
}

// New creates a sampler with the given sampling rate.
// rate=1.0 means keep all; rate=0.0 means drop all.
func New(rate float64) *Processor {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	log.Printf("[sampler] probabilistic head-based sampling rate=%.4f", rate)
	p := &Processor{}
	p.rate.Store(rate)
	return p
}

// SetRate dynamically updates the sampling rate (SIGHUP hot reload).
func (p *Processor) SetRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	old := p.rate.Load().(float64)
	p.rate.Store(rate)
	if old != rate {
		log.Printf("[sampler] rate updated %.4f -> %.4f", old, rate)
	}
}

// Rate returns the current sampling rate.
func (p *Processor) Rate() float64 {
	return p.rate.Load().(float64)
}

func (p *Processor) Name() string { return "sampler" }

// ProcessTraces filters spans based on probabilistic sampling.
// Sampling decision is per-trace (based on trace ID hash).
func (p *Processor) ProcessTraces(payload *model.TracePayload) (*model.TracePayload, error) {
	if payload == nil || payload.Data == nil {
		return nil, nil
	}

	rate := p.rate.Load().(float64)

	if rate >= 1.0 {
		return payload, nil
	}
	if rate <= 0.0 {
		return nil, nil // drop all
	}

	// Filter resource spans — keep only those where trace ID hash passes the threshold
	threshold := uint64(rate * float64(1<<63))
	kept := payload.Data.ResourceSpans[:0]

	for _, rs := range payload.Data.ResourceSpans {
		keptScopes := rs.ScopeSpans[:0]
		for _, ss := range rs.ScopeSpans {
			keptSpans := ss.Spans[:0]
			for _, span := range ss.Spans {
				if shouldSample(span.TraceId, threshold) {
					keptSpans = append(keptSpans, span)
				}
			}
			if len(keptSpans) > 0 {
				ss.Spans = keptSpans
				keptScopes = append(keptScopes, ss)
			}
		}
		if len(keptScopes) > 0 {
			rs.ScopeSpans = keptScopes
			kept = append(kept, rs)
		}
	}

	if len(kept) == 0 {
		return nil, nil
	}
	payload.Data.ResourceSpans = kept

	// Recount spans
	count := 0
	for _, rs := range payload.Data.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			count += len(ss.Spans)
		}
	}
	payload.SpanCount = count
	return payload, nil
}

// shouldSample hashes the trace ID with FNV-1a for uniform distribution,
// then compares against the threshold for deterministic sampling.
func shouldSample(traceID []byte, threshold uint64) bool {
	if len(traceID) == 0 {
		return true // invalid trace ID, keep it
	}
	h := fnv.New64a()
	_, _ = h.Write(traceID)
	hash := h.Sum64()
	return (hash >> 1) < threshold // shift to make unsigned comparison safe
}
