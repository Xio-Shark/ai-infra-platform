#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

try:
    import matplotlib.pyplot as plt
except ImportError:
    plt = None


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("inputs", nargs="+", type=Path)
    p.add_argument("--out", type=Path, default=Path("outputs/figures/summary.png"))
    args = p.parse_args()
    labels = []
    speeds = []
    for path in args.inputs:
        data = json.loads(path.read_text(encoding="utf-8"))
        epochs = data.get("epochs") or []
        last = epochs[-1] if epochs else {}
        labels.append(path.stem)
        speeds.append(float(last.get("samples_per_sec", 0)))
    if plt is None:
        print("matplotlib not installed; printing:", dict(zip(labels, speeds)))
        return
    args.out.parent.mkdir(parents=True, exist_ok=True)
    plt.figure(figsize=(8, 4))
    plt.bar(labels, speeds)
    plt.ylabel("samples/sec (last epoch)")
    plt.xticks(rotation=30, ha="right")
    plt.tight_layout()
    plt.savefig(args.out)
    print(f"wrote {args.out}")


if __name__ == "__main__":
    main()
