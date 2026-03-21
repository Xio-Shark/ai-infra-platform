# 可观测性

## Metrics

当前导出的 Prometheus 指标包括：

- `ai_jobs_submitted_total`
- `ai_jobs_scheduled_total`
- `ai_jobs_running_total`
- `ai_jobs_succeeded_total`
- `ai_jobs_failed_total`
- `ai_jobs_retried_total`
- `ai_jobs_cancelled_total`
- `ai_job_runtime_seconds_sum`
- `ai_job_runtime_seconds_count`

## Trace

每个 job 创建时生成 `trace_id`，并记录以下关键 span：

- `job.submit`
- `job.persisted`
- `job.schedule`
- `job.run_started`
- `job.run_succeeded` 或 `job.run_failed`

可通过 `GET /jobs/{id}/trace` 查看完整时间线。
