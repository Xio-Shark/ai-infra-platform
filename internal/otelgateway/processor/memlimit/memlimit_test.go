package memlimit

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
				ScopeSpans: []*tracepb.ScopeSpans{{Spans: spans}},
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

func TestProcessor_NormalLevel_PassesTraces(t *testing.T) {
	t.Parallel()
	// Use very high watermarks so we stay in normal level
	cfg := Config{
		HighWatermarkMB:     64 * 1024, // 64GB — unreachable in test
		CriticalWatermarkMB: 128 * 1024,
		CheckInterval:       100 * time.Millisecond,
	}
	p := New(cfg, nil)
	defer p.Shutdown()

	// Wait for at least one monitor cycle
	time.Sleep(200 * time.Millisecond)

	result, err := p.ProcessTraces(makeTracePayload(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected trace to pass at normal level")
	}
}

func TestProcessor_NormalLevel_PassesMetrics(t *testing.T) {
	t.Parallel()
	cfg := Config{
		HighWatermarkMB:     64 * 1024,
		CriticalWatermarkMB: 128 * 1024,
		CheckInterval:       100 * time.Millisecond,
	}
	p := New(cfg, nil)
	defer p.Shutdown()

	time.Sleep(200 * time.Millisecond)

	result, err := p.ProcessMetrics(makeMetricPayload(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected metric to pass at normal level")
	}
}

func TestProcessor_CriticalLevel_RejectsTraces(t *testing.T) {
	t.Parallel()
	// Set watermarks to 0 so everything is critical
	cfg := Config{
		HighWatermarkMB:     0,
		CriticalWatermarkMB: 0,
		CheckInterval:       50 * time.Millisecond,
	}
	p := New(cfg, nil)
	defer p.Shutdown()

	// Wait for monitor to set level
	time.Sleep(150 * time.Millisecond)

	if p.Level() != LevelCritical {
		t.Fatalf("expected critical level with 0MB watermark, got %s", p.Level())
	}

	_, err := p.ProcessTraces(makeTracePayload(5))
	if err == nil {
		t.Fatal("expected error at critical level")
	}
}

func TestProcessor_CriticalLevel_RejectsMetrics(t *testing.T) {
	t.Parallel()
	cfg := Config{
		HighWatermarkMB:     0,
		CriticalWatermarkMB: 0,
		CheckInterval:       50 * time.Millisecond,
	}
	p := New(cfg, nil)
	defer p.Shutdown()

	time.Sleep(150 * time.Millisecond)

	_, err := p.ProcessMetrics(makeMetricPayload(10))
	if err == nil {
		t.Fatal("expected error at critical level")
	}
}

func TestProcessor_OnLevelChangeCallback(t *testing.T) {
	t.Parallel()

	var received Level
	called := make(chan struct{}, 1)

	// Watermark=0 guarantees critical level on first check
	cfg := Config{
		HighWatermarkMB:     0,
		CriticalWatermarkMB: 0,
		CheckInterval:       50 * time.Millisecond,
	}
	p := New(cfg, func(l Level) {
		received = l
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer p.Shutdown()

	select {
	case <-called:
		if received != LevelCritical {
			t.Fatalf("expected callback with critical, got %s", received)
		}
	case <-time.After(time.Second):
		t.Fatal("onLevelChange callback not called within 1s")
	}
}

func TestProcessor_LevelMethod(t *testing.T) {
	t.Parallel()
	cfg := Config{
		HighWatermarkMB:     64 * 1024,
		CriticalWatermarkMB: 128 * 1024,
		CheckInterval:       50 * time.Millisecond,
	}
	p := New(cfg, nil)
	defer p.Shutdown()

	time.Sleep(100 * time.Millisecond)

	if p.Level() != LevelNormal {
		t.Fatalf("expected normal level with high watermarks, got %s", p.Level())
	}
}
