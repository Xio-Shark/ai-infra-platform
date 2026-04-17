// Package fanout implements a multi-destination exporter that sends data to multiple backends.
//
// V3: Fan-out traces to Tempo + Jaeger simultaneously.
// Uses "first-error" semantics: if any backend fails, the error is returned but
// other backends still receive the data (best-effort).
package fanout

import (
	"errors"
	"fmt"
	"log"

	"ai-infra-platform/internal/otelgateway/model"
)

// TraceExporter sends traces to multiple backends concurrently.
type TraceExporter struct {
	exporters []model.TraceExporter
	names     []string
}

// NewTraceExporter wraps multiple trace exporters into a fanout.
func NewTraceExporter(exporters map[string]model.TraceExporter) *TraceExporter {
	f := &TraceExporter{}
	for name, exp := range exporters {
		f.names = append(f.names, name)
		f.exporters = append(f.exporters, exp)
	}
	log.Printf("[fanout/trace] configured %d backends: %v", len(f.exporters), f.names)
	return f
}

// ExportTraces sends the payload to all configured backends.
// Errors are collected but do not prevent delivery to other backends.
func (f *TraceExporter) ExportTraces(payload *model.TracePayload) error {
	var errs []error
	for i, exp := range f.exporters {
		if err := exp.ExportTraces(payload); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.names[i], err))
		}
	}
	return errors.Join(errs...)
}

// Shutdown shuts down all wrapped exporters.
func (f *TraceExporter) Shutdown() error {
	var errs []error
	for i, exp := range f.exporters {
		if err := exp.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.names[i], err))
		}
	}
	return errors.Join(errs...)
}

// MetricExporter sends metrics to multiple backends concurrently.
type MetricExporter struct {
	exporters []model.MetricExporter
	names     []string
}

// NewMetricExporter wraps multiple metric exporters into a fanout.
func NewMetricExporter(exporters map[string]model.MetricExporter) *MetricExporter {
	f := &MetricExporter{}
	for name, exp := range exporters {
		f.names = append(f.names, name)
		f.exporters = append(f.exporters, exp)
	}
	log.Printf("[fanout/metric] configured %d backends: %v", len(f.exporters), f.names)
	return f
}

// ExportMetrics sends the payload to all configured backends.
func (f *MetricExporter) ExportMetrics(payload *model.MetricPayload) error {
	var errs []error
	for i, exp := range f.exporters {
		if err := exp.ExportMetrics(payload); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.names[i], err))
		}
	}
	return errors.Join(errs...)
}

// Shutdown shuts down all wrapped exporters.
func (f *MetricExporter) Shutdown() error {
	var errs []error
	for i, exp := range f.exporters {
		if err := exp.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.names[i], err))
		}
	}
	return errors.Join(errs...)
}
