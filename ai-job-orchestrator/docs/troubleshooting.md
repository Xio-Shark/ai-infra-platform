# 排障

- **迁移失败**：确认进程工作目录为模块根目录，或设置 `MIGRATIONS_DIR`。
- **scheduler/worker 无进展**：三者需共享同一 `JOB_DB_DSN` 指向的 SQLite 文件。
- **权限**：`data/` 目录需可写。
