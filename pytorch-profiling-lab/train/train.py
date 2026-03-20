from __future__ import annotations

import argparse
import json
from pathlib import Path

import torch
from torch import amp as torch_amp

from train.engine import evaluate, train_one_epoch
from train.utils import build_model, cifar10_loaders, get_device, load_config


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="configs/baseline.yaml")
    ap.add_argument("--output-json", default="")
    args = ap.parse_args()
    cfg = load_config(args.config)
    device = get_device()
    train_loader, test_loader = cifar10_loaders(
        cfg["data_dir"],
        cfg["batch_size"],
        cfg["num_workers"],
        cfg["pin_memory"],
    )
    model = build_model(cfg.get("model", "resnet18"), device)
    if cfg.get("compile_model"):
        model = torch.compile(model)  # type: ignore[assignment]
    opt = torch.optim.SGD(model.parameters(), lr=float(cfg["lr"]), momentum=0.9, weight_decay=5e-4)
    use_amp = bool(cfg.get("amp")) and device.type == "cuda"
    scaler = torch_amp.GradScaler("cuda", enabled=use_amp) if device.type == "cuda" else None

    summary: dict[str, object] = {"config": args.config, "device": str(device), "epochs": []}
    for epoch in range(int(cfg["epochs"])):
        stats = train_one_epoch(model, train_loader, opt, device, use_amp, scaler)
        acc = evaluate(model, test_loader, device)
        row = {"epoch": epoch, **stats, "val_acc": acc}
        summary["epochs"].append(row)  # type: ignore[union-attr]
        print(json.dumps(row))

    if args.output_json:
        Path(args.output_json).parent.mkdir(parents=True, exist_ok=True)
        Path(args.output_json).write_text(json.dumps(summary, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main()
