package degrade

import (
	"testing"

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
				ScopeSpans: []*tracepb.ScopeSpans{{Spans: spans}},
			}},
		},
		SpanCount: spanCount,
	}
}

func makeMetricPayload() *model.MetricPayload {
	return &model.MetricPayload{
		Data: &colmetricspb.ExportMetricsServiceRequest{
			ResourceMetrics: []*metricspb.ResourceMetrics{{}},
		},
		DataPointCount: 10,
	}
}

func TestProcessor_NormalMode_PassesAll(t *testing.T) {
	t.Parallel()
	p := New(Config{DegradedSamplingRate: 0.1})

	result, err := p.ProcessTraces(makeTracePayload(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected trace to pass in normal mode")
	}

	mResult, err := p.ProcessMetrics(makeMetricPayload())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mResult == nil {
		t.Fatal("expected metric to pass in normal mode")
	}
}

func TestProcessor_CriticalMode_RejectsTraces(t *testing.T) {
	t.Parallel()
	p := New(Config{DegradedSamplingRate: 0.1})
	p.SetMode(ModeCritical)

	result, _ := p.ProcessTraces(makeTracePayload(5))
	if result != nil {
		t.Fatal("expected trace to be rejected in critical mode")
	}
}

func TestProcessor_CriticalMode_PassesMetrics(t *testing.T) {
	t.Parallel()
	p := New(Config{DegradedSamplingRate: 0.1})
	p.SetMode(ModeCritical)

	result, err := p.ProcessMetrics(makeMetricPayload())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected metric to pass even in critical mode")
	}
}

func TestProcessor_DegradedMode_SamplesTraces(t *testing.T) {
	t.Parallel()
	// rate=0.0 means drop all in degraded mode
	p := New(Config{DegradedSamplingRate: 0.0})
	p.SetMode(ModeDegraded)

	dropped := 0
	total := 100
	for i := 0; i < total; i++ {
		result, _ := p.ProcessTraces(makeTracePayload(1))
		if result == nil {
			dropped++
		}
	}

	// With rate=0.0, all should be dropped
	if dropped != total {
		t.Fatalf("expected all %d traces dropped with rate=0.0, got %d dropped", total, dropped)
	}
}

func TestProcessor_DegradedMode_KeepAll(t *testing.T) {
	t.Parallel()
	// rate=1.0 means keep all even in degraded mode
	p := New(Config{DegradedSamplingRate: 1.0})
	p.SetMode(ModeDegraded)

	kept := 0
	total := 100
	for i := 0; i < total; i++ {
		result, _ := p.ProcessTraces(makeTracePayload(1))
		if result != nil {
			kept++
		}
	}

	if kept != total {
		t.Fatalf("expected all %d traces kept with rate=1.0, got %d kept", total, kept)
	}
}

func TestProcessor_SetMode_Transitions(t *testing.T) {
	t.Parallel()
	p := New(Config{DegradedSamplingRate: 0.5})

	if p.Mode() != ModeNormal {
		t.Fatalf("expected initial mode normal, got %s", p.Mode())
	}

	p.SetMode(ModeDegraded)
	if p.Mode() != ModeDegraded {
		t.Fatalf("expected degraded, got %s", p.Mode())
	}

	p.SetMode(ModeCritical)
	if p.Mode() != ModeCritical {
		t.Fatalf("expected critical, got %s", p.Mode())
	}

	p.SetMode(ModeNormal)
	if p.Mode() != ModeNormal {
		t.Fatalf("expected normal, got %s", p.Mode())
	}
}
