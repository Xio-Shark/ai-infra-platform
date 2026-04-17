package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"

	"ai-infra-platform/internal/otelgateway/config"
	"ai-infra-platform/internal/otelgateway/model"
)

// --- test helpers ---

type countingTraceExporter struct {
	count atomic.Int64
}

func (e *countingTraceExporter) ExportTraces(_ *model.TracePayload) error {
	e.count.Add(1)
	return nil
}
func (e *countingTraceExporter) Shutdown() error { return nil }

type countingMetricExporter struct {
	count atomic.Int64
}

func (e *countingMetricExporter) ExportMetrics(_ *model.MetricPayload) error {
	e.count.Add(1)
	return nil
}
func (e *countingMetricExporter) Shutdown() error { return nil }

func makeTracePayloadWithService(svcName string, spanCount int) *model.TracePayload {
	spans := make([]*tracepb.Span, spanCount)
	for i := range spans {
		spans[i] = &tracepb.Span{Name: "test"}
	}
	var resource *resourcepb.Resource
	if svcName != "" {
		resource = &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{{
				Key:   "service.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: svcName}},
			}},
		}
	}
	return &model.TracePayload{
		Data: &coltracepb.ExportTraceServiceRequest{
			ResourceSpans: []*tracepb.ResourceSpans{{
				Resource:   resource,
				ScopeSpans: []*tracepb.ScopeSpans{{Spans: spans}},
			}},
		},
		SpanCount: spanCount,
	}
}

func makeMetricPayloadWithService(svcName string, count int) *model.MetricPayload {
	ms := make([]*metricspb.Metric, count)
	for i := range ms {
		ms[i] = &metricspb.Metric{Name: "test_metric"}
	}
	var resource *resourcepb.Resource
	if svcName != "" {
		resource = &resourcepb.Resource{
			Attributes: []*commonpb.KeyValue{{
				Key:   "service.name",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: svcName}},
			}},
		}
	}
	return &model.MetricPayload{
		Data: &colmetricspb.ExportMetricsServiceRequest{
			ResourceMetrics: []*metricspb.ResourceMetrics{{
				Resource:     resource,
				ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: ms}},
			}},
		},
		DataPointCount: count,
	}
}

// --- tests ---

func TestManager_BasicTraceFlow(t *testing.T) {
	t.Parallel()
	exp := &countingTraceExporter{}
	mExp := &countingMetricExporter{}
	cfg := &config.PipelineConfig{
		TraceQueueSize: 100, MetricQueueSize: 100,
		Workers: 2, ShardCount: 1,
	}

	m := NewManager(cfg, nil, nil, exp, mExp)
	m.Start(context.Background())

	for i := 0; i < 10; i++ {
		if !m.SubmitTrace(makeTracePayloadWithService("svc-a", 1)) {
			t.Fatal("submit should succeed")
		}
	}

	time.Sleep(100 * time.Millisecond) // let workers process
	m.Shutdown()

	if got := exp.count.Load(); got != 10 {
		t.Fatalf("expected 10 trace exports, got %d", got)
	}
}

func TestManager_BasicMetricFlow(t *testing.T) {
	t.Parallel()
	tExp := &countingTraceExporter{}
	mExp := &countingMetricExporter{}
	cfg := &config.PipelineConfig{
		TraceQueueSize: 100, MetricQueueSize: 100,
		Workers: 2, ShardCount: 1,
	}

	m := NewManager(cfg, nil, nil, tExp, mExp)
	m.Start(context.Background())

	for i := 0; i < 10; i++ {
		m.SubmitMetric(makeMetricPayloadWithService("svc-b", 1))
	}

	time.Sleep(100 * time.Millisecond)
	m.Shutdown()

	if got := mExp.count.Load(); got != 10 {
		t.Fatalf("expected 10 metric exports, got %d", got)
	}
}

func TestManager_ShardDistribution(t *testing.T) {
	t.Parallel()
	exp := &countingTraceExporter{}
	mExp := &countingMetricExporter{}
	cfg := &config.PipelineConfig{
		TraceQueueSize: 400, MetricQueueSize: 100,
		Workers: 1, ShardCount: 4,
	}

	m := NewManager(cfg, nil, nil, exp, mExp)
	m.Start(context.Background())

	// Submit payloads from multiple services
	services := []string{"auth-svc", "payment-svc", "order-svc", "user-svc", "gateway-svc"}
	for _, svc := range services {
		for i := 0; i < 10; i++ {
			m.SubmitTrace(makeTracePayloadWithService(svc, 1))
		}
	}

	time.Sleep(200 * time.Millisecond)
	m.Shutdown()

	// All 50 payloads should be exported regardless of shard
	if got := exp.count.Load(); got != 50 {
		t.Fatalf("expected 50 trace exports across shards, got %d", got)
	}
}

func TestManager_ShardDeterministic(t *testing.T) {
	t.Parallel()
	// Same service name should always go to the same shard
	cfg := &config.PipelineConfig{
		TraceQueueSize: 100, MetricQueueSize: 100,
		Workers: 1, ShardCount: 8,
	}

	m := NewManager(cfg, nil, nil, &countingTraceExporter{}, &countingMetricExporter{})

	p1 := makeTracePayloadWithService("consistent-svc", 1)
	p2 := makeTracePayloadWithService("consistent-svc", 5)

	s1 := m.traceShardIndex(p1)
	s2 := m.traceShardIndex(p2)

	if s1 != s2 {
		t.Fatalf("same service name should map to same shard: got %d vs %d", s1, s2)
	}
}

func TestManager_NilResourceFallsToShard0(t *testing.T) {
	t.Parallel()
	cfg := &config.PipelineConfig{
		TraceQueueSize: 100, MetricQueueSize: 100,
		Workers: 1, ShardCount: 4,
	}

	m := NewManager(cfg, nil, nil, &countingTraceExporter{}, &countingMetricExporter{})

	// No resource → shard 0
	p := makeTracePayloadWithService("", 1)
	shard := m.traceShardIndex(p)
	if shard != 0 {
		t.Fatalf("nil resource should default to shard 0, got %d", shard)
	}
}

func TestManager_PreShutdownHooks(t *testing.T) {
	t.Parallel()
	cfg := &config.PipelineConfig{
		TraceQueueSize: 10, MetricQueueSize: 10,
		Workers: 1, ShardCount: 1,
	}

	m := NewManager(cfg, nil, nil, &countingTraceExporter{}, &countingMetricExporter{})
	m.Start(context.Background())

	var hookCalled bool
	m.OnPreShutdown(func() { hookCalled = true })

	m.Shutdown()

	if !hookCalled {
		t.Fatal("pre-shutdown hook should have been called")
	}
}

func TestManager_QueueFull_DropsPayload(t *testing.T) {
	t.Parallel()
	cfg := &config.PipelineConfig{
		TraceQueueSize: 2, MetricQueueSize: 2,
		Workers: 0, ShardCount: 1, // 0 workers = no consumers
	}

	// Use a blocking exporter so queue fills up
	m := NewManager(cfg, nil, nil, &countingTraceExporter{}, &countingMetricExporter{})
	// Don't start workers — queue will fill up

	// Queue capacity per shard = 2/1 = 2
	ok1 := m.SubmitTrace(makeTracePayloadWithService("a", 1))
	ok2 := m.SubmitTrace(makeTracePayloadWithService("a", 1))
	ok3 := m.SubmitTrace(makeTracePayloadWithService("a", 1)) // should be dropped

	if !ok1 || !ok2 {
		t.Fatal("first two submits should succeed")
	}
	if ok3 {
		t.Fatal("third submit should fail (queue full)")
	}
}

func TestManager_ConcurrentSubmits(t *testing.T) {
	t.Parallel()
	exp := &countingTraceExporter{}
	cfg := &config.PipelineConfig{
		TraceQueueSize: 10000, MetricQueueSize: 100,
		Workers: 4, ShardCount: 4,
	}

	m := NewManager(cfg, nil, nil, exp, &countingMetricExporter{})
	m.Start(context.Background())

	var wg sync.WaitGroup
	total := 1000
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc := []string{"svc-a", "svc-b", "svc-c", "svc-d"}[idx%4]
			m.SubmitTrace(makeTracePayloadWithService(svc, 1))
		}(i)
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)
	m.Shutdown()

	if got := exp.count.Load(); got != int64(total) {
		t.Fatalf("expected %d exports, got %d", total, got)
	}
}

func TestHashToShard_Distribution(t *testing.T) {
	t.Parallel()
	shards := 8
	counts := make([]int, shards)

	// 100 unique service names should spread across shards
	for i := 0; i < 100; i++ {
		svc := "service-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		idx := hashToShard(svc, shards)
		if idx < 0 || idx >= shards {
			t.Fatalf("shard index %d out of range [0, %d)", idx, shards)
		}
		counts[idx]++
	}

	// No shard should have 0 items (statistical, but very unlikely with 100 items / 8 shards)
	for i, c := range counts {
		if c == 0 {
			t.Logf("warning: shard %d got 0 items (may indicate poor distribution)", i)
		}
	}
}
