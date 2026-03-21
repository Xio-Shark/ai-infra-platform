# API 参考

## 创建任务

```http
POST /jobs
Content-Type: application/json
```

关键字段：

- `name`
- `job_type`: `training | inference | benchmark`
- `executor`: `shell | k8s-dry-run | http`
- `command`: 非空字符串数组
- `max_retries`

## 查询任务

- `GET /jobs`
- `GET /jobs/{id}`

## 调度与执行

- `POST /jobs/{id}/schedule`
- `POST /jobs/{id}/run`
- `POST /dispatch/once?limit=3`

## 控制类接口

- `POST /jobs/{id}/retry`
- `POST /jobs/{id}/cancel`

## 可观测性接口

- `GET /jobs/{id}/executions`
- `GET /jobs/{id}/trace`
- `GET /metrics`
- `GET /healthz`
