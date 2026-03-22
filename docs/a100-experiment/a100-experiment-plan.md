# A100 实验计划：推理蒸馏微调 + vLLM 部署 + Benchmark 对比 + HuggingFace 发布

> **预计时长**：5–7 小时（含环境搭建）  
> **硬件要求**：1 × NVIDIA A100 80GB SXM（云实例或实验室机器）  
> **产出物**：推理蒸馏模型、GGUF 量化版、benchmark JSON 报告、对比分析文档、HuggingFace 模型页

---

## 0. 总览时间线

| 阶段 | 所需时间 | 产出 |
|------|----------|------|
| Phase 0: 环境准备 | 30 min | conda 环境 + 依赖就绪 |
| Phase 1: 推理蒸馏微调 | 60–90 min | LoRA adapter + 合并模型 |
| Phase 2: vLLM 部署 | 20 min | OpenAI-compatible API online |
| Phase 3: benchctl 对比 | 60 min | JSON 结果 + 对比表（含推理效率指标） |
| Phase 4: 分析报告 | 30 min | Markdown 分析文档 |
| Phase 5: 量化 + 发布 | 30–45 min | GGUF 量化 + HuggingFace 发布 |
| **总计** | **~5.5 h** | |

---

## Phase 0：环境准备

### 0.1 创建 conda 环境

```bash
# 创建干净的 Python 3.11 环境
conda create -n a100-exp python=3.11 -y
conda activate a100-exp

# 验证 GPU
nvidia-smi
# 应显示 A100-SXM4-80GB, CUDA 12.x
```

### 0.2 安装核心依赖

```bash
# Unsloth（含 PyTorch + Triton + Flash Attention）
pip install unsloth

# vLLM
pip install vllm

# 数据集 & 通用工具
pip install datasets transformers accelerate huggingface_hub
pip install matplotlib pandas tabulate

# 可选：Weights & Biases 实验追踪
pip install wandb
```

### 0.3 登录 Hugging Face（下载 gated 模型或上传结果）

```bash
huggingface-cli login --token $HF_TOKEN
```

### 0.4 工作目录结构

```
a100-experiment/
├── scripts/
│   ├── 00_prepare_data.py      # 下载+格式化推理蒸馏数据集
│   ├── 01_finetune.py          # Unsloth QLoRA 蒸馏微调脚本
│   ├── 02_merge_and_export.py  # 合并 LoRA + 导出
│   ├── 03_serve_vllm.sh        # vLLM 启动脚本
│   ├── 04_bench_base.sh        # Base 模型 benchmark
│   ├── 05_bench_finetuned.sh   # 蒸馏模型 benchmark
│   ├── 06_compare.py           # 结果对比 & 可视化
│   └── 07_publish.sh           # GGUF 量化 + HuggingFace 发布
├── data/
│   └── reasoning_distill.jsonl # 推理蒸馏训练数据
├── outputs/
│   ├── lora-adapter/           # LoRA 权重
│   ├── merged-model/           # 合并后完整模型
│   ├── gguf/                   # GGUF 量化版本
│   └── benchmarks/             # benchmark 结果
└── report/
    └── analysis.md             # 最终分析报告
```

```bash
mkdir -p a100-experiment/{scripts,data,outputs/{lora-adapter,merged-model,gguf,benchmarks},report}
cd a100-experiment
```

---

## Phase 1：推理蒸馏微调

### 1.0 下载推理蒸馏数据集 `scripts/00_prepare_data.py`

使用 Claude Opus 蒸馏的公开推理数据集，而非通用指令数据——核心目标是让小模型学会**结构化推理思维链**（reasoning scaffold），使其"想得更短但更准"。

| 数据集 | 条数 | 说明 |
|--------|------|------|
| [nohurry/Opus-4.6-Reasoning-3000x-filtered](https://huggingface.co/datasets/nohurry/Opus-4.6-Reasoning-3000x-filtered) | ~3000 | Claude 4.6 Opus 推理链蒸馏，已过滤 |
| [TeichAI/claude-4.5-opus-high-reasoning-250x](https://huggingface.co/datasets/TeichAI/claude-4.5-opus-high-reasoning-250x) | ~250 | 高质量逻辑推理样本 |

```python
"""
下载并格式化推理蒸馏数据集
- 数据来源：公开 Claude Opus 推理蒸馏 dataset
- 输出格式：ChatML JSONL
"""

from datasets import load_dataset, concatenate_datasets
import json

# 加载推理蒸馏数据
ds_main = load_dataset("nohurry/Opus-4.6-Reasoning-3000x-filtered", split="train")
ds_extra = load_dataset("TeichAI/claude-4.5-opus-high-reasoning-250x", split="train")

# 合并
ds = concatenate_datasets([ds_main, ds_extra])

# 保存为 JSONL（保留原始 conversations 结构）
with open("data/reasoning_distill.jsonl", "w", encoding="utf-8") as f:
    for row in ds:
        f.write(json.dumps(dict(row), ensure_ascii=False) + "\n")

print(f"数据集准备完成：{len(ds)} 条推理蒸馏样本")
```

```bash
python scripts/00_prepare_data.py
```

> **为什么选推理蒸馏而非通用指令？**
> - 通用指令微调（Alpaca）人人都做过，没有差异化价值
> - 推理蒸馏的核心是**迁移大模型的思维结构**（reasoning scaffold），可以量化验证"想得更短但更准"
> - 与你已有的 Ollama vs vLLM 实验形成 **"训练→推理→Benchmark" 完整闭环**

### 1.1 准备训练数据

数据已通过上一步下载到 `data/reasoning_distill.jsonl`，格式为 conversations 结构（与 ChatML 兼容）。

> **推荐规模**：3000+ 条高质量推理蒸馏样本

### 1.2 微调脚本 `scripts/01_finetune.py`

```python
"""
Unsloth QLoRA fine-tuning on A100 80GB
- 基模型：Qwen/Qwen2.5-7B-Instruct（或换成你的目标模型）
- 方法：4-bit QLoRA
- 数据：Claude Opus 推理蒸馏数据集
- 预计耗时：40–60 min（3000 条数据，3 epochs）
"""

from unsloth import FastLanguageModel
from unsloth import is_bfloat16_supported
from trl import SFTTrainer
from transformers import TrainingArguments
from datasets import load_dataset
import torch

# ============ 配置区 ============
MODEL_NAME = "Qwen/Qwen2.5-7B-Instruct"
MAX_SEQ_LENGTH = 4096          # 可调至 8192（A100 内存充足）
LORA_R = 32                    # LoRA rank，越大拟合力越强
LORA_ALPHA = 64                # alpha = 2 * r 是常用配置
LORA_DROPOUT = 0.05
OUTPUT_DIR = "outputs/lora-adapter"
EPOCHS = 3
BATCH_SIZE = 4                 # A100 80GB 可设 4~8
GRADIENT_ACCUMULATION = 4      # 有效 batch = 4 * 4 = 16
LEARNING_RATE = 2e-4
WARMUP_RATIO = 0.1
DATA_PATH = "data/reasoning_distill.jsonl"
# ================================

# 1) 加载模型（4-bit 量化）
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name=MODEL_NAME,
    max_seq_length=MAX_SEQ_LENGTH,
    dtype=None,                # 自动选择（A100 用 bfloat16）
    load_in_4bit=True,         # QLoRA 4-bit 量化
)

# 2) 添加 LoRA adapter
model = FastLanguageModel.get_peft_model(
    model,
    r=LORA_R,
    lora_alpha=LORA_ALPHA,
    lora_dropout=LORA_DROPOUT,
    target_modules=[
        "q_proj", "k_proj", "v_proj", "o_proj",
        "gate_proj", "up_proj", "down_proj",
    ],
    bias="none",
    use_gradient_checkpointing="unsloth",  # Unsloth 优化版
    random_state=42,
)

# 3) 加载数据集
dataset = load_dataset("json", data_files=DATA_PATH, split="train")

# 格式化为 ChatML 格式（Qwen 系列使用）
# 支持两种输入格式：conversations 格式（蒸馏数据集）和 instruction 格式
def format_chat(example):
    """将 conversations 或 instruction/output 转为 ChatML 格式"""
    # 蒸馏数据集通常是 conversations 格式
    if "conversations" in example:
        messages = []
        for turn in example["conversations"]:
            role = turn.get("from", turn.get("role", "user"))
            if role in ("human", "user"):
                role = "user"
            elif role in ("gpt", "assistant"):
                role = "assistant"
            messages.append({"role": role, "content": turn.get("value", turn.get("content", ""))})
    else:
        # 传统 instruction 格式 fallback
        messages = []
        user_content = example["instruction"]
        if example.get("input"):
            user_content += f"\n{example['input']}"
        messages.append({"role": "user", "content": user_content})
        messages.append({"role": "assistant", "content": example["output"]})
    
    text = tokenizer.apply_chat_template(
        messages,
        tokenize=False,
        add_generation_prompt=False,
    )
    return {"text": text}

dataset = dataset.map(format_chat)

# 4) 配置训练器
training_args = TrainingArguments(
    output_dir=OUTPUT_DIR,
    num_train_epochs=EPOCHS,
    per_device_train_batch_size=BATCH_SIZE,
    gradient_accumulation_steps=GRADIENT_ACCUMULATION,
    learning_rate=LEARNING_RATE,
    warmup_ratio=WARMUP_RATIO,
    lr_scheduler_type="cosine",
    bf16=is_bfloat16_supported(),
    fp16=not is_bfloat16_supported(),
    logging_steps=10,
    save_steps=100,
    save_total_limit=3,
    optim="adamw_8bit",        # 8-bit Adam 节省显存
    seed=42,
    report_to="none",          # 改为 "wandb" 启用追踪
)

trainer = SFTTrainer(
    model=model,
    tokenizer=tokenizer,
    train_dataset=dataset,
    dataset_text_field="text",
    max_seq_length=MAX_SEQ_LENGTH,
    args=training_args,
    packing=True,              # Unsloth packing 提升吞吐
)

# 5) 开始训练
print("=" * 60)
print(f"开始推理蒸馏微调：{MODEL_NAME}")
print(f"数据集大小：{len(dataset)} 条推理蒸馏样本")
print(f"有效 batch size：{BATCH_SIZE * GRADIENT_ACCUMULATION}")
print(f"GPU 内存：{torch.cuda.get_device_properties(0).total_mem / 1e9:.1f} GB")
print("=" * 60)

trainer_stats = trainer.train()

# 6) 保存 LoRA adapter
model.save_pretrained(OUTPUT_DIR)
tokenizer.save_pretrained(OUTPUT_DIR)

print(f"\n训练完成！LoRA adapter 保存至：{OUTPUT_DIR}")
print(f"训练时间：{trainer_stats.metrics['train_runtime']:.0f}s")
print(f"训练损失：{trainer_stats.metrics['train_loss']:.4f}")
```

### 1.3 运行微调

```bash
python scripts/01_finetune.py 2>&1 | tee outputs/finetune.log

# 训练期间可在另一终端查看 GPU 利用率
watch -n 1 nvidia-smi
```

### 1.4 合并 LoRA -> 完整模型 `scripts/02_merge_and_export.py`

```python
"""合并 LoRA adapter 至完整模型权重，供 vLLM 加载"""

from unsloth import FastLanguageModel

MODEL_NAME = "Qwen/Qwen2.5-7B-Instruct"
LORA_DIR = "outputs/lora-adapter"
MERGED_DIR = "outputs/merged-model"
MAX_SEQ_LENGTH = 4096

# 加载 base + LoRA
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name=LORA_DIR,
    max_seq_length=MAX_SEQ_LENGTH,
    load_in_4bit=True,
)

# 合并并以 float16 保存（vLLM 加载需要完整精度权重）
model.save_pretrained_merged(
    MERGED_DIR,
    tokenizer,
    save_method="merged_16bit",  # 以 fp16 保存完整权重
)

print(f"合并模型已保存至：{MERGED_DIR}")
```

```bash
python scripts/02_merge_and_export.py
```

---

## Phase 2：vLLM 部署

### 2.1 启动 Base 模型服务 `scripts/03_serve_vllm.sh`

```bash
#!/bin/bash
# vLLM 服务启动脚本
# 用法：bash scripts/03_serve_vllm.sh [base|finetuned]

MODE=${1:-base}

if [ "$MODE" = "base" ]; then
    MODEL_PATH="Qwen/Qwen2.5-7B-Instruct"
    PORT=8000
elif [ "$MODE" = "finetuned" ]; then
    MODEL_PATH="outputs/merged-model"
    PORT=8001
else
    echo "Usage: $0 [base|finetuned]"
    exit 1
fi

echo "=========================================="
echo "启动 vLLM 服务：$MODE"
echo "模型路径：$MODEL_PATH"
echo "监听端口：$PORT"
echo "=========================================="

python -m vllm.entrypoints.openai.api_server \
    --model "$MODEL_PATH" \
    --port "$PORT" \
    --host 0.0.0.0 \
    --dtype auto \
    --gpu-memory-utilization 0.90 \
    --max-model-len 4096 \
    --max-num-batched-tokens 8192 \
    --enable-chunked-prefill \
    --disable-log-requests \
    --served-model-name "$MODE" \
    2>&1 | tee "outputs/vllm-${MODE}.log"
```

### 2.2 启动步骤

```bash
# 终端 1：启动 base 模型
bash scripts/03_serve_vllm.sh base

# 验证：在另一终端
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "base",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }'
```

### 2.3 关键 vLLM 参数说明

| 参数 | 值 | 说明 |
|------|-----|------|
| `--gpu-memory-utilization` | 0.90 | A100 上可激进分配，留 10% headroom |
| `--max-model-len` | 4096 | 与训练时 max_seq_length 对齐 |
| `--max-num-batched-tokens` | 8192 | A100 推荐 >=8192，提升 prefill 吞吐 |
| `--enable-chunked-prefill` | — | 长 prompt 分块 batch，改善 ITL |
| `--dtype auto` | — | A100 自动选 bfloat16 |

---

## Phase 3：benchctl 推理对比

> **benchctl** 即 vLLM 内置的 benchmark 工具链，包括：
> - `benchmark_serving.py`：模拟在线并发请求，测量 TTFT / ITL / throughput
> - `benchmark_throughput.py`：离线批处理最大吞吐
> - `vllm bench`：CLI 封装（v0.6+ 可用）

### 3.1 准备 benchmark 输入数据

```bash
# 使用 ShareGPT 数据集（vLLM 官方推荐）
wget -O data/ShareGPT_V3_unfiltered_cleaned_split.json \
  https://huggingface.co/datasets/anon8231489123/ShareGPT_Vicuna_unfiltered/resolve/main/ShareGPT_V3_unfiltered_cleaned_split.json
```

### 3.2 Benchmark Base 模型 `scripts/04_bench_base.sh`

```bash
#!/bin/bash
# benchmark_serving.py 对 base 模型的在线推理压测

VLLM_REPO=$(python -c "import vllm; import os; print(os.path.dirname(vllm.__file__))")
BENCH_SCRIPT="${VLLM_REPO}/../benchmarks/benchmark_serving.py"

# 如果 benchmark_serving.py 不在安装目录，则从 GitHub 下载
if [ ! -f "$BENCH_SCRIPT" ]; then
    echo "下载 benchmark_serving.py..."
    wget -O /tmp/benchmark_serving.py \
      https://raw.githubusercontent.com/vllm-project/vllm/main/benchmarks/benchmark_serving.py
    BENCH_SCRIPT="/tmp/benchmark_serving.py"
fi

echo "=========================================="
echo "Benchmarking: BASE model (port 8000)"
echo "=========================================="

# --- 测试 1：低并发（QPS=2）---
python "$BENCH_SCRIPT" \
    --backend openai-chat \
    --base-url http://localhost:8000 \
    --model base \
    --dataset-name sharegpt \
    --dataset-path data/ShareGPT_V3_unfiltered_cleaned_split.json \
    --num-prompts 100 \
    --request-rate 2 \
    --save-result \
    --result-dir outputs/benchmarks \
    --result-filename base-qps2.json

# --- 测试 2：中等并发（QPS=8）---
python "$BENCH_SCRIPT" \
    --backend openai-chat \
    --base-url http://localhost:8000 \
    --model base \
    --dataset-name sharegpt \
    --dataset-path data/ShareGPT_V3_unfiltered_cleaned_split.json \
    --num-prompts 200 \
    --request-rate 8 \
    --save-result \
    --result-dir outputs/benchmarks \
    --result-filename base-qps8.json

# --- 测试 3：压力测试（QPS=inf, 全速）---
python "$BENCH_SCRIPT" \
    --backend openai-chat \
    --base-url http://localhost:8000 \
    --model base \
    --dataset-name sharegpt \
    --dataset-path data/ShareGPT_V3_unfiltered_cleaned_split.json \
    --num-prompts 300 \
    --request-rate inf \
    --save-result \
    --result-dir outputs/benchmarks \
    --result-filename base-qps-inf.json

echo "Base 模型 benchmark 完成！结果保存在 outputs/benchmarks/"
```

### 3.3 Benchmark Distilled 模型 `scripts/05_bench_finetuned.sh`

```bash
#!/bin/bash
# 与 base 完全相同的测试配置，仅切换端口

# 先停掉 base 服务，启动 fine-tuned 服务
# 或者并行运行在不同端口

VLLM_REPO=$(python -c "import vllm; import os; print(os.path.dirname(vllm.__file__))")
BENCH_SCRIPT="${VLLM_REPO}/../benchmarks/benchmark_serving.py"
[ ! -f "$BENCH_SCRIPT" ] && BENCH_SCRIPT="/tmp/benchmark_serving.py"

echo "=========================================="
echo "Benchmarking: DISTILLED model (port 8001)"
echo "=========================================="

for QPS in 2 8 inf; do
    PROMPTS=$( [ "$QPS" = "2" ] && echo 100 || ([ "$QPS" = "8" ] && echo 200 || echo 300) )
    
    python "$BENCH_SCRIPT" \
        --backend openai-chat \
        --base-url http://localhost:8001 \
        --model finetuned \
        --dataset-name sharegpt \
        --dataset-path data/ShareGPT_V3_unfiltered_cleaned_split.json \
        --num-prompts "$PROMPTS" \
        --request-rate "$QPS" \
        --save-result \
        --result-dir outputs/benchmarks \
        --result-filename "finetuned-qps${QPS}.json"
    
    echo "QPS=${QPS} 完成"
done

echo "Distilled 模型 benchmark 完成！"
```

### 3.4 离线吞吐测试（可选）

```bash
# 使用 benchmark_throughput.py 测量离线最大吞吐
BENCH_TP="${VLLM_REPO}/../benchmarks/benchmark_throughput.py"
[ ! -f "$BENCH_TP" ] && wget -O /tmp/benchmark_throughput.py \
  https://raw.githubusercontent.com/vllm-project/vllm/main/benchmarks/benchmark_throughput.py && \
  BENCH_TP="/tmp/benchmark_throughput.py"

# Base 模型
python "$BENCH_TP" \
    --model Qwen/Qwen2.5-7B-Instruct \
    --input-len 512 --output-len 128 \
    --num-prompts 500 \
    --dtype auto \
    --output-json outputs/benchmarks/base-throughput.json

# Distilled 模型
python "$BENCH_TP" \
    --model outputs/merged-model \
    --input-len 512 --output-len 128 \
    --num-prompts 500 \
    --dtype auto \
    --output-json outputs/benchmarks/distilled-throughput.json
```

---

## Phase 4：结果对比与分析

### 4.1 对比脚本 `scripts/06_compare.py`

```python
"""
解析 benchmark JSON 结果，生成对比表格和可视化图表
增加推理效率指标：平均输出 token 数（衡量蒸馏效率）
"""

import json
import os
from pathlib import Path
from tabulate import tabulate
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

BENCH_DIR = Path("outputs/benchmarks")
REPORT_DIR = Path("report")
REPORT_DIR.mkdir(exist_ok=True)

def load_result(filename):
    """加载 benchmark_serving.py 的 JSON 输出"""
    filepath = BENCH_DIR / filename
    if not filepath.exists():
        print(f"  [跳过] {filename} 不存在")
        return None
    with open(filepath) as f:
        return json.load(f)

def extract_metrics(data):
    """从 benchmark 结果中提取关键指标（含推理效率指标）"""
    if data is None:
        return {}
    completed = data.get("completed", 0) or 1
    return {
        "throughput_req_s": completed / data.get("duration", 1),
        "throughput_tok_s": data.get("total_output_tokens", 0) / data.get("duration", 1),
        "mean_ttft_ms": data.get("mean_ttft_ms", "N/A"),
        "median_ttft_ms": data.get("median_ttft_ms", "N/A"),
        "p99_ttft_ms": data.get("p99_ttft_ms", "N/A"),
        "mean_itl_ms": data.get("mean_itl_ms", "N/A"),
        "median_itl_ms": data.get("median_itl_ms", "N/A"),
        "p99_itl_ms": data.get("p99_itl_ms", "N/A"),
        "mean_e2e_ms": data.get("mean_e2e_latency_ms", "N/A"),
        # 推理效率指标：蒸馏后模型是否"想得更短"
        "avg_output_tokens": data.get("total_output_tokens", 0) / completed,
        "avg_input_tokens": data.get("total_input_tokens", 0) / completed,
    }

# ---- 加载所有结果 ----
scenarios = ["qps2", "qps8", "qps-inf"]
results = {}

for scenario in scenarios:
    base_data = load_result(f"base-{scenario}.json")
    ft_data = load_result(f"finetuned-{scenario}.json")
    results[scenario] = {
        "base": extract_metrics(base_data),
        "distilled": extract_metrics(ft_data),
    }

# ---- 生成对比表格 ----
table_rows = []
for scenario in scenarios:
    b = results[scenario]["base"]
    f = results[scenario]["distilled"]
    if not b or not f:
        continue
    
    def fmt(v):
        return f"{v:.1f}" if isinstance(v, (int, float)) else "N/A"
    
    # 计算推理效率变化
    b_tok = b.get('avg_output_tokens', 0)
    f_tok = f.get('avg_output_tokens', 0)
    tok_delta = f"({(f_tok - b_tok) / b_tok * 100:+.0f}%)" if b_tok > 0 and isinstance(f_tok, (int, float)) else ""
    
    table_rows.append([
        scenario,
        fmt(b.get('throughput_tok_s')),
        fmt(f.get('throughput_tok_s')),
        fmt(b.get('mean_ttft_ms')),
        fmt(f.get('mean_ttft_ms')),
        fmt(b.get('p99_itl_ms')),
        fmt(f.get('p99_itl_ms')),
        fmt(b_tok),
        f"{fmt(f_tok)} {tok_delta}",
    ])

headers = [
    "QPS 场景",
    "Base tok/s", "Distilled tok/s",
    "Base TTFT(ms)", "Distilled TTFT(ms)",
    "Base P99 ITL(ms)", "Distilled P99 ITL(ms)",
    "Base 平均输出tok", "Distilled 平均输出tok",
]

table_str = tabulate(table_rows, headers=headers, tablefmt="github")
print("\n" + "=" * 80)
print("Base vs Distilled 推理性能对比")
print("=" * 80)
print(table_str)

# ---- 保存 Markdown 报告 ----
report = f"""# A100 推理蒸馏对比报告

## 实验配置
- **GPU**: NVIDIA A100 80GB SXM
- **Base 模型**: Qwen/Qwen2.5-7B-Instruct
- **Distilled**: QLoRA (r=32, alpha=64), 3 epochs, 推理蒸馏数据 ~3000 条
- **训练数据**: Claude 4.6 Opus 推理链蒸馏（Opus-4.6-Reasoning-3000x-filtered + claude-4.5-opus-high-reasoning-250x）
- **推理引擎**: vLLM (continuous batching, chunked prefill)
- **Benchmark 数据集**: ShareGPT

## 性能对比

{table_str}

## 关键发现

> 请根据实际数据填写以下分析

1. **推理效率**：蒸馏后模型平均输出 token 数变化 [X%]——验证"想得更短"的假设
2. **吞吐量对比**：base vs distilled tokens/s 差异 [分析]
3. **延迟对比**：TTFT / ITL 差异 [分析]
4. **高并发表现**：QPS=inf 场景下两者的表现差异 [分析]
5. **结论**：推理蒸馏是否在保持准确性的同时提升了推理效率？

## 跨硬件实验对比矩阵

| 维度 | AI Infra Platform (RTX 4050) | 本次实验 (A100 80GB) |
|------|-------------------------------|---------------------|
| GPU | RTX 4050 Laptop (6GB) | A100 SXM (80GB) |
| 模型 | Qwen2.5-1.5B | Qwen2.5-7B |
| 对比维度 | Ollama vs vLLM（推理引擎对比） | base vs distilled（蒸馏效果对比） |
| 量化 | Q4_K_M / fp16 | QLoRA 4bit -> fp16 合并 |
| 核心发现 | continuous batching +352% QPS | [待填] |
| 发布 | — | HuggingFace + GGUF 量化版 |
"""

report_path = REPORT_DIR / "analysis.md"
with open(report_path, "w", encoding="utf-8") as f:
    f.write(report)

print(f"\n报告已保存至：{report_path}")
```

### 4.2 运行对比

```bash
python scripts/06_compare.py
```

---

## Phase 5：GGUF 量化 + HuggingFace 发布

### 5.1 安装 llama.cpp 量化工具

```bash
# 方案一：使用 llama-cpp-python（简单）
pip install llama-cpp-python

# 方案二：编译 llama.cpp（更灵活，推荐）
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp && make -j$(nproc) && cd ..
```

### 5.2 转换为 GGUF 格式

```bash
# 从 HF 格式转换为 GGUF
python llama.cpp/convert_hf_to_gguf.py \
    outputs/merged-model \
    --outfile outputs/gguf/model-fp16.gguf \
    --outtype f16

# 多精度量化
./llama.cpp/llama-quantize outputs/gguf/model-fp16.gguf outputs/gguf/model-Q4_K_M.gguf Q4_K_M
./llama.cpp/llama-quantize outputs/gguf/model-fp16.gguf outputs/gguf/model-Q8_0.gguf Q8_0

ls -lh outputs/gguf/
# 应看到 fp16 (~14GB)、Q4_K_M (~4GB)、Q8_0 (~7GB)
```

### 5.3 发布到 HuggingFace `scripts/07_publish.sh`

```bash
#!/bin/bash
set -e

HF_USERNAME="Xio-Shark"  # 替换为你的 HuggingFace 用户名
MODEL_NAME="Qwen2.5-7B-Reasoning-Distilled"

echo "=========================================="
echo "发布模型到 HuggingFace"
echo "=========================================="

# 创建 HF 模型仓库
huggingface-cli repo create "${MODEL_NAME}" --type model || true
huggingface-cli repo create "${MODEL_NAME}-GGUF" --type model || true

# 上传完整模型（fp16）
huggingface-cli upload "${HF_USERNAME}/${MODEL_NAME}" \
    outputs/merged-model/ .

# 上传 GGUF 量化版
huggingface-cli upload "${HF_USERNAME}/${MODEL_NAME}-GGUF" \
    outputs/gguf/ .

# 上传 benchmark 报告作为模型卡的补充
huggingface-cli upload "${HF_USERNAME}/${MODEL_NAME}" \
    report/analysis.md report/analysis.md

echo "发布完成！"
echo "模型页面：https://huggingface.co/${HF_USERNAME}/${MODEL_NAME}"
echo "GGUF 页面：https://huggingface.co/${HF_USERNAME}/${MODEL_NAME}-GGUF"
```

```bash
bash scripts/07_publish.sh
```

---

## 完整执行流程（一键式）

```bash
#!/bin/bash
set -e

echo "====== Phase 0: 环境检查 ======"
nvidia-smi
python -c "import unsloth; import vllm; print('环境就绪')"

echo "====== Phase 1: 推理蒸馏微调 ======"
python scripts/00_prepare_data.py
python scripts/01_finetune.py 2>&1 | tee outputs/finetune.log
python scripts/02_merge_and_export.py 2>&1 | tee outputs/merge.log

echo "====== Phase 2: vLLM 部署 Base 模型 ======"
bash scripts/03_serve_vllm.sh base &
BASE_PID=$!
sleep 60  # 等待模型加载完毕

echo "====== Phase 3a: Benchmark Base ======"
bash scripts/04_bench_base.sh

echo "====== 停止 Base, 启动 Distilled ======"
kill $BASE_PID
sleep 10
bash scripts/03_serve_vllm.sh finetuned &
FT_PID=$!
sleep 60

echo "====== Phase 3b: Benchmark Distilled ======"
bash scripts/05_bench_finetuned.sh

echo "====== Phase 4: 对比分析 ======"
kill $FT_PID
python scripts/06_compare.py

echo "====== Phase 5: 量化 + 发布 ======"
bash scripts/07_publish.sh

echo "====== 实验完成！======"
echo "查看报告：cat report/analysis.md"
```

---

## 关键指标解读

| 指标 | 含义 | 优化方向 |
|------|------|----------|
| **TTFT** (Time to First Token) | 首 token 延迟，反映 prefill 速度 | 更长 `max-num-batched-tokens` |
| **ITL** (Inter-Token Latency) | 相邻 token 间隔，影响用户体验 | `enable-chunked-prefill` |
| **Throughput (tok/s)** | 系统整体吞吐 | 更高 `gpu-memory-utilization` |
| **QPS** | 请求级别吞吐 | continuous batching |
| **P50/P95/P99** | 尾延迟分布 | 减少 KV cache eviction |
| **平均输出 token 数** | 蒸馏效率核心指标 | 蒸馏数据质量 + scaffold 设计 |

---

## 注意事项 & 故障排查

### 常见问题

| 问题 | 解决方案 |
|------|----------|
| Unsloth 安装失败 | 确认 CUDA 12.x + Python 3.11，`pip install unsloth --no-deps` 重试 |
| vLLM OOM | 降低 `--gpu-memory-utilization 0.85` 或 `--max-model-len 2048` |
| benchmark 结果异常 | 确认 vLLM 已完全启动（日志显示 "Started server process"） |
| LoRA 合并后模型加载失败 | 检查 `save_method="merged_16bit"` 是否正确执行 |
| TTFT 过高 | 检查是否有其他进程占用 GPU，`nvidia-smi` 确认独占 |
| GGUF 转换失败 | 确认 llama.cpp 已编译且 `convert_hf_to_gguf.py` 路径正确 |
| HF 上传超时 | 使用 `huggingface-cli upload --resume` 断点续传 |

### GPU 监控（全程运行）

```bash
# 终端 1：实时 GPU 监控
nvidia-smi dmon -s pucvmet -d 5 | tee outputs/gpu_monitor.csv

# 或使用 gpustat
pip install gpustat
watch -n 1 gpustat --color
```

---

## 实验成果对简历的价值

完成本实验后，可为 **AI Infra 推理性能工程** 简历增加以下亮点：

1. **推理蒸馏微调**：A100 上使用 Unsloth QLoRA 对 Qwen2.5-7B 进行 Claude Opus 推理蒸馏，训练数据 3000+ 条
2. **vLLM 生产部署**：配置 continuous batching + chunked prefill，调优 `gpu_memory_utilization` 等关键参数
3. **推理效率量化**：base vs distilled 三场景压测，量化 TTFT / ITL / throughput / **平均输出 token 数**差异
4. **跨硬件对比矩阵**：RTX 4050（引擎对比）+ A100（蒸馏对比）形成 **完整 AI Infra 实验矩阵**
5. **HuggingFace 发布**：模型 + GGUF 量化版公开发布，可量化社区影响力
