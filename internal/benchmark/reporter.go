package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Report 完整的压测报告
type Report struct {
	RunName  string            `json:"run_name"`
	Config   Config            `json:"config"`
	Summary  Summary           `json:"summary"`
	Samples  []Sample          `json:"samples"`
	Metadata map[string]string `json:"metadata"`
}

// WriteReport 将报告同时输出为 JSON 和 Markdown
func WriteReport(outputDir, runName string, cfg Config, summary Summary, samples []Sample) (string, string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create output dir: %w", err)
	}

	report := Report{
		RunName: runName,
		Config:  cfg,
		Summary: summary,
		Samples: samples,
		Metadata: map[string]string{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"target_url":   cfg.TargetURL,
			"model":        cfg.Model,
		},
	}

	jsonPath := filepath.Join(outputDir, runName+".json")
	if err := writeJSON(jsonPath, report); err != nil {
		return "", "", err
	}

	mdPath := filepath.Join(outputDir, runName+".md")
	if err := writeMarkdown(mdPath, report); err != nil {
		return jsonPath, "", err
	}

	return jsonPath, mdPath, nil
}

func writeJSON(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, r Report) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Benchmark Report: %s\n\n", r.RunName))

	b.WriteString("## Configuration\n\n")
	b.WriteString(fmt.Sprintf("| Parameter | Value |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| Target | %s |\n", r.Config.TargetURL))
	b.WriteString(fmt.Sprintf("| Model | %s |\n", r.Config.Model))
	b.WriteString(fmt.Sprintf("| Concurrency | %d |\n", r.Config.Concurrency))
	b.WriteString(fmt.Sprintf("| Total Requests | %d |\n", r.Config.TotalReqs))
	b.WriteString(fmt.Sprintf("| Max Tokens | %d |\n", r.Config.MaxTokens))

	b.WriteString("\n## Results\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| QPS | %.2f |\n", r.Summary.QPS))
	b.WriteString(fmt.Sprintf("| Success Rate | %.2f%% |\n", r.Summary.SuccessRate*100))
	b.WriteString(fmt.Sprintf("| P50 Latency | %.2f ms |\n", r.Summary.P50LatencyMs))
	b.WriteString(fmt.Sprintf("| P95 Latency | %.2f ms |\n", r.Summary.P95LatencyMs))
	b.WriteString(fmt.Sprintf("| P99 Latency | %.2f ms |\n", r.Summary.P99LatencyMs))
	b.WriteString(fmt.Sprintf("| Avg Latency | %.2f ms |\n", r.Summary.AvgLatencyMs))
	b.WriteString(fmt.Sprintf("| Min Latency | %.2f ms |\n", r.Summary.MinLatencyMs))
	b.WriteString(fmt.Sprintf("| Max Latency | %.2f ms |\n", r.Summary.MaxLatencyMs))
	b.WriteString(fmt.Sprintf("| Avg Tokens/s | %.2f |\n", r.Summary.AvgTokensSec))
	b.WriteString(fmt.Sprintf("| Runtime | %.3f s |\n", r.Summary.RuntimeSec))

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
