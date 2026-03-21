// benchctl 是独立的 Go 压测 CLI 工具，
// 用于对 OpenAI-compatible 推理服务执行并发基准测试。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ai-infra-platform/internal/benchmark"
)

const defaultEndpoint = "/v1/chat/completions"

func main() {
	os.Exit(run())
}

func run() int {
	cfg := parseFlags()

	log.Printf("benchctl: target=%s concurrency=%d total=%d",
		cfg.TargetURL, cfg.Concurrency, cfg.TotalReqs)

	ctx := context.Background()
	start := time.Now()
	samples, err := benchmark.Run(ctx, cfg)
	runtimeSec := time.Since(start).Seconds()

	if err != nil {
		log.Printf("benchmark failed: %v", err)
		return 1
	}

	summary := benchmark.Summarize(samples, runtimeSec)
	printSummary(summary)

	outputDir := "reports/generated"
	runName := fmt.Sprintf("bench_%s", time.Now().Format("20060102_150405"))
	jsonPath, mdPath, err := benchmark.WriteReport(outputDir, runName, cfg, summary, samples)
	if err != nil {
		log.Printf("write report failed: %v", err)
		return 1
	}

	log.Printf("json report: %s", jsonPath)
	log.Printf("markdown report: %s", mdPath)
	return exitCode(summary)
}

func parseFlags() benchmark.Config {
	target := flag.String("target", "http://localhost:11434", "推理服务地址")
	endpoint := flag.String("endpoint", defaultEndpoint, "请求路径")
	model := flag.String("model", "qwen2.5:1.5b", "模型名称")
	concurrency := flag.Int("concurrency", 5, "并发请求数")
	total := flag.Int("total", 20, "总请求数")
	maxTokens := flag.Int("max-tokens", 64, "最大输出 token 数")
	prompts := flag.String("prompts", "Hello, tell me a joke;What is AI?;Explain Go concurrency", "分号分隔的 prompt 列表")
	flag.Parse()

	promptList := strings.Split(*prompts, ";")
	return benchmark.Config{
		TargetURL:   *target,
		Endpoint:    *endpoint,
		Model:       *model,
		Concurrency: *concurrency,
		TotalReqs:   *total,
		MaxTokens:   *maxTokens,
		Prompts:     promptList,
	}
}

func printSummary(s benchmark.Summary) {
	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("  QPS:            %.2f\n", s.QPS)
	fmt.Printf("  Success Rate:   %.2f%%\n", s.SuccessRate*100)
	fmt.Printf("  P50 Latency:    %.2f ms\n", s.P50LatencyMs)
	fmt.Printf("  P95 Latency:    %.2f ms\n", s.P95LatencyMs)
	fmt.Printf("  P99 Latency:    %.2f ms\n", s.P99LatencyMs)
	fmt.Printf("  Avg Tokens/s:   %.2f\n", s.AvgTokensSec)
	fmt.Printf("  Runtime:        %.3f s\n", s.RuntimeSec)
	fmt.Printf("=========================\n\n")
}

func exitCode(s benchmark.Summary) int {
	if s.ErrorCount > 0 {
		return 1
	}
	return 0
}
