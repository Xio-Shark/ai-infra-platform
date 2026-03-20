from __future__ import annotations

import time
import torch
import torch.nn.functional as F
from torch import nn
from torch import amp as torch_amp
from torch.utils.data import DataLoader
from tqdm import tqdm


def train_one_epoch(
    model: nn.Module,
    loader: DataLoader,
    optimizer: torch.optim.Optimizer,
    device: torch.device,
    use_amp: bool,
    scaler: torch_amp.GradScaler | None,
) -> dict[str, float]:
    model.train()
    t0 = time.perf_counter()
    total_loss = 0.0
    n = 0
    for x, y in tqdm(loader, desc="train", leave=False):
        x = x.to(device, non_blocking=True)
        y = y.to(device, non_blocking=True)
        optimizer.zero_grad(set_to_none=True)
        if use_amp and scaler is not None and device.type == "cuda":
            with torch_amp.autocast(device_type="cuda", dtype=torch.float16):
                logits = model(x)
                loss = F.cross_entropy(logits, y)
            scaler.scale(loss).backward()
            scaler.step(optimizer)
            scaler.update()
        else:
            logits = model(x)
            loss = F.cross_entropy(logits, y)
            loss.backward()
            optimizer.step()
        total_loss += float(loss.detach()) * x.size(0)
        n += x.size(0)
    dt = time.perf_counter() - t0
    return {
        "loss": total_loss / max(1, n),
        "wall_sec": dt,
        "samples_per_sec": n / dt if dt > 0 else 0.0,
    }


@torch.no_grad()
def evaluate(model: nn.Module, loader: DataLoader, device: torch.device) -> float:
    model.eval()
    correct = 0
    total = 0
    for x, y in loader:
        x = x.to(device, non_blocking=True)
        y = y.to(device, non_blocking=True)
        pred = model(x).argmax(dim=1)
        correct += int((pred == y).sum().item())
        total += y.numel()
    return correct / max(1, total)
