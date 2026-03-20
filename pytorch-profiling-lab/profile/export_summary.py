from __future__ import annotations

import argparse
import json
from pathlib import Path


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("trace", type=Path)
    p.add_argument("--out", type=Path, default=Path("outputs/tables/profiler_hint.json"))
    args = p.parse_args()
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(
        json.dumps({"trace": str(args.trace.resolve()), "hint": "Use TensorBoard or chrome tracing"}, indent=2),
        encoding="utf-8",
    )
    print(f"wrote {args.out}")


if __name__ == "__main__":
    main()
