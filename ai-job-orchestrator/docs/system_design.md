# 系统设计（摘要）

- **api-server**：创建 / 查询 Job，暴露 `/metrics`。
- **scheduler**：事务内将 `pending` 抢占为 `scheduled`。
- **worker**：将 `scheduled` 抢占为 `running`，执行 `ShellExecutor`（默认）后标记 `succeeded` / `failed`。
- **存储**：SQLite + 文件 DSN，迁移按 `migrations/*.sql` 顺序执行。
