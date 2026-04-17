package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Receivers  ReceiverConfig  `yaml:"receivers"`
	Pipeline   PipelineConfig  `yaml:"pipeline"`
	Processors ProcessorConfig `yaml:"processors"`
	Exporters  ExporterConfig  `yaml:"exporters"`
	Retry      RetryConfig     `yaml:"retry"`
	WAL        WALConfig       `yaml:"wal"`
	Metrics    MetricsConfig   `yaml:"metrics"`
}

// ===== Receiver =====

type ReceiverConfig struct {
	GRPC GRPCReceiverConfig `yaml:"grpc"`
	HTTP HTTPReceiverConfig `yaml:"http"`
}

type GRPCReceiverConfig struct {
	Addr           string `yaml:"addr"`
	MaxRecvMsgSize int    `yaml:"max_recv_msg_size"` // bytes
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type HTTPReceiverConfig struct {
	Addr           string `yaml:"addr"`
	MaxRequestBody int    `yaml:"max_request_body"` // bytes
}

// ===== Pipeline =====

type PipelineConfig struct {
	TraceQueueSize  int `yaml:"trace_queue_size"`
	MetricQueueSize int `yaml:"metric_queue_size"`
	Workers         int `yaml:"workers"`
	ShardCount      int `yaml:"shard_count"` // V2: number of shards per signal type
}

// ===== Processors =====

type ProcessorConfig struct {
	Batch     BatchConfig     `yaml:"batch"`
	Sampler   SamplerConfig   `yaml:"sampler"`
	MemLimit  MemLimitConfig  `yaml:"memlimit"`
	RateLimit RateLimitConfig `yaml:"ratelimit"`
	Degrade   DegradeConfig   `yaml:"degrade"`
}

type BatchConfig struct {
	MaxItems      int           `yaml:"max_items"`
	MaxBytes      int           `yaml:"max_bytes"`
	FlushInterval time.Duration `yaml:"flush_interval"`
}

type SamplerConfig struct {
	SamplingRate float64 `yaml:"sampling_rate"` // 0.0 ~ 1.0
}

type MemLimitConfig struct {
	Enabled             bool          `yaml:"enabled"`
	HighWatermarkMB     uint64        `yaml:"high_watermark_mb"`
	CriticalWatermarkMB uint64        `yaml:"critical_watermark_mb"`
	CheckInterval       time.Duration `yaml:"check_interval"`
}

type RateLimitConfig struct {
	Enabled          bool    `yaml:"enabled"`
	TraceRatePerSec  float64 `yaml:"trace_rate_per_sec"`
	MetricRatePerSec float64 `yaml:"metric_rate_per_sec"`
	BurstMultiplier  float64 `yaml:"burst_multiplier"`
}

type DegradeConfig struct {
	Enabled              bool    `yaml:"enabled"`
	DegradedSamplingRate float64 `yaml:"degraded_sampling_rate"`
}

// ===== Exporters =====

type ExporterConfig struct {
	Tempo           TempoExporterConfig  `yaml:"tempo"`
	Jaeger          JaegerExporterConfig `yaml:"jaeger"` // V3: Jaeger fanout target
	VictoriaMetrics VMExporterConfig     `yaml:"victoria_metrics"`
}

type TempoExporterConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Insecure bool          `yaml:"insecure"`
	Timeout  time.Duration `yaml:"timeout"`
}

// JaegerExporterConfig configures the optional Jaeger OTLP exporter for trace dual-write.
type JaegerExporterConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Endpoint string        `yaml:"endpoint"` // OTLP gRPC endpoint (e.g., jaeger:4317)
	Insecure bool          `yaml:"insecure"`
	Timeout  time.Duration `yaml:"timeout"`
}

type VMExporterConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
}

// ===== Retry =====

type RetryConfig struct {
	Enabled        bool          `yaml:"enabled"`
	MaxAttempts    int           `yaml:"max_attempts"`
	InitialDelay   time.Duration `yaml:"initial_delay"`
	MaxDelay       time.Duration `yaml:"max_delay"`
	JitterFraction float64       `yaml:"jitter_fraction"` // 0.0-1.0
}

// ===== WAL =====

type WALConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Dir            string        `yaml:"dir"`
	SegmentMaxSize int64         `yaml:"segment_max_size"` // bytes
	SyncInterval   time.Duration `yaml:"sync_interval"`
}

// ===== Self-monitoring =====

type MetricsConfig struct {
	Addr string `yaml:"addr"` // /metrics listen address
}

// Load reads a YAML config file and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Receivers: ReceiverConfig{
			GRPC: GRPCReceiverConfig{
				Addr:           ":4317",
				MaxRecvMsgSize: 4 * 1024 * 1024, // 4MB
				MaxConcurrent:  100,
			},
			HTTP: HTTPReceiverConfig{
				Addr:           ":4318",
				MaxRequestBody: 4 * 1024 * 1024,
			},
		},
		Pipeline: PipelineConfig{
			TraceQueueSize:  10000,
			MetricQueueSize: 10000,
			Workers:         4,
			ShardCount:      4, // V2: 4 shards per signal type
		},
		Processors: ProcessorConfig{
			Batch: BatchConfig{
				MaxItems:      500,
				MaxBytes:      2 * 1024 * 1024,
				FlushInterval: 5 * time.Second,
			},
			Sampler: SamplerConfig{
				SamplingRate: 1.0,
			},
			MemLimit: MemLimitConfig{
				Enabled:             false, // V2: opt-in
				HighWatermarkMB:     384,   // 75% of 512MB
				CriticalWatermarkMB: 460,   // 90% of 512MB
				CheckInterval:       time.Second,
			},
			RateLimit: RateLimitConfig{
				Enabled:          false, // V2: opt-in
				TraceRatePerSec:  10000,
				MetricRatePerSec: 50000,
				BurstMultiplier:  2.0,
			},
			Degrade: DegradeConfig{
				Enabled:              false, // V2: opt-in
				DegradedSamplingRate: 0.1,   // keep 10% in degraded mode
			},
		},
		Exporters: ExporterConfig{
			Tempo: TempoExporterConfig{
				Endpoint: "localhost:4317",
				Insecure: true,
				Timeout:  10 * time.Second,
			},
			VictoriaMetrics: VMExporterConfig{
				Endpoint: "http://localhost:8428/api/v1/write",
				Timeout:  10 * time.Second,
			},
		},
		Retry: RetryConfig{
			Enabled:        false, // V2: opt-in
			MaxAttempts:    5,
			InitialDelay:   100 * time.Millisecond,
			MaxDelay:       10 * time.Second,
			JitterFraction: 0.2,
		},
		WAL: WALConfig{
			Enabled:        false, // V2: opt-in
			Dir:            "/tmp/gateway-wal",
			SegmentMaxSize: 64 * 1024 * 1024, // 64MB
			SyncInterval:   500 * time.Millisecond,
		},
		Metrics: MetricsConfig{
			Addr: ":8888",
		},
	}
}
