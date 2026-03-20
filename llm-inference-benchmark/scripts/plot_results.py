#!/usr/bin/env python3
"""Placeholder: load benchmark JSON summaries and plot (add matplotlib as needed)."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("inputs", nargs="+", type=Path, help="JSON files from run_benchmark --output-json")
    args = p.parse_args()
    for path in args.inputs:
        data = json.loads(path.read_text(encoding="utf-8"))
        print(path.name, data)


if __name__ == "__main__":
    main()
