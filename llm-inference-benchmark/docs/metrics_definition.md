# 指标定义

| 指标 | 说明 |
|------|------|
| QPS | 成功请求数 / 时间窗口 |
| P50/P95/P99 latency | 端到端请求耗时（压测客户端侧） |
| TTFT | 流式场景下首个含 `delta.content` 的 SSE 片段时间 − 请求开始 |
| tokens/s | 粗略：`completion_tokens/耗时` 或 流式按字符/4 估算 |
| `llm_bench_inference_requests_total` | 网关侧请求计数（按 status 标签） |
| `llm_bench_inference_latency_seconds` | 非流式请求处理直方图 |
