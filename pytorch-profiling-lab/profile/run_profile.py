from __future__ import annotations

import argparse
from itertools import cycle
from pathlib import Path

import torch
from torch.profiler import ProfilerActivity, profile

from profile.trace_handler import chrome_schedule, export_chrome_json
from train.utils import build_model, cifar10_loaders, get_device, load_config


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="configs/baseline.yaml")
    ap.add_argument("--steps", type=int, default=8)
    ap.add_argument("--out", type=Path, default=Path("outputs/traces/profile.json"))
    args = ap.parse_args()
    cfg = load_config(args.config)
    device = get_device()
    train_loader, _ = cifar10_loaders(
        cfg["data_dir"],
        cfg["batch_size"],
        cfg["num_workers"],
        cfg["pin_memory"],
    )
    model = build_model(cfg.get("model", "resnet18"), device)
    model.train()
    opt = torch.optim.SGD(model.parameters(), lr=float(cfg["lr"]), momentum=0.9)
    use_amp = bool(cfg.get("amp")) and device.type == "cuda"
    scaler = torch.amp.GradScaler("cuda", enabled=use_amp) if device.type == "cuda" else None

    activities = [ProfilerActivity.CPU]
    if device.type == "cuda":
        activities.append(ProfilerActivity.CUDA)

    it = cycle(train_loader)
    with profile(
        activities=activities,
        schedule=chrome_schedule(wait=0, warmup=1, active=3, repeat=1),
        on_trace_ready=lambda p: export_chrome_json(p, args.out),
        record_shapes=True,
        profile_memory=True,
        with_stack=True,
    ) as prof:
        for step in range(args.steps):
            x, y = next(it)
            x = x.to(device, non_blocking=True)
            y = y.to(device, non_blocking=True)
            opt.zero_grad(set_to_none=True)
            if use_amp and scaler is not None:
                with torch.amp.autocast(device_type="cuda", dtype=torch.float16):
                    logits = model(x)
                    loss = torch.nn.functional.cross_entropy(logits, y)
                scaler.scale(loss).backward()
                scaler.step(opt)
                scaler.update()
            else:
                logits = model(x)
                loss = torch.nn.functional.cross_entropy(logits, y)
                loss.backward()
                opt.step()
            prof.step()

    print(f"Chrome trace: {args.out.resolve()}")
    sort_key = "cuda_time_total" if device.type == "cuda" else "cpu_time_total"
    try:
        print(prof.key_averages().table(sort_by=sort_key, row_limit=12))
    except Exception:
        print(prof.key_averages().table(sort_by="self_cpu_time_total", row_limit=12))


if __name__ == "__main__":
    main()
