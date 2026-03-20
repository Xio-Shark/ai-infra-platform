from __future__ import annotations

import math

import torch
import torch.nn as nn


class PositionalEncoding(nn.Module):
    def __init__(self, d_model: int, max_len: int = 512) -> None:
        super().__init__()
        pe = torch.zeros(max_len, d_model)
        pos = torch.arange(0, max_len, dtype=torch.float).unsqueeze(1)
        div = torch.exp(torch.arange(0, d_model, 2).float() * (-math.log(10000.0) / d_model))
        pe[:, 0::2] = torch.sin(pos * div)
        pe[:, 1::2] = torch.cos(pos * div)
        self.register_buffer("pe", pe.unsqueeze(0))

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return x + self.pe[:, : x.size(1)]


class SmallTransformerClassifier(nn.Module):
    """Tiny encoder for flattened CIFAR-10 patches (demo only)."""

    def __init__(self, num_classes: int = 10, d_model: int = 128, nhead: int = 4, nlayers: int = 2) -> None:
        super().__init__()
        patch = 4
        self.patch = patch
        dim = 3 * patch * patch
        self.proj = nn.Linear(dim, d_model)
        self.pos = PositionalEncoding(d_model, max_len=128)
        enc = nn.TransformerEncoderLayer(
            d_model, nhead, dim_feedforward=d_model * 4, batch_first=True
        )
        self.encoder = nn.TransformerEncoder(enc, num_layers=nlayers)
        self.fc = nn.Linear(d_model, num_classes)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        b, _, h, w = x.shape
        p = self.patch
        x = x.unfold(2, p, p).unfold(3, p, p)
        x = x.contiguous().view(b, -1, 3 * p * p)
        z = self.proj(x)
        z = self.pos(z)
        z = self.encoder(z)
        return self.fc(z.mean(dim=1))
