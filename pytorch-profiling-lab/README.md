# PyTorch Profiling Lab

CIFAR-10 + ResNet18 基线训练、`torch.profiler` 导出 Chrome trace，以及多组配置对比实验入口。

## 环境

```bash
cd pytorch-profiling-lab
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
```

## 训练

```bash
export PYTHONPATH=.
python -m train.train --config configs/baseline.yaml
```

## Profiling

```bash
export PYTHONPATH=.
python -m profile.run_profile --config configs/baseline.yaml --out outputs/traces/profile.json
```

在 Chrome 打开 `chrome://tracing` 并加载导出的 JSON。

## 实验脚本

```bash
./scripts/run_all_experiments.sh
```

各实验对应 `experiments/` 与 `configs/`（`dataloader_tuned` 提高 `num_workers`，`pin_memory_off` 关闭 `pin_memory`，`amp` / `compile` 等）。

## 指标

`train.py` 每个 epoch 打印 `wall_sec`、`samples_per_sec`、`val_acc`；可用 `--output-json` 保存汇总供 `scripts/plot_results.py` 作图。
