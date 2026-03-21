# AI Infra Platform｜API 请求与响应样例

以下为固定输入输出，可在本地 `make run` 后直接验证。

## 1. 健康检查

```bash
curl -s http://localhost:8080/healthz
```

```json
{"status":"ok"}
```

## 2. 创建 Training Job

```bash
curl -s -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "train-resnet50",
    "job_type": "training",
    "executor": "shell",
    "command": ["echo", "training started"],
    "priority": 5,
    "max_retries": 2
  }'
```

```json
{
  "id": "job-xxxx-xxxx",
  "name": "train-resnet50",
  "job_type": "training",
  "executor": "shell",
  "status": "pending",
  "priority": 5,
  "max_retries": 2,
  "created_at": "2026-03-22T02:55:00Z"
}
```

## 3. 调度并执行

```bash
curl -s -X POST http://localhost:8080/jobs/{id}/run
```

```json
{
  "job_id": "job-xxxx-xxxx",
  "execution_id": "exec-xxxx-xxxx",
  "status": "succeeded",
  "started_at": "2026-03-22T02:55:01Z",
  "finished_at": "2026-03-22T02:55:01Z",
  "logs": "training started\n"
}
```

## 4. 查询执行记录

```bash
curl -s http://localhost:8080/jobs/{id}/executions
```

```json
[
  {
    "id": "exec-xxxx-xxxx",
    "job_id": "job-xxxx-xxxx",
    "status": "succeeded",
    "logs": "training started\n"
  }
]
```

## 5. 查询 Trace 时间线

```bash
curl -s http://localhost:8080/jobs/{id}/trace
```

```json
{
  "trace_id": "trace-xxxx",
  "spans": [
    {"name": "job.submit", "timestamp": "..."},
    {"name": "job.persisted", "timestamp": "..."},
    {"name": "job.schedule", "timestamp": "..."},
    {"name": "job.run_started", "timestamp": "..."},
    {"name": "job.run_succeeded", "timestamp": "..."}
  ]
}
```

## 6. 查询 Prometheus 指标

```bash
curl -s http://localhost:8080/metrics | head -20
```

```text
# HELP ai_jobs_submitted_total Total number of jobs submitted
# TYPE ai_jobs_submitted_total counter
ai_jobs_submitted_total 1
# HELP ai_jobs_succeeded_total Total number of jobs succeeded
# TYPE ai_jobs_succeeded_total counter
ai_jobs_succeeded_total 1
```

## 7. 创建 Benchmark Job

```bash
curl -s -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "bench-qwen",
    "job_type": "benchmark",
    "executor": "benchmark",
    "command": ["benchmark"],
    "priority": 3,
    "metadata": {
      "bench_target": "http://localhost:11434",
      "bench_model": "qwen2.5:1.5b",
      "bench_concurrency": "3",
      "bench_total_reqs": "5",
      "bench_max_tokens": "32"
    }
  }'
```

## 8. 批量调度

```bash
curl -s -X POST "http://localhost:8080/dispatch/once?limit=3"
```
