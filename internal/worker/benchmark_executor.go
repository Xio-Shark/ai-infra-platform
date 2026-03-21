package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-infra-platform/internal/benchmark"
	"ai-infra-platform/internal/model"
)

const benchmarkExecutorName = "benchmark"

// 从 Job metadata 读取的配置键
const (
	metaKeyTarget      = "bench_target"
	metaKeyEndpoint    = "bench_endpoint"
	metaKeyModel       = "bench_model"
	metaKeyConcurrency = "bench_concurrency"
	metaKeyTotalReqs   = "bench_total_reqs"
	metaKeyMaxTokens   = "bench_max_tokens"
	metaKeyPrompts     = "bench_prompts"
)

// BenchmarkExecutor 将压测作为 Job 执行器，
// 通过 Job 的 metadata 传入压测配置。
type BenchmarkExecutor struct{}

func (e BenchmarkExecutor) Name() string {
	return benchmarkExecutorName
}

// Execute 从 Job metadata 解析配置，执行压测，
// 返回结果 JSON 作为 execution logs。
func (e BenchmarkExecutor) Execute(ctx context.Context, job model.Job) (Result, error) {
	cfg, err := parseBenchmarkConfig(job)
	if err != nil {
		return Result{ExitCode: 1}, fmt.Errorf("parse benchmark config: %w", err)
	}

	start := time.Now()
	samples, err := benchmark.Run(ctx, cfg)
	runtimeSec := time.Since(start).Seconds()
	if err != nil {
		return Result{ExitCode: 1}, fmt.Errorf("run benchmark: %w", err)
	}

	summary := benchmark.Summarize(samples, runtimeSec)
	resultJSON, _ := json.MarshalIndent(summary, "", "  ")

	return Result{
		Logs:     string(resultJSON),
		ExitCode: exitCodeFromSummary(summary),
	}, nil
}

func parseBenchmarkConfig(job model.Job) (benchmark.Config, error) {
	target := job.Metadata[metaKeyTarget]
	if target == "" {
		return benchmark.Config{}, fmt.Errorf("metadata %q is required", metaKeyTarget)
	}
	promptsRaw := job.Metadata[metaKeyPrompts]
	if promptsRaw == "" {
		promptsRaw = "Hello, tell me a joke"
	}

	return benchmark.Config{
		TargetURL:   target,
		Endpoint:    stringOrDefault(job.Metadata[metaKeyEndpoint], "/v1/chat/completions"),
		Model:       stringOrDefault(job.Metadata[metaKeyModel], "qwen2.5:1.5b"),
		Concurrency: intOrDefault(job.Metadata[metaKeyConcurrency], 5),
		TotalReqs:   intOrDefault(job.Metadata[metaKeyTotalReqs], 20),
		MaxTokens:   intOrDefault(job.Metadata[metaKeyMaxTokens], 64),
		Prompts:     strings.Split(promptsRaw, ";"),
	}, nil
}

func exitCodeFromSummary(s benchmark.Summary) int {
	if s.ErrorCount > 0 {
		return 1
	}
	return 0
}

func stringOrDefault(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func intOrDefault(val string, fallback int) int {
	if val == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(val, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}
