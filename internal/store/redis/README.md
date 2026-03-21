Redis 适配层骨架目录。

当前阶段仅保留 `store.Repository` 的注入边界，不连接真实 Redis。

- 实现文件：`store.go`
- 当前行为：所有方法返回 `store.ErrNotImplemented`
- 目的：预留未来缓存、分布式锁或队列能力的落点
