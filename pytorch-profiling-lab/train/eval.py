from __future__ import annotations

import argparse

import torch

from train.engine import evaluate
from train.utils import build_model, cifar10_loaders, get_device, load_config


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="configs/baseline.yaml")
    args = ap.parse_args()
    cfg = load_config(args.config)
    device = get_device()
    _, test_loader = cifar10_loaders(
        cfg["data_dir"],
        cfg["batch_size"],
        cfg["num_workers"],
        cfg["pin_memory"],
    )
    model = build_model(cfg.get("model", "resnet18"), device)
    ckpt = cfg.get("checkpoint")
    if ckpt:
        model.load_state_dict(torch.load(ckpt, map_location=device))
    acc = evaluate(model, test_loader, device)
    print(f"accuracy: {acc:.4f}")


if __name__ == "__main__":
    main()
