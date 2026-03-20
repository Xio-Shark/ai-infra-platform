from __future__ import annotations

import json
import random
from pathlib import Path


def prompt_of_length(chars: int, seed: int | None = None) -> str:
    rng = random.Random(seed)
    alphabet = "abcdefghijklmnopqrstuvwxyz0123456789 "
    out: list[str] = []
    n = max(8, chars)
    while len("".join(out)) < n:
        out.append(rng.choice(alphabet))
    return "".join(out)[:n]


def load_jsonl_prompts(path: Path, limit: int | None = None) -> list[str]:
    lines: list[str] = []
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                if isinstance(obj, dict) and "text" in obj:
                    lines.append(str(obj["text"]))
                elif isinstance(obj, dict) and "prompt" in obj:
                    lines.append(str(obj["prompt"]))
                else:
                    lines.append(line)
            except json.JSONDecodeError:
                lines.append(line)
            if limit is not None and len(lines) >= limit:
                break
    return lines
