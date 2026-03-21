"""
PyTorch Inference Benchmark: fp32 vs fp16 精度-速度对比
模型: Qwen2.5-1.5B (通过 Hugging Face Transformers 加载)
硬件: RTX 4050 Laptop GPU (6141 MiB)

运行方式:
  pip install torch transformers accelerate
  python scripts/pytorch_inference_bench.py
"""

import json
import time
import sys
import os
from datetime import datetime

def check_deps():
    """检查依赖是否安装"""
    try:
        import torch
        import transformers
        return True
    except ImportError as e:
        print(f"缺少依赖: {e}")
        print("安装: pip install torch transformers accelerate")
        return False

def get_device_info():
    """获取设备信息"""
    import torch
    info = {
        "pytorch_version": torch.__version__,
        "cuda_available": torch.cuda.is_available(),
    }
    if torch.cuda.is_available():
        info["gpu_name"] = torch.cuda.get_device_name(0)
        info["gpu_memory_mb"] = round(torch.cuda.get_device_properties(0).total_memory / (1024 * 1024))
    return info

def run_inference(model, tokenizer, prompts, max_new_tokens=64, dtype_label="fp32"):
    """运行推理并采集指标"""
    import torch
    results = []
    
    for i, prompt in enumerate(prompts):
        inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
        input_len = inputs["input_ids"].shape[1]
        
        # GPU 预热（第一次推理可能慢）
        if i == 0:
            with torch.no_grad():
                _ = model.generate(**inputs, max_new_tokens=8)
            if torch.cuda.is_available():
                torch.cuda.synchronize()
        
        # 正式推理
        start = time.perf_counter()
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=max_new_tokens,
                do_sample=False,  # greedy decode 保证可复现
            )
        if torch.cuda.is_available():
            torch.cuda.synchronize()
        elapsed = time.perf_counter() - start
        
        output_tokens = outputs.shape[1] - input_len
        tokens_per_sec = output_tokens / elapsed if elapsed > 0 else 0
        
        results.append({
            "prompt_idx": i,
            "input_tokens": input_len,
            "output_tokens": output_tokens,
            "latency_ms": round(elapsed * 1000, 2),
            "tokens_per_sec": round(tokens_per_sec, 2),
            "dtype": dtype_label,
        })
        
        print(f"  [{dtype_label}] prompt {i+1}/{len(prompts)}: "
              f"{elapsed*1000:.0f}ms, {output_tokens} tokens, {tokens_per_sec:.1f} tok/s")
    
    return results

def get_gpu_memory_mb():
    """获取当前 GPU 显存使用量"""
    import torch
    if torch.cuda.is_available():
        return torch.cuda.max_memory_allocated() // (1024 * 1024)
    return 0

def summarize(results):
    """汇总指标"""
    latencies = [r["latency_ms"] for r in results]
    tps = [r["tokens_per_sec"] for r in results]
    latencies.sort()
    n = len(latencies)
    return {
        "count": n,
        "avg_latency_ms": round(sum(latencies) / n, 2),
        "p50_latency_ms": round(latencies[n // 2], 2),
        "p95_latency_ms": round(latencies[int(n * 0.95)], 2) if n >= 20 else round(latencies[-1], 2),
        "avg_tokens_per_sec": round(sum(tps) / n, 2),
        "max_gpu_memory_mb": get_gpu_memory_mb(),
    }

def main():
    if not check_deps():
        sys.exit(1)
    
    import torch
    from transformers import AutoTokenizer, AutoModelForCausalLM
    
    model_name = "Qwen/Qwen2.5-1.5B"
    max_new_tokens = 64
    prompts = [
        "Hello, tell me a joke about programming.",
        "What is the difference between a process and a thread?",
        "Explain how a GPU accelerates deep learning inference.",
        "What is KV-cache in transformer inference?",
        "Describe the concept of continuous batching in LLM serving.",
    ] * 2  # 10 次推理
    
    device_info = get_device_info()
    print(f"\n{'='*60}")
    print(f"PyTorch Inference Benchmark")
    print(f"Model: {model_name}")
    print(f"Device: {device_info}")
    print(f"Prompts: {len(prompts)}, Max tokens: {max_new_tokens}")
    print(f"{'='*60}\n")
    
    results_all = {}
    
    # === FP32 ===
    print("[1/2] Loading model in fp32...")
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    
    device = "cuda" if torch.cuda.is_available() else "cpu"
    model_fp32 = AutoModelForCausalLM.from_pretrained(
        model_name, torch_dtype=torch.float32
    ).to(device)
    model_fp32.eval()
    
    if torch.cuda.is_available():
        torch.cuda.reset_peak_memory_stats()
    
    print(f"  Model loaded. GPU memory: {get_gpu_memory_mb()} MB")
    fp32_results = run_inference(model_fp32, tokenizer, prompts, max_new_tokens, "fp32")
    fp32_summary = summarize(fp32_results)
    fp32_summary["dtype"] = "fp32"
    results_all["fp32"] = fp32_summary
    
    del model_fp32
    if torch.cuda.is_available():
        torch.cuda.empty_cache()
    
    # === FP16 ===
    print("\n[2/2] Loading model in fp16...")
    if torch.cuda.is_available():
        torch.cuda.reset_peak_memory_stats()
    
    model_fp16 = AutoModelForCausalLM.from_pretrained(
        model_name, torch_dtype=torch.float16
    ).to(device)
    model_fp16.eval()
    
    print(f"  Model loaded. GPU memory: {get_gpu_memory_mb()} MB")
    fp16_results = run_inference(model_fp16, tokenizer, prompts, max_new_tokens, "fp16")
    fp16_summary = summarize(fp16_results)
    fp16_summary["dtype"] = "fp16"
    results_all["fp16"] = fp16_summary
    
    del model_fp16
    if torch.cuda.is_available():
        torch.cuda.empty_cache()
    
    # === 输出报告 ===
    print(f"\n{'='*60}")
    print("SUMMARY")
    print(f"{'='*60}")
    
    report = {
        "timestamp": datetime.now().isoformat(),
        "model": model_name,
        "device": device_info,
        "max_new_tokens": max_new_tokens,
        "num_prompts": len(prompts),
        "results": results_all,
    }
    
    # 输出对比表格
    print(f"\n{'dtype':<8} {'Avg Latency':>12} {'P50 Latency':>12} {'Avg tok/s':>10} {'GPU Mem':>10}")
    print("-" * 56)
    for dtype in ["fp32", "fp16"]:
        r = results_all[dtype]
        print(f"{dtype:<8} {r['avg_latency_ms']:>10.0f}ms {r['p50_latency_ms']:>10.0f}ms "
              f"{r['avg_tokens_per_sec']:>8.1f} {r['max_gpu_memory_mb']:>8}MB")
    
    # 计算加速比
    if results_all["fp32"]["avg_latency_ms"] > 0:
        speedup = results_all["fp32"]["avg_latency_ms"] / results_all["fp16"]["avg_latency_ms"]
        mem_save = 1 - results_all["fp16"]["max_gpu_memory_mb"] / max(results_all["fp32"]["max_gpu_memory_mb"], 1)
        print(f"\nfp16 vs fp32: Latency {speedup:.2f}x faster, Memory {mem_save*100:.0f}% saved")
    
    # 保存 JSON
    out_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "reports", "generated")
    os.makedirs(out_dir, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    json_path = os.path.join(out_dir, f"pytorch_bench_{ts}.json")
    with open(json_path, "w") as f:
        json.dump(report, f, indent=2)
    print(f"\nJSON report: {json_path}")

if __name__ == "__main__":
    main()
