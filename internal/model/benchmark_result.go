package model

// BenchmarkResult 压测 Job 的执行结果，
// 关联到 Execution 记录，可通过 API 查询。
type BenchmarkResult struct {
	QPS          float64 `json:"qps"`
	SuccessRate  float64 `json:"success_rate"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	AvgTokensSec float64 `json:"avg_tokens_per_sec"`
	TotalReqs    int     `json:"total_requests"`
	SuccessCount int     `json:"success_count"`
	RuntimeSec   float64 `json:"runtime_sec"`
}
