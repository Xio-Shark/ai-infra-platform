# 简历 Bullet — AI Infra Platform

## 推荐使用（已用真实数据填充）

### AI Infra Platform｜面向 AI 训练/推理/压测的作业编排平台

技术栈：Go / net/http / MySQL / Kubernetes / Prometheus

- 设计并实现 AI 作业编排平台，拆分 api-server、scheduler、worker 三个服务，支持 training/inference/benchmark 三种 Job 类型，提供 12 条 API，覆盖任务提交、优先级调度、失败重试、执行回查与取消
- 抽象 Shell、K8s dry-run、K8s apply、HTTP、Benchmark 五类执行器，其中 Benchmark 执行器内置 Go goroutine 并发压测客户端，对接 OpenAI-compatible 推理服务（Ollama/vLLM），自动采集 QPS、P50/P95/P99 延迟、TTFT、tokens/s 等指标并生成报告
- 实测 Qwen2.5-1.5B，CPU baseline QPS=1.13、P50=2621ms、29.5 tok/s；并发从 3 提升到 10 后 P50 延迟增长 236%（排队瓶颈），定位出 CPU 推理串行瓶颈
- 全链路接入 Prometheus 指标（submit/schedule/run/success/fail/retry 六阶段计数器 + runtime histogram）与 trace 时间线查询，支持 memory/MySQL 双存储后端

## 面试提问准备

**Q: 为什么用 Go 写压测客户端而不是用现成工具？**
A: 现有工具（wrk/vegeta）不理解 LLM 推理的 token 级指标（TTFT、tokens/s）。自研客户端可以解析 OpenAI-compatible 响应体中的 usage 字段和 x-ttft-ms header，计算推理特有指标。

**Q: benchmark executor 是怎么集成到编排平台的？**
A: 通过策略模式。所有 executor 实现同一个 `Executor` interface，benchmark executor 从 Job 的 metadata map 读取压测配置（target/concurrency/prompts），调用 `internal/benchmark.Run()` 执行，结果 JSON 写入 execution 的 logs 字段。

**Q: 为什么并发增加 QPS 没变？**
A: Ollama CPU 模式下推理是串行的（一次只处理一个请求），增加并发只增加排队等待时间。QPS 硬顶 ≈ 1/avg_latency。这说明推理服务的吞吐瓶颈在计算侧而非 IO 侧，解法是 GPU 加速或模型量化。
