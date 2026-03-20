# AI Job Orchestrator

SQLite 持久化、HTTP API、独立 scheduler / worker 进程的 AI 作业编排骨架（training / inference / eval / benchmark）。

## 构建

```bash
cd ai-job-orchestrator
go mod tidy
make build
```

需在模块根目录运行（以便读取 `./migrations`），或通过 `MIGRATIONS_DIR` 指定 SQL 目录。

## 本地演示

三个终端，同一工作目录：

```bash
# 1) API
./bin/api-server

# 2) 调度：pending -> scheduled
./bin/scheduler

# 3) Worker：scheduled -> running -> succeeded（默认 Shell 执行 payload）
./bin/worker
```

创建任务：

```bash
curl -s -X POST localhost:8080/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d @examples/training_job.json | jq .
```

查询状态与执行记录：

```bash
JOB_ID=...
curl -s localhost:8080/api/v1/jobs/$JOB_ID | jq .
curl -s localhost:8080/api/v1/jobs/$JOB_ID/executions | jq .
```

指标：`GET http://localhost:8080/metrics`

## 环境变量

| 变量 | 说明 |
|------|------|
| `JOB_DB_DSN` | SQLite DSN，默认 `file:./data/jobs.db?...` |
| `MIGRATIONS_DIR` | 迁移 SQL 目录，默认 `migrations` |
| `API_LISTEN_ADDR` | 默认 `:8080` |
| `WORKER_EXECUTOR` | 空/`shell`（默认）、`http`、`k8s`（stub 会失败） |
| `SCHEDULER_INTERVAL_MS` | 调度轮询间隔毫秒 |

## Docker / K8s

`deploy/docker-compose.yml` 与 `deploy/k8s/*.yaml` 为占位模板，可按集群环境补全镜像与配置。
