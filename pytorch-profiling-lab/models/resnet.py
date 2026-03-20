from __future__ import annotations

import torch.nn as nn
from torchvision import models


def build_resnet18(num_classes: int = 10) -> nn.Module:
    m = models.resnet18(weights=None)
    m.fc = nn.Linear(m.fc.in_features, num_classes)
    return m
