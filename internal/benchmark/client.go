// Package benchmark 提供面向 OpenAI-compatible 推理服务的
// 并发压测客户端，使用 goroutine 池实现可控并发。
package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config 压测参数配置对象
type Config struct {
	TargetURL   string // 推理服务地址，如 http://localhost:11434
	Endpoint    string // 请求路径，如 /v1/chat/completions
	Model       string // 模型名称
	Concurrency int    // 并发请求数
	TotalReqs   int    // 总请求数
	MaxTokens   int    // 最大输出 token 数
	Prompts     []string
}

// chatRequest 为 OpenAI-compatible chat completion 请求体
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse 为 OpenAI-compatible 响应的子集
type chatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Run 执行压测，返回所有请求的采样结果
func Run(ctx context.Context, cfg Config) ([]Sample, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 120 * time.Second}
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	samples := make([]Sample, 0, cfg.TotalReqs)

	for i := 0; i < cfg.TotalReqs; i++ {
		prompt := cfg.Prompts[i%len(cfg.Prompts)]
		wg.Add(1)
		go func(idx int, prompt string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			sample := executeRequest(ctx, client, cfg, prompt, idx)
			mu.Lock()
			samples = append(samples, sample)
			mu.Unlock()
		}(i, prompt)
	}

	wg.Wait()
	return samples, nil
}

func executeRequest(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	prompt string,
	idx int,
) Sample {
	reqBody := chatRequest{
		Model: cfg.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: 0.7,
		Stream:      false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := cfg.TargetURL + cfg.Endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return failedSample(idx, 0, err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return failedSample(idx, latency, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return failedSample(idx, latency,
			fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return failedSample(idx, latency, fmt.Errorf("decode response: %w", err))
	}

	outputTokens := chatResp.Usage.CompletionTokens
	tps := tokensPerSecond(outputTokens, latency)
	ttftHeader := resp.Header.Get("x-ttft-ms")

	return Sample{
		Index:        idx,
		LatencyMs:    float64(latency.Milliseconds()),
		TTFTMs:       parseFloatHeader(ttftHeader),
		PromptTokens: chatResp.Usage.PromptTokens,
		OutputTokens: outputTokens,
		TokensPerSec: tps,
		Success:      true,
	}
}

func failedSample(idx int, latency time.Duration, err error) Sample {
	return Sample{
		Index:     idx,
		LatencyMs: float64(latency.Milliseconds()),
		Success:   false,
		Error:     err.Error(),
	}
}

func validateConfig(cfg Config) error {
	if cfg.TargetURL == "" {
		return fmt.Errorf("target URL is required")
	}
	if cfg.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if cfg.TotalReqs <= 0 {
		return fmt.Errorf("total requests must be positive")
	}
	if len(cfg.Prompts) == 0 {
		return fmt.Errorf("at least one prompt is required")
	}
	return nil
}

func tokensPerSecond(tokens int, latency time.Duration) float64 {
	if latency <= 0 || tokens <= 0 {
		return 0
	}
	return float64(tokens) / latency.Seconds()
}
