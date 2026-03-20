from __future__ import annotations

from pathlib import Path

from torch.profiler import schedule


def chrome_schedule(wait: int = 1, warmup: int = 1, active: int = 2, repeat: int = 1):
    return schedule(wait=wait, warmup=warmup, active=active, repeat=repeat)


def export_chrome_json(prof, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    prof.export_chrome_trace(str(path))
