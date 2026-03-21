# Ollama vs vLLM 推理引擎对比报告

## 测试环境

- **GPU**: NVIDIA GeForce RTX 4050 Laptop GPU (6141 MiB)
- **模型**: Qwen2.5-1.5B
- **Ollama**: v0.18.1, 量化 Q4_K_M, 单推理线程
- **vLLM**: latest (Docker), fp16, continuous batching, CUDA graphs, FlashAttention
- **压测工具**: benchctl (Go goroutine 并发池)
- **max_tokens**: 64

## 并发扩展性对比

| 并发 (c) | Ollama QPS | vLLM QPS | Ollama tok/s | vLLM tok/s | Ollama P50 | vLLM P50 |
|----------|-----------|---------|-------------|-----------|-----------|---------|
| **3** | 2.48 | 1.44 | 48.65 | 30.09 | 1,162 ms | 2,126 ms |
| **5** | ~2.4* | **3.10** | ~48* | **40.33** | ~1,200* | **1,529 ms** |
| **10** | 2.36 | **6.51** | 48.18 | **41.10** | 4,137 ms | **1,447 ms** |
| **20** | 2.48 | — | 49.04 | — | 7,946 ms | — |

> *Ollama c=5 数据为估算值（基于 c=3 和 c=10 的线性趋势）

## 关键发现

### 1. QPS 扩展性（最核心差异）

```
QPS
 7 ┊                           ╱ vLLM (continuous batching)
 6 ┊                         ╱
 5 ┊                       ╱
 4 ┊                     ╱
 3 ┊                   ╱
 2 ┊─────────────────── Ollama (serial decode)
 1 ┊
 0 ┊───┬───┬───┬───┬───
   c=1  c=3  c=5  c=10 c=20
```

- **Ollama QPS 完全不随并发增长**：c=3 到 c=20，QPS 始终 2.36–2.48，因为 Ollama 对并发请求串行 decode
- **vLLM QPS 随并发线性增长**：c=3 → 1.44，c=5 → 3.10（+115%），c=10 → 6.51（+110%），continuous batching 实现并发扩展

### 2. 延迟表现

- **低并发 (c=3)**：Ollama 胜（P50 1,162ms vs 2,126ms），因为 Q4_K_M 量化的单请求速度更快
- **高并发 (c=10)**：**vLLM 胜**（P50 1,447ms vs 4,137ms），Ollama 因排队导致延迟线性膨胀，vLLM 通过 batching 保持延迟稳定

### 3. 每请求吞吐 (tokens/s)

- Ollama 单请求 48.65 tok/s（Q4_K_M 量化加速）
- vLLM 单请求 30.09 tok/s（fp16 精度更高，但量化优势消失）
- **但 vLLM 在 c=10 时 总吞吐 = 6.51 QPS × 64 tok = 416 tok/s（系统级），远超 Ollama 的 2.36 × 64 = 151 tok/s**

### 4. 瓶颈定位确认

| 瓶颈类型 | Ollama | vLLM |
|----------|--------|------|
| 并发调度 | ❌ 串行 decode，1 个请求占独占 GPU | ✅ continuous batching，多请求共享 GPU |
| GPU SM 利用率 | 46-54%（nvidia-smi 实测） | 更高（batching 填满 SM pipeline） |
| 量化加速 | ✅ Q4_K_M 4-bit（单请求快） | ❌ fp16（精度换速度） |
| 高并发 QPS 扩展 | ❌ 2.4 固定 | ✅ 线性增长 |

## 结论

1. **Ollama 适合单用户低并发场景**（demos / 个人使用），Q4_K_M 量化让单请求速度极快（48.65 tok/s）
2. **vLLM 适合多用户高并发生产环境**，continuous batching 让 QPS 随并发线性增长，系统级吞吐远超 Ollama
3. **GPU 资源利用率**：Ollama 浪费 ~50% SM 算力（单线程），vLLM 通过 batching 充分利用 GPU Pipeline
4. **生产建议**：推理服务部署应选 vLLM / SGLang + fp16 / bf16，端侧轻量推理可用 Ollama + 量化

## 原始数据

- vLLM c=3: [`bench_20260322_040235.json`](generated/bench_20260322_040235.json)
- vLLM c=5: [`bench_20260322_040302.json`](generated/bench_20260322_040302.json)
- vLLM c=10: [`bench_20260322_040334.json`](generated/bench_20260322_040334.json)
- Ollama c=3 (GPU): [`bench_20260322_025612.json`](generated/bench_20260322_025612.json)
