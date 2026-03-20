from __future__ import annotations

import torch.nn as nn
from torchvision import models


def build_vit_b_16(num_classes: int = 10) -> nn.Module:
    m = models.vit_b_16(weights=None, image_size=32, patch_size=8)
    m.heads.head = nn.Linear(m.heads.head.in_features, num_classes)
    return m
