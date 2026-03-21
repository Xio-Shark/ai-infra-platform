# Benchmark 对比报告：CPU Baseline vs High Concurrency

## 测试环境

- **硬件**: MacBook (Apple Silicon), CPU 推理
- **推理引擎**: Ollama v0.18.2
- **模型**: Qwen2.5-1.5B (986MB)
- **压测工具**: benchctl (Go goroutine 并发客户端)
- **时间**: 2026-03-21

## 测试参数

| 参数 | Baseline | High Concurrency |
|---|---|---|
| 并发数 | 3 | 10 |
| 总请求数 | 10 | 30 |
| max_tokens | 64 | 64 |
| 模型 | qwen2.5:1.5b | qwen2.5:1.5b |

## 结果对比

| 指标 | Baseline (c=3) | High Concurrency (c=10) | 变化 |
|---|---|---|---|
| **QPS** | 1.13 | 1.13 | 持平 |
| **P50 延迟** | 2,621 ms | 8,812 ms | +236% ↑ |
| **P95 延迟** | 2,679 ms | 8,841 ms | +230% ↑ |
| **P99 延迟** | 2,697 ms | 8,883 ms | +229% ↑ |
| **Avg Tokens/s** | 29.55 | 11.49 | -61% ↓ |
| **Success Rate** | 100% | 100% | 持平 |

## 瓶颈分析

1. **QPS 未随并发线性增长**：CPU 推理是串行瓶颈，Ollama 在 CPU 模式下无法并行处理多个推理请求，增加并发只增加排队时间
2. **延迟显著增长**：P50 从 2.6s → 8.8s，说明请求在队列中等待而非被并行处理
3. **tokens/s 大幅下降**：从 29.55 → 11.49，每个请求的有效 decode 时间被排队时间稀释

## 结论

- CPU 推理下的 **QPS 硬顶约 1.1**，不受并发数影响
- 提升吞吐的唯一路径是 **GPU 加速**（预期 RTX 4050 可将 tokens/s 提升 3-5x）
- 对于实际生产部署，应通过请求限流（rate limiting）将并发控制在与推理能力匹配的水平
