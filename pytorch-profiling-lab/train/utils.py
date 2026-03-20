from __future__ import annotations

from pathlib import Path
from typing import Any

import torch
import yaml
from torch.utils.data import DataLoader
from torchvision import datasets, transforms


def load_config(path: str | Path) -> dict[str, Any]:
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)


def get_device() -> torch.device:
    return torch.device("cuda" if torch.cuda.is_available() else "cpu")


def cifar10_loaders(
    data_dir: str | Path,
    batch_size: int,
    num_workers: int,
    pin_memory: bool,
) -> tuple[DataLoader, DataLoader]:
    tfm = transforms.Compose(
        [
            transforms.RandomHorizontalFlip(),
            transforms.ToTensor(),
            transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2470, 0.2435, 0.2616)),
        ]
    )
    test_tfm = transforms.Compose(
        [
            transforms.ToTensor(),
            transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2470, 0.2435, 0.2616)),
        ]
    )
    root = Path(data_dir)
    root.mkdir(parents=True, exist_ok=True)
    train_ds = datasets.CIFAR10(str(root), train=True, download=True, transform=tfm)
    test_ds = datasets.CIFAR10(str(root), train=False, download=True, transform=test_tfm)
    train_loader = DataLoader(
        train_ds,
        batch_size=batch_size,
        shuffle=True,
        num_workers=num_workers,
        pin_memory=pin_memory,
        persistent_workers=num_workers > 0,
    )
    test_loader = DataLoader(
        test_ds,
        batch_size=batch_size,
        shuffle=False,
        num_workers=num_workers,
        pin_memory=pin_memory,
        persistent_workers=num_workers > 0,
    )
    return train_loader, test_loader


def build_model(name: str, device: torch.device) -> torch.nn.Module:
    from models import resnet, transformer_small, vit

    name = name.lower()
    if name == "resnet18":
        m = resnet.build_resnet18()
    elif name in ("vit", "vit_b_16"):
        m = vit.build_vit_b_16()
    elif name in ("small_transformer", "transformer"):
        m = transformer_small.SmallTransformerClassifier()
    else:
        raise ValueError(f"unknown model: {name}")
    return m.to(device)
