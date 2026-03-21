package benchmark

import (
	"math"
	"sort"
	"strconv"
)

// Sample 单次请求的采样数据
type Sample struct {
	Index        int     `json:"index"`
	LatencyMs    float64 `json:"latency_ms"`
	TTFTMs       float64 `json:"ttft_ms,omitempty"`
	PromptTokens int     `json:"prompt_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TokensPerSec float64 `json:"tokens_per_sec"`
	Success      bool    `json:"success"`
	Error        string  `json:"error,omitempty"`
}

// Summary 压测结果聚合指标
type Summary struct {
	TotalRequests int     `json:"total_requests"`
	SuccessCount  int     `json:"success_count"`
	ErrorCount    int     `json:"error_count"`
	SuccessRate   float64 `json:"success_rate"`
	QPS           float64 `json:"qps"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	MinLatencyMs  float64 `json:"min_latency_ms"`
	MaxLatencyMs  float64 `json:"max_latency_ms"`
	AvgTTFTMs    float64 `json:"avg_ttft_ms,omitempty"`
	AvgTokensSec float64 `json:"avg_tokens_per_sec"`
	RuntimeSec   float64 `json:"runtime_sec"`
}

// Summarize 从采样结果聚合为 Summary
func Summarize(samples []Sample, runtimeSec float64) Summary {
	total := len(samples)
	if total == 0 {
		return Summary{RuntimeSec: runtimeSec}
	}

	var successCount int
	latencies := make([]float64, 0, total)
	var totalLatency, totalTPS, totalTTFT float64
	var ttftCount int

	for _, s := range samples {
		if !s.Success {
			continue
		}
		successCount++
		latencies = append(latencies, s.LatencyMs)
		totalLatency += s.LatencyMs
		totalTPS += s.TokensPerSec
		if s.TTFTMs > 0 {
			totalTTFT += s.TTFTMs
			ttftCount++
		}
	}

	sort.Float64s(latencies)
	errorCount := total - successCount
	successRate := float64(successCount) / float64(total)
	qps := safeDiv(float64(successCount), runtimeSec)
	avgLatency := safeDiv(totalLatency, float64(successCount))
	avgTPS := safeDiv(totalTPS, float64(successCount))
	avgTTFT := safeDiv(totalTTFT, float64(ttftCount))

	return Summary{
		TotalRequests: total,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		SuccessRate:   round(successRate, 4),
		QPS:           round(qps, 2),
		P50LatencyMs:  percentile(latencies, 0.50),
		P95LatencyMs:  percentile(latencies, 0.95),
		P99LatencyMs:  percentile(latencies, 0.99),
		AvgLatencyMs:  round(avgLatency, 2),
		MinLatencyMs:  minFloat(latencies),
		MaxLatencyMs:  maxFloat(latencies),
		AvgTTFTMs:    round(avgTTFT, 2),
		AvgTokensSec: round(avgTPS, 2),
		RuntimeSec:   round(runtimeSec, 3),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return round(sorted[lower], 2)
	}
	frac := idx - float64(lower)
	return round(sorted[lower]*(1-frac)+sorted[upper]*frac, 2)
}

func minFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return round(vals[0], 2)
}

func maxFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return round(vals[len(vals)-1], 2)
}

func safeDiv(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}

func round(val float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(val*pow) / pow
}

func parseFloatHeader(val string) float64 {
	if val == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(val, 64)
	return f
}
