from __future__ import annotations

from dataclasses import dataclass, field


def _percentile(sorted_vals: list[float], p: float) -> float:
    if not sorted_vals:
        return 0.0
    if len(sorted_vals) == 1:
        return sorted_vals[0]
    k = (len(sorted_vals) - 1) * p / 100.0
    f = int(k)
    c = min(f + 1, len(sorted_vals) - 1)
    if f == c:
        return sorted_vals[f]
    return sorted_vals[f] + (sorted_vals[c] - sorted_vals[f]) * (k - f)


@dataclass
class RequestStats:
    latencies: list[float] = field(default_factory=list)
    ttfts: list[float] = field(default_factory=list)
    tokens_per_sec: list[float] = field(default_factory=list)
    successes: int = 0
    failures: int = 0

    def add(
        self,
        ok: bool,
        latency_sec: float,
        ttft_sec: float | None = None,
        tps: float | None = None,
    ) -> None:
        if ok:
            self.successes += 1
            self.latencies.append(latency_sec)
            if ttft_sec is not None:
                self.ttfts.append(ttft_sec)
            if tps is not None:
                self.tokens_per_sec.append(tps)
        else:
            self.failures += 1

    def summary(self) -> dict[str, float | int]:
        lat = sorted(self.latencies)
        tt = sorted(self.ttfts)
        tps = sorted(self.tokens_per_sec)
        total = self.successes + self.failures
        return {
            "requests_total": total,
            "successes": self.successes,
            "failures": self.failures,
            "success_rate": self.successes / total if total else 0.0,
            "latency_p50_ms": _percentile(lat, 50) * 1000,
            "latency_p95_ms": _percentile(lat, 95) * 1000,
            "latency_p99_ms": _percentile(lat, 99) * 1000,
            "ttft_p50_ms": _percentile(tt, 50) * 1000 if tt else 0.0,
            "ttft_p95_ms": _percentile(tt, 95) * 1000 if tt else 0.0,
            "tokens_per_sec_p50": _percentile(tps, 50) if tps else 0.0,
        }
