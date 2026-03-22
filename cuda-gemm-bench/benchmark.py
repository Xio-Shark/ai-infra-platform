#!/usr/bin/env python3
"""
CUDA GEMM Benchmark 自动化运行 + 报告生成
用法: python3 benchmark.py [--M 4096] [--N 4096] [--K 4096] [--warmup 5] [--iters 20]

功能:
  1. 编译三个 CUDA kernel
  2. 依次运行并捕获 JSON 输出
  3. 生成对比表格 + Markdown 报告
"""

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime


def run_command(cmd: str, desc: str) -> str:
    """执行命令并返回输出"""
    print(f"\n{'='*60}")
    print(f"  {desc}")
    print(f"  $ {cmd}")
    print(f"{'='*60}")
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"  [错误] 返回码 {result.returncode}")
        print(f"  stderr: {result.stderr}")
        return ""
    print(result.stdout)
    return result.stdout


def extract_json(output: str) -> dict:
    """从程序输出中提取 JSON 行"""
    for line in output.strip().split("\n"):
        line = line.strip()
        if line.startswith("{") and line.endswith("}"):
            try:
                return json.loads(line)
            except json.JSONDecodeError:
                continue
    return {}


def get_gpu_info() -> str:
    """获取 GPU 型号"""
    try:
        result = subprocess.run(
            "nvidia-smi --query-gpu=name --format=csv,noheader",
            shell=True, capture_output=True, text=True
        )
        return result.stdout.strip().split("\n")[0]
    except Exception:
        return "Unknown"


def main():
    parser = argparse.ArgumentParser(description="CUDA GEMM Benchmark")
    parser.add_argument("--M", type=int, default=4096, help="矩阵行数")
    parser.add_argument("--N", type=int, default=4096, help="矩阵列数")
    parser.add_argument("--K", type=int, default=4096, help="内积维度")
    parser.add_argument("--warmup", type=int, default=5, help="预热迭代次数")
    parser.add_argument("--iters", type=int, default=20, help="测量迭代次数")
    args = parser.parse_args()

    gpu_name = get_gpu_info()
    print(f"\nGPU: {gpu_name}")
    print(f"矩阵大小: {args.M}×{args.K} × {args.K}×{args.N} (FP16)")
    print(f"Warmup: {args.warmup}, Benchmark: {args.iters} 次")

    # 1. 编译
    run_command("make all", "编译 CUDA kernels")

    # 2. 运行三个 kernel
    kernels = [
        ("naive",  f"./naive_gemm {args.M} {args.N} {args.K} {args.warmup} {args.iters}"),
        ("tiled",  f"./tiled_gemm {args.M} {args.N} {args.K} {args.warmup} {args.iters}"),
        ("cublas", f"./cublas_gemm {args.M} {args.N} {args.K} {args.warmup} {args.iters}"),
    ]

    results = []
    for name, cmd in kernels:
        output = run_command(cmd, f"Benchmark: {name}")
        data = extract_json(output)
        if data:
            results.append(data)
        else:
            print(f"  [警告] 未能从 {name} 输出中提取 JSON")

    if not results:
        print("\n[错误] 没有获得任何 benchmark 结果")
        sys.exit(1)

    # 3. 保存 JSON 结果
    output_data = {
        "gpu": gpu_name,
        "matrix_size": {"M": args.M, "N": args.N, "K": args.K},
        "dtype": "FP16",
        "warmup": args.warmup,
        "iters": args.iters,
        "timestamp": datetime.now().isoformat(),
        "results": results,
    }

    with open("results.json", "w") as f:
        json.dump(output_data, f, indent=2, ensure_ascii=False)
    print(f"\n结果已保存至 results.json")

    # 4. 生成 Markdown 报告
    naive_r  = next((r for r in results if r.get("kernel") == "naive"), {})
    tiled_r  = next((r for r in results if r.get("kernel") == "tiled"), {})
    cublas_r = next((r for r in results if r.get("kernel") == "cublas"), {})

    # 计算加速比
    naive_ms = naive_r.get("avg_ms", 1)
    tiled_speedup = naive_ms / tiled_r.get("avg_ms", 1) if tiled_r else 0
    cublas_speedup = naive_ms / cublas_r.get("avg_ms", 1) if cublas_r else 0

    report = f"""# CUDA GEMM 性能阶梯实验报告

## 实验配置

| 项目 | 值 |
|---|---|
| GPU | {gpu_name} |
| 矩阵大小 | {args.M}×{args.K} × {args.K}×{args.N} |
| 数据类型 | FP16 (half) |
| Warmup | {args.warmup} 次 |
| 测量 | {args.iters} 次取平均 |
| 日期 | {datetime.now().strftime('%Y-%m-%d')} |

## 性能对比

| Kernel | 平均耗时 (ms) | TFLOPS | 效率 (vs 理论峰值) | vs Naive 加速比 |
|---|---|---|---|---|
| **Naive** (Global Mem) | {naive_r.get('avg_ms', 'N/A'):.2f} | {naive_r.get('tflops', 0):.2f} | {naive_r.get('tflops', 0)/312*100:.1f}% | 1.0× |
| **Tiled** (Shared Mem) | {tiled_r.get('avg_ms', 'N/A'):.2f} | {tiled_r.get('tflops', 0):.2f} | {tiled_r.get('tflops', 0)/312*100:.1f}% | {tiled_speedup:.1f}× |
| **cuBLAS** (Tensor Core) | {cublas_r.get('avg_ms', 'N/A'):.2f} | {cublas_r.get('tflops', 0):.2f} | {cublas_r.get('tflops', 0)/312*100:.1f}% | {cublas_speedup:.1f}× |

## 关键发现

### 1. Naive → Tiled: Shared Memory 优化 ({tiled_speedup:.1f}× 加速)

- Naive kernel 每个线程从 Global Memory 读取 2K 次（A 的一行 + B 的一列）
- Tiled kernel 通过 32×32 分块，将 Global Mem 访问减少 ~32 倍
- 瓶颈从 **内存带宽** 转移到 **计算吞吐**

### 2. Tiled → cuBLAS: Tensor Core 加速 ({cublas_speedup/tiled_speedup:.1f}× 进一步加速)

- cuBLAS 使用 A100 的第三代 Tensor Core（FP16 → FP16 with FP32 accumulate）
- 内部优化包括：双缓冲 prefetch、warp-level MMA 指令、自动 tile 尺寸调优
- 接近理论峰值说明 cuBLAS 对 A100 架构高度调优

### 3. 性能瓶颈转移规律

```
Naive: Memory Bound (Global Mem 带宽受限)
  ↓ Shared Memory Tiling
Tiled: Compute Bound (ALU 利用率上升但无 Tensor Core)
  ↓ Tensor Core + 极致优化
cuBLAS: 接近理论峰值
```

## 面试要点

1. **为什么 Tiled 比 Naive 快？** → Shared Mem 数据复用，减少 Global Mem 访问
2. **为什么 cuBLAS 比 Tiled 快？** → Tensor Core 硬件加速 + warp-level MMA + 双缓冲
3. **什么是 Memory Coalescing？** → 相邻线程访问连续内存地址，合并为一次传输
4. **Occupancy 和性能的关系？** → 更多 active warps 可以隐藏内存延迟
"""

    with open("report.md", "w", encoding="utf-8") as f:
        f.write(report)
    print(f"报告已保存至 report.md")

    # 5. 打印摘要
    print(f"\n{'='*60}")
    print(f"  GEMM 性能阶梯总结 ({gpu_name})")
    print(f"{'='*60}")
    for r in results:
        print(f"  {r.get('kernel', '?'):>8s}: {r.get('tflops', 0):8.2f} TFLOPS  ({r.get('avg_ms', 0):8.2f} ms)")
    print(f"{'='*60}")


if __name__ == "__main__":
    main()
