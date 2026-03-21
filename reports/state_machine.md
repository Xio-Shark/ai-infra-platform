# 状态机

## Job 状态

- `pending`: 已提交，等待调度
- `scheduled`: 已被调度器选中
- `running`: 执行器已启动
- `succeeded`: 执行成功
- `failed`: 执行失败且没有可用重试
- `retrying`: 执行失败但仍可手动重试
- `cancelled`: 在执行前被取消

## 关键转换

- `pending/retrying -> scheduled`
- `scheduled -> running`
- `running -> succeeded`
- `running -> retrying`
- `running -> failed`
- `pending/scheduled/retrying -> cancelled`
