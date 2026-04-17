package sampler

import (
	"encoding/binary"
	"testing"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"ai-infra-platform/internal/otelgateway/model"
)

func makeTracePayload(traceIDs ...[]byte) *model.TracePayload {
	var spans []*tracepb.Span
	for _, tid := range traceIDs {
		sid := make([]byte, 8)
		binary.BigEndian.PutUint64(sid, uint64(len(spans)+1))
		spans = append(spans, &tracepb.Span{
			TraceId: tid,
			SpanId:  sid,
			Name:    "test-span",
		})
	}
	return &model.TracePayload{
		Data: &coltracepb.ExportTraceServiceRequest{
			ResourceSpans: []*tracepb.ResourceSpans{
				{
					ScopeSpans: []*tracepb.ScopeSpans{
						{Spans: spans},
					},
				},
			},
		},
		SpanCount: len(spans),
	}
}

func makeTraceID(val uint64) []byte {
	tid := make([]byte, 16)
	binary.BigEndian.PutUint64(tid[:8], val)
	return tid
}

func TestSampler_KeepAll(t *testing.T) {
	t.Parallel()
	s := New(1.0)
	payload := makeTracePayload(makeTraceID(1), makeTraceID(2), makeTraceID(3))

	result, err := s.ProcessTraces(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result with rate=1.0")
	}
	if result.SpanCount != 3 {
		t.Errorf("expected 3 spans, got %d", result.SpanCount)
	}
}

func TestSampler_DropAll(t *testing.T) {
	t.Parallel()
	s := New(0.0)
	payload := makeTracePayload(makeTraceID(1), makeTraceID(2))

	result, err := s.ProcessTraces(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result with rate=0.0")
	}
}

func TestSampler_NilPayload(t *testing.T) {
	t.Parallel()
	s := New(0.5)

	result, err := s.ProcessTraces(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil payload")
	}
}

func TestSampler_Deterministic(t *testing.T) {
	t.Parallel()
	s := New(0.5)
	tid := makeTraceID(42)
	payload1 := makeTracePayload(tid)
	payload2 := makeTracePayload(tid)

	r1, _ := s.ProcessTraces(payload1)
	r2, _ := s.ProcessTraces(payload2)

	// Same trace ID should produce same sampling decision
	if (r1 == nil) != (r2 == nil) {
		t.Error("sampling should be deterministic for the same trace ID")
	}
}

func TestSampler_ClampRate(t *testing.T) {
	t.Parallel()
	s1 := New(-0.5)
	if s1.Rate() != 0 {
		t.Errorf("expected rate 0 for negative input, got %f", s1.Rate())
	}
	s2 := New(2.0)
	if s2.Rate() != 1 {
		t.Errorf("expected rate 1 for input > 1, got %f", s2.Rate())
	}
}

func TestSampler_PartialSampling(t *testing.T) {
	t.Parallel()
	// With rate=0.5, approximately half of 1000 distinct trace IDs should be kept
	s := New(0.5)
	kept := 0
	total := 1000

	for i := 0; i < total; i++ {
		payload := makeTracePayload(makeTraceID(uint64(i)))
		result, err := s.ProcessTraces(payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			kept++
		}
	}

	ratio := float64(kept) / float64(total)
	// Allow 15% tolerance
	if ratio < 0.35 || ratio > 0.65 {
		t.Errorf("expected ~50%% sampling, got %.1f%% (%d/%d)", ratio*100, kept, total)
	}
}

func TestSampler_SetRate_HotReload(t *testing.T) {
	t.Parallel()
	s := New(1.0)

	// Initially keep all
	payload := makeTracePayload(makeTraceID(1))
	result, _ := s.ProcessTraces(payload)
	if result == nil {
		t.Fatal("expected non-nil result with rate=1.0")
	}

	// Hot reload: drop all
	s.SetRate(0.0)
	if s.Rate() != 0.0 {
		t.Errorf("expected rate 0.0 after SetRate, got %f", s.Rate())
	}

	payload2 := makeTracePayload(makeTraceID(2))
	result2, _ := s.ProcessTraces(payload2)
	if result2 != nil {
		t.Error("expected nil result after setting rate=0.0")
	}

	// Hot reload: back to keep all
	s.SetRate(1.0)
	if s.Rate() != 1.0 {
		t.Errorf("expected rate 1.0 after SetRate, got %f", s.Rate())
	}

	// SetRate clamps out-of-range values
	s.SetRate(-1.0)
	if s.Rate() != 0.0 {
		t.Errorf("expected rate 0.0 after SetRate(-1), got %f", s.Rate())
	}
	s.SetRate(5.0)
	if s.Rate() != 1.0 {
		t.Errorf("expected rate 1.0 after SetRate(5), got %f", s.Rate())
	}
}
