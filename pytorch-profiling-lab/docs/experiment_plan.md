# 实验计划

1. Baseline：`configs/baseline.yaml`
2. DataLoader：`num_workers` 扫描（`dataloader_tuned.yaml`）
3. `pin_memory`：`baseline` vs `pin_memory_off.yaml`
4. AMP：`amp.yaml`（需 CUDA）
5. `torch.compile`：`compile.yaml`（PyTorch 2.x）

每组记录 `samples_per_sec`、显存与 trace 观察到的 DataLoader / CUDA 等待。
