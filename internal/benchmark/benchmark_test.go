package benchmark

import (
	"testing"
)

func TestSummarize_Empty(t *testing.T) {
	summary := Summarize(nil, 1.0)
	if summary.TotalRequests != 0 {
		t.Errorf("expected 0 total, got %d", summary.TotalRequests)
	}
}

func TestSummarize_AllSuccess(t *testing.T) {
	samples := []Sample{
		{Index: 0, LatencyMs: 100, OutputTokens: 10, TokensPerSec: 100, Success: true},
		{Index: 1, LatencyMs: 200, OutputTokens: 20, TokensPerSec: 100, Success: true},
		{Index: 2, LatencyMs: 300, OutputTokens: 30, TokensPerSec: 100, Success: true},
		{Index: 3, LatencyMs: 400, OutputTokens: 40, TokensPerSec: 100, Success: true},
	}
	summary := Summarize(samples, 2.0)

	if summary.TotalRequests != 4 {
		t.Errorf("total: want 4, got %d", summary.TotalRequests)
	}
	if summary.SuccessCount != 4 {
		t.Errorf("success: want 4, got %d", summary.SuccessCount)
	}
	if summary.ErrorCount != 0 {
		t.Errorf("errors: want 0, got %d", summary.ErrorCount)
	}
	if summary.SuccessRate != 1.0 {
		t.Errorf("rate: want 1.0, got %f", summary.SuccessRate)
	}
	if summary.QPS != 2.0 {
		t.Errorf("QPS: want 2.0, got %f", summary.QPS)
	}
	if summary.P50LatencyMs < 200 || summary.P50LatencyMs > 300 {
		t.Errorf("P50: want ~250, got %f", summary.P50LatencyMs)
	}
	if summary.MinLatencyMs != 100 {
		t.Errorf("min: want 100, got %f", summary.MinLatencyMs)
	}
	if summary.MaxLatencyMs != 400 {
		t.Errorf("max: want 400, got %f", summary.MaxLatencyMs)
	}
}

func TestSummarize_WithErrors(t *testing.T) {
	samples := []Sample{
		{Index: 0, LatencyMs: 100, Success: true, TokensPerSec: 50},
		{Index: 1, LatencyMs: 200, Success: false, Error: "timeout"},
	}
	summary := Summarize(samples, 1.0)

	if summary.SuccessCount != 1 {
		t.Errorf("success: want 1, got %d", summary.SuccessCount)
	}
	if summary.ErrorCount != 1 {
		t.Errorf("errors: want 1, got %d", summary.ErrorCount)
	}
	if summary.SuccessRate != 0.5 {
		t.Errorf("rate: want 0.5, got %f", summary.SuccessRate)
	}
}

func TestPercentile(t *testing.T) {
	sorted := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	p50 := percentile(sorted, 0.50)
	if p50 < 50 || p50 > 60 {
		t.Errorf("P50: want ~55, got %f", p50)
	}
	p99 := percentile(sorted, 0.99)
	if p99 < 99 || p99 > 100 {
		t.Errorf("P99: want ~100, got %f", p99)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		ok   bool
	}{
		{"valid", Config{TargetURL: "http://x", Concurrency: 1, TotalReqs: 1, Prompts: []string{"hi"}}, true},
		{"no url", Config{Concurrency: 1, TotalReqs: 1, Prompts: []string{"hi"}}, false},
		{"no concurrency", Config{TargetURL: "http://x", Concurrency: 0, TotalReqs: 1, Prompts: []string{"hi"}}, false},
		{"no prompts", Config{TargetURL: "http://x", Concurrency: 1, TotalReqs: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if tt.ok && err != nil {
				t.Errorf("expected ok, got %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
