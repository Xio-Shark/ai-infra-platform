# AI Infra Platform

[![CI](https://github.com/Xio-Shark/ai-infra-platform/actions/workflows/ci.yml/badge.svg)](https://github.com/Xio-Shark/ai-infra-platform/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

面向 AI 训练、推理和压测任务的作业编排平台，内置 Go 并发压测客户端。

## 核心能力

- **任务编排**：提交、调度、执行、重试、取消，支持 training / inference / benchmark 三种 Job 类型
- **多类型执行器**：Shell、K8s dry-run、K8s apply、HTTP、Benchmark（Go 并发压测）
- **内置压测客户端**：goroutine 并发池，对接 OpenAI-compatible 推理服务，采集 QPS / P50 / P95 / P99 / TTFT / tokens/s
- **Benchmark as Job**：压测可通过 API 提交为 benchmark 类型 Job，调度执行后结果关联到执行记录
- **可观测性**：Prometheus 指标（submit/schedule/run/success/fail/retry/cancel + runtime histogram）、trace 时间线
- **可插拔存储**：内存 store（默认） / MySQL（可选）

## 系统架构

```text
┌──────────────────────────────────────────────────────┐
│                    api-server (:8080)                 │
│  POST /jobs  GET /jobs  POST /jobs/{id}/run  ...     │
│  GET /metrics  GET /healthz  GET /jobs/{id}/trace     │
├──────────────────────────────────────────────────────┤
│   JobService │ ExecutionService │ Dispatcher          │
├──────────────┼──────────────────┼────────────────────┤
│  Store       │  Worker Registry │  Telemetry          │
│  (Memory/    │  ┌─ Shell        │  ┌─ Prometheus      │
│   MySQL)     │  ├─ K8s DryRun   │  └─ Trace Timeline  │
│              │  ├─ K8s Apply    │                      │
│              │  ├─ HTTP         │                      │
│              │  └─ Benchmark ◄──── internal/benchmark  │
└──────────────┴──────────────────┴────────────────────┘

┌──────────────────────────────────────────────────────┐
│                   benchctl CLI                        │
│  独立运行的 Go 压测工具，直接对接推理服务              │
│  go run ./cmd/benchctl --target http://... --concurrency 10│
└──────────────────────────────────────────────────────┘
```

## 快速开始

### 启动 api-server

```bash
make test        # 运行全部测试
make run         # 启动 api-server (:8080)
```

### 提交 benchmark job

```bash
# 新终端
bash scripts/seed_jobs.sh
bash scripts/smoke_test.sh
```

### 独立运行压测 CLI

```bash
make build-benchctl
./bin/benchctl \
  --target http://localhost:11434 \
  --model qwen2.5:1.5b \
  --concurrency 10 \
  --total 50 \
  --max-tokens 64
```

## API

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/jobs` | 创建任务 |
| `GET` | `/jobs` | 列出任务 |
| `GET` | `/jobs/{id}` | 查询任务 |
| `POST` | `/jobs/{id}/schedule` | 手动调度 |
| `POST` | `/jobs/{id}/run` | 调度并执行 |
| `POST` | `/jobs/{id}/retry` | 重试 |
| `POST` | `/jobs/{id}/cancel` | 取消 |
| `GET` | `/jobs/{id}/executions` | 执行记录 |
| `GET` | `/jobs/{id}/trace` | trace 时间线 |
| `POST` | `/dispatch/once?limit=N` | 批量调度 |
| `GET` | `/metrics` | Prometheus 指标 |
| `GET` | `/healthz` | 健康检查 |

## 项目结构

```text
.
├── cmd/
│   ├── api-server/main.go      # HTTP API 主入口
│   ├── scheduler/main.go       # 调度器演示
│   ├── worker/main.go          # Worker 演示
│   └── benchctl/main.go        # Go 压测 CLI
├── internal/
│   ├── model/                  # 领域模型 (Job, Execution, BenchmarkResult)
│   ├── store/                  # Repository 接口 + Memory/MySQL 实现
│   ├── service/                # JobService + ExecutionService
│   ├── scheduler/              # Dispatcher + PriorityQueue + Retry
│   ├── worker/                 # Shell / K8s / HTTP / Benchmark 执行器
│   ├── benchmark/              # Go 并发压测核心 (client, metrics, reporter)
│   ├── api/                    # HTTP Router
│   └── telemetry/              # Prometheus Metrics + Trace Timeline
├── deploy/                     # Docker Compose + K8s manifests
├── examples/                   # 示例 Job JSON
├── migrations/                 # SQL DDL
├── scripts/                    # 运维脚本
├── reports/                    # 压测报告归档
└── docs/                       # 设计文档
```

## 技术栈

- **语言**：Go 1.22
- **Web**：`net/http` 标准库
- **存储**：Memory（默认）/ MySQL（`STORE_BACKEND=mysql`）
- **执行**：`os/exec`（Shell）/ K8s manifest 构建 / `net/http`（HTTP）/ 内置 benchmark 引擎
- **观测**：自实现 Prometheus exporter + 内存 trace 时间线
- **依赖**：仅 `go-sql-driver/mysql`（标准库之外唯一依赖）

## 关键设计

### 状态机

```text
pending → scheduled → running → succeeded
                   ↘ failed → retrying → pending (重试)
pending/scheduled/retrying → cancelled
```

### Benchmark 执行器

benchmark executor 从 Job 的 `metadata` 字段读取压测配置（target、concurrency、prompts 等），调用 `internal/benchmark.Run()` 执行，结果 JSON 写入 execution 的 `logs` 字段。

### 压测指标

| Metric | Description |
|---|---|
| QPS | 成功请求数 / 总耗时 |
| P50/P95/P99 | 延迟分位数 |
| TTFT | Time To First Token（从 `x-ttft-ms` header） |
| tokens/s | 输出 token 数 / 有效时间 |
| success_rate | 成功比例 |
