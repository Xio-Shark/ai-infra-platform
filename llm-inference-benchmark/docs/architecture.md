# 架构说明

- **网关**：`server/app.py` 暴露 OpenAI 兼容 `POST /v1/chat/completions`，可选将请求转发至 `VLLM_BASE_URL`；未配置时使用内存 mock。
- **指标**：`prometheus_client` 在进程内注册 Counter / Histogram，路径 `/metrics`。
- **压测**：`benchmark/run_benchmark.py` 使用 `httpx` 异步并发，支持流式 TTFT 近似统计。
