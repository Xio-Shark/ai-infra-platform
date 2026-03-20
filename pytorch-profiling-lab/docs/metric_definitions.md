# 指标

- **step wall time**：单个 epoch 墙上时钟 `wall_sec`
- **throughput**：`samples_per_sec = 样本数 / wall_sec`
- **accuracy**：CIFAR-10 验证集 top-1
- **Profiler**：`cpu_time_total` / `cuda_time_total` 排序查看算子与数据加载占比
