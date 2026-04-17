// Package pipeline manages the trace and metric processing pipelines.
// V2: supports sharding by service name for improved concurrency isolation.
package pipeline

import (
	"context"
	"hash/fnv"
	"log"
	"sync"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"

	"ai-infra-platform/internal/otelgateway/config"
	"ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// Manager orchestrates trace and metric pipelines with sharded queues.
type Manager struct {
	cfg              *config.PipelineConfig
	shardCount       int
	traceShards      []chan *model.TracePayload
	metricShards     []chan *model.MetricPayload
	traceProcs       []model.TraceProcessor
	metricProcs      []model.MetricProcessor
	traceExp         model.TraceExporter
	metricExp        model.MetricExporter
	wg               sync.WaitGroup
	cancel           context.CancelFunc
	preShutdownHooks []func() // executed after workers drain, before exporters close
}

// NewManager creates a pipeline manager with sharded queues for traces and metrics.
func NewManager(
	cfg *config.PipelineConfig,
	traceProcs []model.TraceProcessor,
	metricProcs []model.MetricProcessor,
	traceExp model.TraceExporter,
	metricExp model.MetricExporter,
) *Manager {
	shards := cfg.ShardCount
	if shards < 1 {
		shards = 1
	}

	// Divide total queue capacity across shards
	traceQPerShard := cfg.TraceQueueSize / shards
	if traceQPerShard < 1 {
		traceQPerShard = 1
	}
	metricQPerShard := cfg.MetricQueueSize / shards
	if metricQPerShard < 1 {
		metricQPerShard = 1
	}

	traceShards := make([]chan *model.TracePayload, shards)
	metricShards := make([]chan *model.MetricPayload, shards)
	for i := 0; i < shards; i++ {
		traceShards[i] = make(chan *model.TracePayload, traceQPerShard)
		metricShards[i] = make(chan *model.MetricPayload, metricQPerShard)
	}

	return &Manager{
		cfg:          cfg,
		shardCount:   shards,
		traceShards:  traceShards,
		metricShards: metricShards,
		traceProcs:   traceProcs,
		metricProcs:  metricProcs,
		traceExp:     traceExp,
		metricExp:    metricExp,
	}
}

// Start launches worker goroutines for all shards.
func (m *Manager) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)

	// Set capacity gauges (total across all shards)
	metrics.QueueCapacity.WithLabelValues("trace").Set(float64(m.cfg.TraceQueueSize))
	metrics.QueueCapacity.WithLabelValues("metric").Set(float64(m.cfg.MetricQueueSize))

	workers := m.cfg.Workers
	if workers < 1 {
		workers = 1
	}

	// Each shard gets `workers` goroutines per signal type
	for shard := 0; shard < m.shardCount; shard++ {
		for w := 0; w < workers; w++ {
			m.wg.Add(2)
			go m.traceWorker(ctx, shard)
			go m.metricWorker(ctx, shard)
		}
	}

	log.Printf("[pipeline] started %d shards × %d workers per signal (trace_queue=%d, metric_queue=%d)",
		m.shardCount, workers, m.cfg.TraceQueueSize, m.cfg.MetricQueueSize)
}

// SubmitTrace enqueues a trace payload to the appropriate shard.
// Shard selection: hash(service.name) % shardCount.
func (m *Manager) SubmitTrace(p *model.TracePayload) bool {
	shard := m.traceShardIndex(p)
	select {
	case m.traceShards[shard] <- p:
		m.updateTraceDepth()
		return true
	default:
		metrics.QueueDropTotal.WithLabelValues("trace").Inc()
		return false
	}
}

// SubmitMetric enqueues a metric payload to the appropriate shard.
func (m *Manager) SubmitMetric(p *model.MetricPayload) bool {
	shard := m.metricShardIndex(p)
	select {
	case m.metricShards[shard] <- p:
		m.updateMetricDepth()
		return true
	default:
		metrics.QueueDropTotal.WithLabelValues("metric").Inc()
		return false
	}
}

// OnPreShutdown registers a hook that runs after workers drain but before exporters close.
func (m *Manager) OnPreShutdown(fn func()) {
	m.preShutdownHooks = append(m.preShutdownHooks, fn)
}

// Shutdown drains queues and waits for workers to finish.
func (m *Manager) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
	for i := 0; i < m.shardCount; i++ {
		close(m.traceShards[i])
		close(m.metricShards[i])
	}
	m.wg.Wait()

	// Run pre-shutdown hooks (e.g., flush batch processors)
	for _, hook := range m.preShutdownHooks {
		hook()
	}

	if m.traceExp != nil {
		if err := m.traceExp.Shutdown(); err != nil {
			log.Printf("[pipeline] trace exporter shutdown error: %v", err)
		}
	}
	if m.metricExp != nil {
		if err := m.metricExp.Shutdown(); err != nil {
			log.Printf("[pipeline] metric exporter shutdown error: %v", err)
		}
	}
	log.Println("[pipeline] shutdown complete")
}

// traceShardIndex extracts service.name from the first ResourceSpans and hashes it.
func (m *Manager) traceShardIndex(p *model.TracePayload) int {
	if m.shardCount <= 1 {
		return 0
	}
	if p == nil || p.Data == nil || len(p.Data.ResourceSpans) == 0 {
		return 0
	}
	rs := p.Data.ResourceSpans[0]
	if rs.Resource == nil {
		return 0
	}
	svcName := extractServiceName(rs.Resource.Attributes)
	return hashToShard(svcName, m.shardCount)
}

// metricShardIndex extracts service.name from the first ResourceMetrics and hashes it.
func (m *Manager) metricShardIndex(p *model.MetricPayload) int {
	if m.shardCount <= 1 {
		return 0
	}
	if p == nil || p.Data == nil || len(p.Data.ResourceMetrics) == 0 {
		return 0
	}
	rm := p.Data.ResourceMetrics[0]
	if rm.Resource == nil {
		return 0
	}
	svcName := extractServiceName(rm.Resource.Attributes)
	return hashToShard(svcName, m.shardCount)
}

// extractServiceName finds the "service.name" attribute value.
func extractServiceName(attrs []*commonpb.KeyValue) string {
	for _, kv := range attrs {
		if kv.Key == "service.name" {
			if sv := kv.Value.GetStringValue(); sv != "" {
				return sv
			}
		}
	}
	return ""
}

// hashToShard maps a string key to a shard index using FNV-1a.
func hashToShard(key string, shards int) int {
	if key == "" || shards <= 1 {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % shards
}

func (m *Manager) updateTraceDepth() {
	var total int
	for _, ch := range m.traceShards {
		total += len(ch)
	}
	metrics.QueueDepth.WithLabelValues("trace").Set(float64(total))
}

func (m *Manager) updateMetricDepth() {
	var total int
	for _, ch := range m.metricShards {
		total += len(ch)
	}
	metrics.QueueDepth.WithLabelValues("metric").Set(float64(total))
}

func (m *Manager) traceWorker(ctx context.Context, shard int) {
	defer m.wg.Done()
	for p := range m.traceShards[shard] {
		m.updateTraceDepth()

		var err error
		current := p
		for _, proc := range m.traceProcs {
			current, err = proc.ProcessTraces(current)
			if err != nil {
				log.Printf("[pipeline] shard=%d trace processor %s error: %v", shard, proc.Name(), err)
				break
			}
			if current == nil {
				break // filtered out (e.g., sampler)
			}
		}
		if err != nil || current == nil {
			continue
		}

		if m.traceExp != nil {
			if err := m.traceExp.ExportTraces(current); err != nil {
				log.Printf("[pipeline] shard=%d trace export error: %v", shard, err)
			}
		}
	}
}

func (m *Manager) metricWorker(ctx context.Context, shard int) {
	defer m.wg.Done()
	for p := range m.metricShards[shard] {
		m.updateMetricDepth()

		var err error
		current := p
		for _, proc := range m.metricProcs {
			current, err = proc.ProcessMetrics(current)
			if err != nil {
				log.Printf("[pipeline] shard=%d metric processor %s error: %v", shard, proc.Name(), err)
				break
			}
			if current == nil {
				break
			}
		}
		if err != nil || current == nil {
			continue
		}

		if m.metricExp != nil {
			if err := m.metricExp.ExportMetrics(current); err != nil {
				log.Printf("[pipeline] shard=%d metric export error: %v", shard, err)
			}
		}
	}
}
