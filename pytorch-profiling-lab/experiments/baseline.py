from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def main() -> None:
    env = {**os.environ, "PYTHONPATH": str(ROOT)}
    subprocess.run(
        [sys.executable, "-m", "train.train", "--config", str(ROOT / "configs/baseline.yaml")],
        cwd=ROOT,
        env=env,
        check=True,
    )


if __name__ == "__main__":
    main()
