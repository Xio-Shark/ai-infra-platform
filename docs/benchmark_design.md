# Benchmark 设计文档

## 概述

平台内置 Go 并发压测客户端，用于对 OpenAI-compatible 推理服务执行基准测试。

压测能力以两种形式暴露：

1. **`benchctl` CLI** — 独立命令行工具，直接对接推理服务
2. **`benchmark` executor** — 作为 Job 类型集成到编排平台，通过 API 提交/调度/执行

## 架构

```text
benchctl CLI ──► internal/benchmark ──► OpenAI-compatible API
                      ▲
benchmark executor ───┘
   (from Job metadata)
```

## 压测流程

1. 根据 `Prompts` 列表和 `TotalReqs` 生成 workload
2. 使用 goroutine 池（大小 = `Concurrency`）并发发送 HTTP POST
3. 每个请求采集 `latency_ms`、`ttft_ms`、`output_tokens`、`tokens_per_sec`
4. 聚合为 `Summary`（QPS / P50 / P95 / P99 / TTFT / tokens/s / success_rate）
5. 输出 JSON + Markdown 报告

## 配置参数

| 参数 | CLI flag | Job metadata key | 默认值 |
|---|---|---|---|
| 推理服务地址 | `--target` | `bench_target` | `http://localhost:11434` |
| 请求路径 | `--endpoint` | `bench_endpoint` | `/v1/chat/completions` |
| 模型名称 | `--model` | `bench_model` | `qwen2.5:1.5b` |
| 并发数 | `--concurrency` | `bench_concurrency` | `5` |
| 总请求数 | `--total` | `bench_total_reqs` | `20` |
| 最大输出 token | `--max-tokens` | `bench_max_tokens` | `64` |
| prompt 列表 | `--prompts` | `bench_prompts` | 内置默认 |

## 指标定义

| Metric | Description |
|---|---|
| **QPS** | 成功请求数 / 总耗时 |
| **P50/P95/P99** | 延迟分位数（仅计算成功请求） |
| **TTFT** | Time To First Token（从 `x-ttft-ms` 响应 header） |
| **tokens/s** | output_tokens / 有效 decode 时间 |
| **success_rate** | 成功请求 / 总请求 |

## 实测结果

详见 [reports/benchmark_comparison.md](../reports/benchmark_comparison.md)。
