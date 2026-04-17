package retry

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"ai-infra-platform/internal/otelgateway/model"
)

// mockTraceExporter records calls and can be configured to fail.
type mockTraceExporter struct {
	calls     atomic.Int32
	failUntil int32 // fail the first N calls
}

func (m *mockTraceExporter) ExportTraces(_ *model.TracePayload) error {
	n := m.calls.Add(1)
	if n <= m.failUntil {
		return errors.New("mock: temporary failure")
	}
	return nil
}

func (m *mockTraceExporter) Shutdown() error { return nil }

// mockMetricExporter records calls and can be configured to fail.
type mockMetricExporter struct {
	calls     atomic.Int32
	failUntil int32
}

func (m *mockMetricExporter) ExportMetrics(_ *model.MetricPayload) error {
	n := m.calls.Add(1)
	if n <= m.failUntil {
		return errors.New("mock: temporary failure")
	}
	return nil
}

func (m *mockMetricExporter) Shutdown() error { return nil }

func TestTraceExporter_SuccessOnFirstAttempt(t *testing.T) {
	t.Parallel()
	inner := &mockTraceExporter{}
	cfg := Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, JitterFraction: 0}
	exp := NewTraceExporter(inner, "test", cfg)

	err := exp.ExportTraces(&model.TracePayload{})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if got := inner.calls.Load(); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestTraceExporter_RetryThenSuccess(t *testing.T) {
	t.Parallel()
	inner := &mockTraceExporter{failUntil: 2} // fail first 2, succeed on 3rd
	cfg := Config{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, JitterFraction: 0}
	exp := NewTraceExporter(inner, "test", cfg)

	err := exp.ExportTraces(&model.TracePayload{})
	if err != nil {
		t.Fatalf("expected nil error after retries, got: %v", err)
	}
	if got := inner.calls.Load(); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestTraceExporter_ExhaustedRetries(t *testing.T) {
	t.Parallel()
	inner := &mockTraceExporter{failUntil: 100} // always fail
	cfg := Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, JitterFraction: 0}
	exp := NewTraceExporter(inner, "test", cfg)

	err := exp.ExportTraces(&model.TracePayload{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if got := inner.calls.Load(); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestMetricExporter_RetryThenSuccess(t *testing.T) {
	t.Parallel()
	inner := &mockMetricExporter{failUntil: 1}
	cfg := Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, JitterFraction: 0}
	exp := NewMetricExporter(inner, "test", cfg)

	err := exp.ExportMetrics(&model.MetricPayload{})
	if err != nil {
		t.Fatalf("expected nil error after retry, got: %v", err)
	}
	if got := inner.calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
}

func TestBackoff_ExponentialGrowth(t *testing.T) {
	t.Parallel()
	cfg := Config{InitialDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second, JitterFraction: 0}

	d0 := backoff(cfg, 0)
	d1 := backoff(cfg, 1)
	d2 := backoff(cfg, 2)

	if d0 != 100*time.Millisecond {
		t.Fatalf("attempt 0: expected 100ms, got %s", d0)
	}
	if d1 != 200*time.Millisecond {
		t.Fatalf("attempt 1: expected 200ms, got %s", d1)
	}
	if d2 != 400*time.Millisecond {
		t.Fatalf("attempt 2: expected 400ms, got %s", d2)
	}
}

func TestBackoff_CappedAtMaxDelay(t *testing.T) {
	t.Parallel()
	cfg := Config{InitialDelay: time.Second, MaxDelay: 5 * time.Second, JitterFraction: 0}

	d := backoff(cfg, 10) // 2^10 * 1s = 1024s >> 5s
	if d != 5*time.Second {
		t.Fatalf("expected max delay 5s, got %s", d)
	}
}
