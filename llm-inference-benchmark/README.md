# LLM Inference Benchmark

OpenAI 兼容 API 网关（可转发 vLLM/SGLang）、异步压测客户端，以及 Prometheus / Grafana 监控骨架。

## 快速开始

```bash
cd llm-inference-benchmark
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 终端 1：API（默认 mock；转发上游需设置 VLLM_BASE_URL）
export PYTHONPATH=.
uvicorn server.app:app --host 0.0.0.0 --port 8080

# 终端 2：压测
export PYTHONPATH=.
python -m benchmark.run_benchmark --config configs/benchmark.yaml
```

转发到本机 vLLM（示例）：

```bash
export VLLM_BASE_URL=http://127.0.0.1:8000/v1
uvicorn server.app:app --host 0.0.0.0 --port 8080
```

## 监控

```bash
docker compose up -d
```

- Prometheus: <http://localhost:9090>
- Grafana: <http://localhost:3000>（默认 `admin` / `admin`）

`configs/prometheus.yml` 通过 `host.docker.internal:8080` 抓取本机 API 的 `/metrics`；Linux 若不可用请在 compose 中改为宿主机 IP。

## 目录说明

- `server/` — FastAPI 网关与 `/metrics`
- `benchmark/` — 并发压测、延迟 / TTFT / tokens/s 汇总
- `configs/` — 模型与压测、Prometheus 配置
- `monitoring/` — Grafana 面板与告警规则
- `traces/` — OTLP 可选（需 `OTEL_EXPORTER_OTLP_ENDPOINT` 与 exporter 依赖）

## 可选 OTLP

```bash
pip install opentelemetry-exporter-otlp
export OTEL_EXPORTER_OTLP_ENDPOINT=127.0.0.1:4317
```
