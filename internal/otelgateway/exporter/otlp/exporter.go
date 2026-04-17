// Package otlp implements a trace exporter that sends OTLP data to Tempo or any OTLP-compatible backend.
package otlp

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"ai-infra-platform/internal/otelgateway/config"
	gwmetrics "ai-infra-platform/internal/otelgateway/metrics"
	"ai-infra-platform/internal/otelgateway/model"
)

// TraceExporter sends traces to an OTLP gRPC backend (e.g., Tempo, Jaeger).
type TraceExporter struct {
	label  string // metrics label (e.g., "tempo", "jaeger")
	cfg    config.TempoExporterConfig
	conn   *grpc.ClientConn
	client coltracepb.TraceServiceClient
}

// New creates and connects the trace exporter with label "tempo" (backward-compatible).
func New(cfg config.TempoExporterConfig) (*TraceExporter, error) {
	return NewWithLabel(cfg, "tempo")
}

// NewWithLabel creates an OTLP trace exporter with a custom metrics label.
// Use this to create exporters for different backends (e.g., "jaeger").
func NewWithLabel(cfg config.TempoExporterConfig, label string) (*TraceExporter, error) {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(16 * 1024 * 1024)),
	}
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(cfg.Endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter dial %s: %w", cfg.Endpoint, err)
	}

	log.Printf("[exporter/otlp/%s] connected to %s (insecure=%v, timeout=%s)",
		label, cfg.Endpoint, cfg.Insecure, cfg.Timeout)

	return &TraceExporter{
		label:  label,
		cfg:    cfg,
		conn:   conn,
		client: coltracepb.NewTraceServiceClient(conn),
	}, nil
}

// ExportTraces sends a trace payload to the OTLP backend.
func (e *TraceExporter) ExportTraces(payload *model.TracePayload) error {
	gwmetrics.ExportAttemptTotal.WithLabelValues(e.label).Inc()

	ctx, cancel := context.WithTimeout(context.Background(), e.cfg.Timeout)
	defer cancel()

	start := time.Now()
	_, err := e.client.Export(ctx, payload.Data)
	elapsed := time.Since(start)
	gwmetrics.ExportLatency.WithLabelValues(e.label).Observe(elapsed.Seconds())

	if err != nil {
		gwmetrics.ExportFailureTotal.WithLabelValues(e.label).Inc()
		return fmt.Errorf("otlp/%s export: %w", e.label, err)
	}

	gwmetrics.ExportSuccessTotal.WithLabelValues(e.label).Inc()
	return nil
}

// Shutdown closes the gRPC connection.
func (e *TraceExporter) Shutdown() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}
