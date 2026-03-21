MySQL 适配层骨架目录。

当前阶段提供一个基于 `database/sql` 的 MySQL 可选实现，用于替换默认的 `MemoryStore`。

- 驱动：`github.com/go-sql-driver/mysql`
- 启用方式：`STORE_BACKEND=mysql` + `MYSQL_DSN=...`
- 范围：覆盖 jobs / executions 两张表的增改查
- 说明：本轮只补代码与接线，不在本机连接真实数据库
