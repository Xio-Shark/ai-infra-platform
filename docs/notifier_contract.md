# Notifier Contract

## 角色

`cmd/notifier` 当前被定义为独立的最小 HTTP 服务骨架，而不是库函数。这样未来接 webhook、IM、邮件或消息队列时，不会和 `api-server` 耦合在同一个进程边界内。

## 当前暴露的接口

- `GET /healthz`
  存活检查
- `GET /readyz`
  暴露当前目标 `API_SERVER_URL`
- `GET /config`
  返回 notifier 当前配置快照
- `GET /contract`
  返回 notifier 与 `api-server` 的契约说明

## 与 api-server 的契约

- notifier 不直接操作 `store.Repository`
- notifier 通过 API 层读取作业与执行信息：
  - `GET /jobs/{id}`
  - `GET /jobs/{id}/executions`
  - `GET /jobs/{id}/trace`
- notifier 当前不做真实投递，只保留边界与配置项

## 当前阶段约束

- 不接真实外部通知渠道
- 不维护通知投递状态表
- 不引入额外的消息中间件客户端
