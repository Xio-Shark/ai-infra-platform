from __future__ import annotations

import json
import sys
from typing import Any, TextIO


def print_summary(summary: dict[str, Any], out: TextIO = sys.stdout) -> None:
    for k, v in summary.items():
        if isinstance(v, float):
            out.write(f"{k}: {v:.4f}\n")
        else:
            out.write(f"{k}: {v}\n")


def write_json(path: str, data: dict[str, Any]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)
