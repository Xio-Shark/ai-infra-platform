from __future__ import annotations

import argparse
from pathlib import Path


def main() -> None:
    p = argparse.ArgumentParser(description="Open chrome://tracing and load the exported JSON.")
    p.add_argument("trace", type=Path)
    args = p.parse_args()
    print(f"Load in Chrome tracing: {args.trace.resolve()}")


if __name__ == "__main__":
    main()
