package ratelimit

import (
	"testing"
	"time"

	"ai-infra-platform/internal/otelgateway/model"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func makeTracePayload(spanCount int) *model.TracePayload {
	spans := make([]*tracepb.Span, spanCount)
	for i := range spans {
		spans[i] = &tracepb.Span{Name: "test"}
	}
	return &model.TracePayload{
		Data: &coltracepb.ExportTraceServiceRequest{
			ResourceSpans: []*tracepb.ResourceSpans{{
				ScopeSpans: []*tracepb.ScopeSpans{{
					Spans: spans,
				}},
			}},
		},
		SpanCount: spanCount,
	}
}

func makeMetricPayload(count int) *model.MetricPayload {
	return &model.MetricPayload{
		Data: &colmetricspb.ExportMetricsServiceRequest{
			ResourceMetrics: []*metricspb.ResourceMetrics{{}},
		},
		DataPointCount: count,
	}
}

func TestTokenBucket_AllowsBurst(t *testing.T) {
	t.Parallel()
	b := newBucket(100, 2.0) // 100/s, burst = 200

	// Should allow up to burst (200 tokens available initially)
	if !b.tryConsume(200) {
		t.Fatal("expected burst of 200 to be allowed")
	}
	// Next request should be rejected (bucket empty)
	if b.tryConsume(1) {
		t.Fatal("expected rejection after burst exhausted")
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	t.Parallel()
	b := newBucket(1000, 1.0) // 1000/s, burst = 1000

	// Drain the bucket
	b.tryConsume(1000)
	if b.tryConsume(1) {
		t.Fatal("expected rejection after drain")
	}

	// Wait for refill (100ms = ~100 tokens at 1000/s)
	time.Sleep(150 * time.Millisecond)

	if !b.tryConsume(50) {
		t.Fatal("expected 50 tokens to be available after 150ms refill at 1000/s")
	}
}

func TestProcessor_TraceRateLimit(t *testing.T) {
	t.Parallel()
	cfg := Config{TraceRatePerSec: 100, MetricRatePerSec: 100, BurstMultiplier: 1.0}
	p := New(cfg)

	// First batch within burst should pass
	payload := makeTracePayload(50)
	result, err := p.ProcessTraces(payload)
	if err != nil {
		t.Fatalf("expected first batch to pass, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for first batch")
	}

	// Drain remaining burst
	payload = makeTracePayload(50)
	_, _ = p.ProcessTraces(payload)

	// Next batch should be rejected (bucket empty)
	payload = makeTracePayload(10)
	_, err = p.ProcessTraces(payload)
	if err == nil {
		t.Fatal("expected rate limit rejection")
	}
}

func TestProcessor_MetricRateLimit(t *testing.T) {
	t.Parallel()
	cfg := Config{TraceRatePerSec: 100, MetricRatePerSec: 50, BurstMultiplier: 1.0}
	p := New(cfg)

	// Drain metric bucket
	payload := makeMetricPayload(50)
	_, _ = p.ProcessMetrics(payload)

	// Should be rejected
	payload = makeMetricPayload(10)
	_, err := p.ProcessMetrics(payload)
	if err == nil {
		t.Fatal("expected metric rate limit rejection")
	}
}

func TestProcessor_IndependentBuckets(t *testing.T) {
	t.Parallel()
	cfg := Config{TraceRatePerSec: 50, MetricRatePerSec: 50, BurstMultiplier: 1.0}
	p := New(cfg)

	// Drain trace bucket
	p.ProcessTraces(makeTracePayload(50))

	// Metric bucket should still work
	result, err := p.ProcessMetrics(makeMetricPayload(10))
	if err != nil {
		t.Fatalf("metric bucket should be independent, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil metric result")
	}
}
