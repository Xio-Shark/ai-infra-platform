# Infra 单仓

本目录包含三个可独立演进的项目（详见 `prompt.md`）：

| 目录 | 说明 |
|------|------|
| `llm-inference-benchmark/` | OpenAI 兼容网关、压测客户端、Prometheus/Grafana |
| `pytorch-profiling-lab/` | CIFAR-10 训练与 `torch.profiler` 实验骨架 |
| `ai-job-orchestrator/` | Go 作业编排 API + scheduler + worker（SQLite） |

各子项目自带 `README.md`。开发时进入对应子目录按文档操作即可。
