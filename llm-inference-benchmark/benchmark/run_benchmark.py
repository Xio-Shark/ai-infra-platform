from __future__ import annotations

import argparse
import asyncio
import json
import time
from pathlib import Path

import httpx
import yaml

from benchmark.metrics import RequestStats
from benchmark.reporter import print_summary, write_json
from benchmark.workload_generator import load_jsonl_prompts, prompt_of_length


def _parse_sse_ttft_and_tokens(
    body: bytes, start: float, end: float
) -> tuple[float | None, float | None]:
    """First non-empty delta content => TTFT; rough output tokens from content chunks."""
    text = body.decode("utf-8", errors="replace")
    ttft: float | None = None
    out_chars = 0
    for line in text.splitlines():
        if not line.startswith("data:"):
            continue
        payload = line[5:].strip()
        if payload == "[DONE]":
            break
        try:
            obj = json.loads(payload)
        except json.JSONDecodeError:
            continue
        choices = obj.get("choices") or []
        if not choices:
            continue
        delta = (choices[0].get("delta") or {}) if isinstance(choices[0], dict) else {}
        c = delta.get("content")
        if c:
            if ttft is None:
                ttft = time.perf_counter() - start
            out_chars += len(c)
    duration = end - start
    tps = (out_chars / 4) / duration if duration > 0 else None
    return ttft, tps


async def one_request(
    client: httpx.AsyncClient,
    url: str,
    model: str,
    prompt: str,
    max_tokens: int,
    stream: bool,
) -> tuple[bool, float, float | None, float | None]:
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": max_tokens,
        "stream": stream,
    }
    t0 = time.perf_counter()
    try:
        if stream:
            async with client.stream("POST", url, json=payload) as resp:
                if resp.status_code >= 400:
                    await resp.aread()
                    return False, time.perf_counter() - t0, None, None
                chunks: list[bytes] = []
                async for c in resp.aiter_bytes():
                    chunks.append(c)
                t1 = time.perf_counter()
                ttft, tps = _parse_sse_ttft_and_tokens(b"".join(chunks), t0, t1)
                return True, t1 - t0, ttft, tps
        resp = await client.post(url, json=payload)
        ok = resp.status_code < 400
        t1 = time.perf_counter()
        tps = None
        if ok:
            try:
                data = resp.json()
                u = data.get("usage") or {}
                comp = int(u.get("completion_tokens") or 0)
                if comp > 0 and (t1 - t0) > 0:
                    tps = comp / (t1 - t0)
            except Exception:
                pass
        return ok, t1 - t0, None if stream else None, tps
    except Exception:
        return False, time.perf_counter() - t0, None, None


async def worker(
    wid: int,
    n: int,
    url: str,
    model: str,
    max_tokens: int,
    stream: bool,
    prompts: list[str],
    stats: RequestStats,
    timeout: float,
) -> None:
    limits = httpx.Limits(max_connections=100, max_keepalive_connections=50)
    async with httpx.AsyncClient(timeout=httpx.Timeout(timeout), limits=limits) as client:
        for i in range(n):
            p = prompts[(wid * n + i) % len(prompts)]
            ok, lat, ttft, tps = await one_request(
                client, url, model, p, max_tokens, stream
            )
            stats.add(ok, lat, ttft, tps)


async def run(cfg: dict) -> RequestStats:
    base = cfg["base_url"].rstrip("/")
    ep = cfg["endpoint"]
    url = f"{base}{ep}"
    conc = int(cfg["concurrency"])
    total = int(cfg["total_requests"])
    per = max(1, (total + conc - 1) // conc)
    model = cfg["model"]
    max_tokens = int(cfg["max_tokens"])
    stream = bool(cfg.get("stream", True))
    timeout = float(cfg.get("timeout_sec", 120))
    pch = cfg.get("prompt_chars")
    ds = cfg.get("dataset_path")

    if ds:
        prompts = load_jsonl_prompts(Path(ds))
    else:
        prompts = [prompt_of_length(int(pch or 256), seed=i) for i in range(256)]

    if not prompts:
        prompts = ["hello"]

    stats = RequestStats()
    tasks = []
    sent = 0
    for w in range(conc):
        take = min(per, total - sent)
        if take <= 0:
            break
        sent += take
        tasks.append(
            asyncio.create_task(
                worker(w, take, url, model, max_tokens, stream, prompts, stats, timeout)
            )
        )
    await asyncio.gather(*tasks)
    return stats


def main() -> None:
    ap = argparse.ArgumentParser(description="LLM OpenAI-compatible load generator")
    ap.add_argument("--config", default="configs/benchmark.yaml")
    ap.add_argument("--output-json", default="")
    args = ap.parse_args()
    with open(args.config, encoding="utf-8") as f:
        cfg = yaml.safe_load(f)
    stats = asyncio.run(run(cfg))
    summary = stats.summary()
    print_summary(summary)
    if args.output_json:
        write_json(args.output_json, summary)


if __name__ == "__main__":
    main()
