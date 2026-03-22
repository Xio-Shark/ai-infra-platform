# CUDA GEMM 性能阶梯 Benchmark

从 Naive CUDA Kernel 到 cuBLAS Tensor Core，逐级优化矩阵乘法，量化验证 GPU 内存层次与计算单元的性能瓶颈转移。

## 三级实现

| 实现 | 优化要点 | 预期性能 (A100 FP16) |
|---|---|---|
| `naive_gemm.cu` | 每线程算一个元素，全走 Global Memory | ~2 TFLOPS |
| `tiled_gemm.cu` | Shared Memory 32×32 分块，数据复用 32× | ~30 TFLOPS |
| `cublas_gemm.cu` | cuBLAS `cublasHgemm`，Tensor Core 加速 | ~250 TFLOPS |

## 快速开始

```bash
# 在 A100 / 有 CUDA 的 GPU 机器上
make all                    # 编译
python3 benchmark.py        # 自动运行 + 生成报告

# 自定义矩阵大小
python3 benchmark.py --M 8192 --N 8192 --K 8192

# 单独运行某个 kernel
make bench_naive
make bench_tiled
make bench_cublas
```

## 产出

- `results.json` — 结构化 benchmark 数据
- `report.md` — 性能对比表格 + 分析 + 面试要点

## 知识点清单（面试必备）

1. **CUDA 线程模型**: Grid → Block → Thread, `threadIdx` / `blockIdx` / `blockDim`
2. **Global vs Shared Memory**: 带宽差异 ~10×，Shared Mem 作为 scratchpad 缓存
3. **Memory Coalescing**: 相邻线程访问连续地址 → 合并为一次 128B 事务
4. **Tiling 优化**: 减少 Global Mem 访问次数，从 O(K) 降到 O(K/TILE)
5. **`__syncthreads()`**: Block 内线程同步屏障，确保 Shared Mem 一致性
6. **Tensor Core**: FP16 矩阵乘加（MMA），4×4×4 或 16×16×16 warp-level 指令
7. **Occupancy**: Active warps / Max warps per SM，影响延迟隐藏能力

## 架构说明

```
naive_gemm.cu       ← 入门：理解 CUDA 编程模型
tiled_gemm.cu       ← 进阶：Shared Memory + 同步
cublas_gemm.cu      ← 参考：生产级 Tensor Core 上限
benchmark.py        ← 自动化：编译 + 运行 + 报告
Makefile            ← 构建：nvcc 编译选项
```

## 注意事项

- `Makefile` 默认 `-arch=sm_80`（A100），如在其他 GPU 上运行请修改：
  - RTX 4050/4060/4090: `sm_89`
  - RTX 3090: `sm_86`
  - V100: `sm_70`
