// Package model defines the internal data types for the gateway pipeline.
// V1 uses lightweight wrappers around OTLP protobuf types.
package model

import (
	"time"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
)

// SignalType distinguishes trace vs metric payloads in the pipeline.
type SignalType int

const (
	SignalTrace  SignalType = iota
	SignalMetric
)

func (s SignalType) String() string {
	switch s {
	case SignalTrace:
		return "trace"
	case SignalMetric:
		return "metric"
	default:
		return "unknown"
	}
}

// TracePayload wraps an OTLP trace export request with metadata.
type TracePayload struct {
	Data       *coltracepb.ExportTraceServiceRequest
	ReceivedAt time.Time
	SpanCount  int
}

// MetricPayload wraps an OTLP metric export request with metadata.
type MetricPayload struct {
	Data          *colmetricspb.ExportMetricsServiceRequest
	ReceivedAt    time.Time
	DataPointCount int
}

// Processor is the interface every processing stage implements.
type Processor interface {
	Name() string
}

// TraceProcessor processes trace payloads.
type TraceProcessor interface {
	Processor
	ProcessTraces(payload *TracePayload) (*TracePayload, error)
}

// MetricProcessor processes metric payloads.
type MetricProcessor interface {
	Processor
	ProcessMetrics(payload *MetricPayload) (*MetricPayload, error)
}

// TraceExporter sends trace data to a backend.
type TraceExporter interface {
	ExportTraces(payload *TracePayload) error
	Shutdown() error
}

// MetricExporter sends metric data to a backend.
type MetricExporter interface {
	ExportMetrics(payload *MetricPayload) error
	Shutdown() error
}
