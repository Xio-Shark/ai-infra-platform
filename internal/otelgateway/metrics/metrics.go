package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Gateway metrics registry — all metrics use "gateway_" prefix.

var (
	// ===== Ingest =====

	IngestRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_ingest_requests_total",
		Help: "Total number of ingest requests received.",
	})

	IngestItemsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_ingest_items_total",
		Help: "Total number of items (spans/metrics) received.",
	}, []string{"signal"}) // signal=trace|metric

	IngestDecodeFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_ingest_decode_failures_total",
		Help: "Total number of decode failures at ingest.",
	})

	// ===== Queue (V2) =====

	QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gateway_queue_depth",
		Help: "Current queue depth.",
	}, []string{"signal"})

	QueueCapacity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gateway_queue_capacity",
		Help: "Queue capacity.",
	}, []string{"signal"})

	QueueDropTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_queue_drop_total",
		Help: "Total number of items dropped due to full queue.",
	}, []string{"signal"})

	BackpressureEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_backpressure_events_total",
		Help: "Total number of backpressure events triggered.",
	})

	// ===== Batch =====

	BatchFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_batch_flush_total",
		Help: "Total number of batch flushes.",
	}, []string{"signal"})

	BatchItemsHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_batch_items_histogram",
		Help:    "Number of items per batch flush.",
		Buckets: []float64{1, 10, 50, 100, 200, 500, 1000, 2000, 5000},
	}, []string{"signal"})

	BatchFlushLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_batch_flush_latency_seconds",
		Help:    "Latency of batch flush operations.",
		Buckets: prometheus.DefBuckets,
	}, []string{"signal"})

	BatchTimeoutFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_batch_timeout_flush_total",
		Help: "Total batch flushes triggered by timeout.",
	}, []string{"signal"})

	BatchSizeFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_batch_size_flush_total",
		Help: "Total batch flushes triggered by size limit.",
	}, []string{"signal"})

	// ===== Export =====

	ExportAttemptTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_export_attempt_total",
		Help: "Total export attempts.",
	}, []string{"exporter"})

	ExportSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_export_success_total",
		Help: "Total successful exports.",
	}, []string{"exporter"})

	ExportFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_export_failure_total",
		Help: "Total failed exports.",
	}, []string{"exporter"})

	ExportRetryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_export_retry_total",
		Help: "Total export retries.",
	}, []string{"exporter"})

	ExportLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_export_latency_seconds",
		Help:    "Latency of export operations.",
		Buckets: prometheus.DefBuckets,
	}, []string{"exporter"})

	// ===== WAL (V2) =====

	WALAppendTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_wal_append_total",
		Help: "Total WAL append operations.",
	})

	WALReplayTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_wal_replay_total",
		Help: "Total WAL replay operations.",
	})

	WALReplayLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gateway_wal_replay_latency_seconds",
		Help:    "Latency of WAL replay operations.",
		Buckets: prometheus.DefBuckets,
	})

	WALCorruptionTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_wal_corruption_total",
		Help: "Total WAL corruption events detected.",
	})

	// ===== Degrade (V2) =====

	DegradeMode = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_degrade_mode",
		Help: "Current degrade mode (0=normal, 1=degraded, 2=critical).",
	})

	SamplingRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_sampling_ratio",
		Help: "Current effective sampling ratio.",
	})

	RejectedSpansTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_rejected_spans_total",
		Help: "Total rejected spans due to rate limiting or degradation.",
	})
)
