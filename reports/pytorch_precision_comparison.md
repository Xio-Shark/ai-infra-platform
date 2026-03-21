# PyTorch 推理精度-速度对比报告

## 测试环境

- **GPU**: NVIDIA GeForce RTX 4050 Laptop GPU (6140 MiB)
- **PyTorch**: 2.5.1+cu121
- **模型**: Qwen/Qwen2.5-1.5B (Hugging Face Transformers)
- **推理方式**: `model.generate()` greedy decode, 无额外优化
- **Prompts**: 10 次推理, max_new_tokens=64

## 对比结果

| 指标 | fp32 | fp16 | 变化 |
|------|------|------|------|
| **Avg Latency** | 42,368 ms | 4,061 ms | **10.4x faster** |
| **P50 Latency** | 45,014 ms | 3,965 ms | **11.4x faster** |
| **P95 Latency** | 50,723 ms | 4,965 ms | **10.2x faster** |
| **Avg Tokens/s** | 1.54 | 15.86 | **10.3x** |
| **GPU Memory** | 5,903 MB | 2,957 MB | **-50%** |

## 分析

### fp32 的问题

- **几乎占满 6GB VRAM**（5,903 / 6,140 = 96%），Qwen2.5-1.5B fp32 权重 ~6GB，RTX 4050 Laptop 刚好能放下
- decode 速度极慢（1.54 tok/s），因为 fp32 矩阵乘法无法利用 RTX 4050 的 Tensor Core（Tensor Core 只支持 fp16/bf16/int8）
- 对于 6GB 笔记本 GPU，fp32 推理**不可用**

### fp16 的优势

- **内存减半**：2,957MB，留出 3GB 空间给 KV-cache 和 batch
- **速度 10x**：15.86 tok/s，因为 RTX 4050 Tensor Core 对 fp16 有原生加速
- **P50/P95 稳定**：标准差小，抖动可控

### 与 Ollama 对比

| 框架 | 精度 | Avg Tokens/s | 说明 |
|------|------|-------------|------|
| PyTorch (native) | fp32 | 1.54 | Tensor Core 未启用 |
| PyTorch (native) | fp16 | 15.86 | Tensor Core 加速 |
| Ollama | Q4_K_M (量化) | 48.65 | 量化 + llama.cpp 优化 |

- Ollama 使用 Q4_K_M 量化（~4bit），比 fp16 更激进，速度 3x（48.65 vs 15.86）
- 说明**量化是移动端 / 笔记本推理的关键优化**，fp16 是数据中心 GPU 的标配

## 结论

1. **fp16 vs fp32**：在 RTX 4050 Laptop 上，fp16 推理速度是 fp32 的 **10.4x**，内存减半——fp32 在笔记本 GPU 上**不可用**
2. **fp16 vs 量化（Ollama）**：Ollama Q4_K_M 量化进一步加速 3x（48.65 vs 15.86 tok/s），说明 4-bit 量化在端侧部署上有显著优势
3. **生产环境建议**：数据中心使用 fp16 / bf16 + Tensor Core；端侧部署使用 GPTQ / AWQ 4-bit 量化

## 原始数据

- JSON: [`pytorch_bench_20260322_033909.json`](generated/pytorch_bench_20260322_033909.json)
