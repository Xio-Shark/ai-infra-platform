下面直接给你可落地版本。
我按你当前背景（**Go / K8s / Prometheus / OpenTelemetry / 调度平台**）来设计，尽量做到：

* **能做出来**
* **能写进简历**
* **能在面试里讲清楚**
* **和 JD 关键词高度贴合**

---

# 1）LLM 推理服务压测与优化平台

## 仓库名建议

`llm-inference-benchmark`

---

## README 大纲

### 1. 项目简介

* 这个项目解决什么问题
* 为什么需要对 LLM 推理服务做 benchmark 和 tuning
* 项目核心能力：

  * 部署推理引擎
  * 压测
  * 指标采集
  * 参数调优
  * 优化结果分析

### 2. 系统架构

* 推理引擎：vLLM / SGLang
* 服务接口：OpenAI-compatible API / FastAPI
* 压测客户端：自研 benchmark client
* 指标采集：Prometheus
* 可视化：Grafana
* tracing：OpenTelemetry（可选增强）

### 3. 环境说明

* OS / Python 版本
* CUDA 版本
* GPU 型号
* 模型版本
* vLLM/SGLang 版本

### 4. 快速开始

* 安装依赖
* 下载模型
* 启动推理服务
* 运行 benchmark
* 启动监控面板

### 5. 核心指标定义

* QPS
* P50 / P95 / P99 latency
* TTFT
* TPOT
* tokens/s
* GPU utilization
* GPU memory usage
* error rate

### 6. Benchmark 场景设计

* 不同并发
* 不同 prompt 长度
* 不同输出长度
* 不同 batch / engine 参数
* 不同量化方式（如果做）

### 7. 调优项

* `max_num_seqs`
* `max_num_batched_tokens`
* batch 策略
* context length
* quantization
* tensor parallel（有条件再做）

### 8. 实验结果

* baseline
* 调优前后对比
* 指标图表
* 瓶颈分析

### 9. 可观测性设计

* Prometheus metrics
* Grafana dashboard
* tracing 拆分 queue/prefill/decode（可选）

### 10. 项目复现

* 一键脚本
* 配置说明
* 常见问题

### 11. 后续优化方向

* 多模型对比
* 多机部署
* 请求分级与限流
* prefix cache / KV cache tuning

---

## 最小可交付版本清单（MVP）

你至少要做到这些：

### 基础能力

* [ ] 跑通一个推理引擎（优先 vLLM）
* [ ] 跑通一个开源模型（建议 Qwen2.5-1.5B/3B/7B，按显卡选）
* [ ] 提供统一推理接口
* [ ] 写一个 benchmark client

### Benchmark 能力

* [ ] 支持设置并发数
* [ ] 支持设置 prompt 长度
* [ ] 支持设置输出 token 长度
* [ ] 输出：

  * [ ] latency
  * [ ] TTFT
  * [ ] tokens/s
  * [ ] success rate

### 监控能力

* [ ] 接 Prometheus
* [ ] 做一个 Grafana dashboard
* [ ] 监控 GPU util / 显存 / 请求吞吐 / 延迟分布

### 优化能力

* [ ] 先做 baseline
* [ ] 至少调 2 组参数
* [ ] 给出优化前后结果对比
* [ ] 写一份简短实验报告

### 最终结果

* [ ] README 可复现
* [ ] 有 dashboard 截图
* [ ] 有结果图表
* [ ] 有结论：为什么变快 / 为什么没变快

---

## GitHub 仓库目录结构

```text
llm-inference-benchmark/
├── README.md
├── requirements.txt
├── .gitignore
├── docker-compose.yml
├── configs/
│   ├── model.yaml
│   ├── benchmark.yaml
│   └── prometheus.yml
├── server/
│   ├── app.py
│   ├── routers/
│   │   └── inference.py
│   ├── services/
│   │   └── engine_client.py
│   └── schemas/
│       └── request_response.py
├── benchmark/
│   ├── run_benchmark.py
│   ├── workload_generator.py
│   ├── metrics.py
│   ├── reporter.py
│   └── datasets/
│       ├── short_prompts.jsonl
│       ├── medium_prompts.jsonl
│       └── long_prompts.jsonl
├── monitoring/
│   ├── exporter.py
│   ├── grafana/
│   │   └── dashboard.json
│   └── prometheus/
│       └── rules.yml
├── traces/
│   └── otel_setup.py
├── scripts/
│   ├── start_server.sh
│   ├── run_benchmark.sh
│   ├── collect_gpu_stats.sh
│   └── plot_results.py
├── reports/
│   ├── baseline.md
│   ├── tuning_round_1.md
│   └── final_report.md
└── docs/
    ├── architecture.md
    ├── metrics_definition.md
    └── experiment_design.md
```

---

# 2）PyTorch Profiling 与性能优化实验

## 仓库名建议

`pytorch-profiling-lab`

---

## README 大纲

### 1. 项目简介

* 目标：用 profiler 做训练/推理性能剖析
* 不追求 SOTA，只关注性能定位和优化过程
* 展示你会：

  * profile
  * 设计实验
  * 做优化
  * 写分析报告

### 2. 实验对象

* 模型：ResNet / ViT / BERT / 小型 Transformer
* 任务：训练或推理
* 数据集：CIFAR-10 / IMDB / synthetic data 都行

### 3. 环境说明

* PyTorch / CUDA / Python 版本
* GPU / CPU / 内存环境
* 是否开启 AMP / bf16

### 4. Baseline 实现

* 原始训练脚本
* baseline 配置
* 初始吞吐 / step time / 显存

### 5. Profiling 方法

* 使用 `torch.profiler`
* 采集：

  * CPU time
  * CUDA time
  * kernel
  * memory
  * dataloader wait
* trace 输出方式

### 6. 优化项设计

* `num_workers`
* `pin_memory`
* AMP / bf16
* batch size
* `torch.compile`
* gradient checkpointing（选做）
* fused optimizer（选做）

### 7. 实验结果

* baseline vs optimized
* 每个优化项的收益
* 哪些优化几乎无效
* 为什么

### 8. 结论

* 主要瓶颈是什么
* 哪些优化适用于什么场景
* 经验总结

### 9. 复现实验

* 如何运行 baseline
* 如何运行 profile
* 如何运行 compare
* 如何查看 trace

### 10. 后续扩展

* 多卡
* 分布式训练
* inference profiling
* attention kernel 对比

---

## 最小可交付版本清单（MVP）

### 基础能力

* [ ] 选一个模型和一个任务
* [ ] 跑通 baseline 训练/推理脚本
* [ ] 输出基本指标：

  * [ ] step time
  * [ ] throughput
  * [ ] GPU memory

### Profiling 能力

* [ ] 集成 `torch.profiler`
* [ ] 导出 trace
* [ ] 能看出 CPU / CUDA / data loading 主要耗时

### 实验能力

* [ ] 至少做 4 组实验对比：

  * [ ] num_workers
  * [ ] pin_memory
  * [ ] AMP
  * [ ] batch size / torch.compile 二选一
* [ ] 每组实验有记录和结论

### 输出能力

* [ ] 有图表
* [ ] 有 trace 截图
* [ ] 有实验报告
* [ ] 能清楚说出瓶颈来源

---

## GitHub 仓库目录结构

```text
pytorch-profiling-lab/
├── README.md
├── requirements.txt
├── .gitignore
├── configs/
│   ├── baseline.yaml
│   ├── amp.yaml
│   ├── dataloader_tuned.yaml
│   └── compile.yaml
├── data/
│   └── README.md
├── models/
│   ├── resnet.py
│   ├── vit.py
│   └── transformer_small.py
├── train/
│   ├── train.py
│   ├── eval.py
│   ├── engine.py
│   └── utils.py
├── profile/
│   ├── run_profile.py
│   ├── trace_handler.py
│   ├── analyze_trace.py
│   └── export_summary.py
├── experiments/
│   ├── baseline.py
│   ├── exp_num_workers.py
│   ├── exp_pin_memory.py
│   ├── exp_amp.py
│   └── exp_compile.py
├── scripts/
│   ├── run_baseline.sh
│   ├── run_profile.sh
│   ├── run_all_experiments.sh
│   └── plot_results.py
├── outputs/
│   ├── logs/
│   ├── traces/
│   ├── tables/
│   └── figures/
├── reports/
│   ├── baseline.md
│   ├── profiling_notes.md
│   └── final_report.md
└── docs/
    ├── experiment_plan.md
    ├── metric_definitions.md
    └── optimization_summary.md
```

---

# 3）AI Job Orchestrator：训练/推理任务编排平台

这个项目最适合直接从你现有 **GO Cloud** 改。

## 仓库名建议

`ai-job-orchestrator`

---

## README 大纲

### 1. 项目简介

* 面向 AI 训练/推理任务的作业编排平台
* 支持任务提交、调度、执行、回查、观测
* 和通用任务调度的区别：

  * 需要记录模型/镜像/数据集版本
  * 需要资源规格
  * 需要更强的运行指标

### 2. 系统架构

* api-server
* scheduler
* worker
* notifier
* Kubernetes Job executor
* MySQL / Redis
* Prometheus / OpenTelemetry

### 3. 核心数据模型

* job
* task attempt
* execution record
* model version
* dataset version
* resource spec
* job type（training / inference / eval）

### 4. 任务生命周期

* pending
* scheduled
* running
* succeeded
* failed
* retrying
* cancelled

### 5. 调度与执行机制

* 抢占控制
* 幂等
* 重试机制
* 优先级
* 并发限制
* executor 抽象

### 6. AI 场景扩展

* 训练任务模板
* 推理压测任务模板
* 评测任务模板
* GPU 资源字段
* 模型与镜像版本记录

### 7. 可观测性设计

* task latency
* retry count
* success rate
* queue wait time
* execution runtime
* tracing from submit to execution

### 8. API 说明

* 创建任务
* 查询任务
* 查询执行记录
* 重试 / 取消
* 查询日志 / 指标

### 9. 部署方式

* 本地
* Docker Compose
* Kubernetes
* 示例任务

### 10. 系统设计取舍

* 为什么拆分成多个服务
* 为什么用 Redis 分布式锁
* 为什么用 Job 而不是直接内置执行

### 11. 后续扩展

* quota
* GPU 队列
* Ray Job / Volcano
* 多租户
* cost accounting

---

## 最小可交付版本清单（MVP）

### 基础能力

* [ ] 在现有 GO Cloud 基础上新增 AI job type：

  * [ ] training
  * [ ] inference
* [ ] 新增任务字段：

  * [ ] model_version
  * [ ] dataset_version
  * [ ] image_tag
  * [ ] resource_spec
  * [ ] job_type

### 调度与执行

* [ ] 任务提交接口
* [ ] 任务状态流转
* [ ] K8s Job 执行
* [ ] 重试机制
* [ ] 日志查询
* [ ] 执行记录查询

### 观测能力

* [ ] Prometheus metrics
* [ ] 任务级 runtime / success rate / retry count
* [ ] OpenTelemetry trace 打通 submit -> schedule -> run

### 演示能力

* [ ] 提供一个 training job demo
* [ ] 提供一个 inference benchmark job demo
* [ ] README 能完整复现

---

## GitHub 仓库目录结构

```text
ai-job-orchestrator/
├── README.md
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── deploy/
│   ├── docker-compose.yml
│   ├── k8s/
│   │   ├── api-server.yaml
│   │   ├── scheduler.yaml
│   │   ├── worker.yaml
│   │   ├── mysql.yaml
│   │   ├── redis.yaml
│   │   └── prometheus.yaml
│   └── grafana/
│       └── dashboard.json
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   ├── scheduler/
│   │   └── main.go
│   ├── worker/
│   │   └── main.go
│   └── notifier/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── router.go
│   ├── scheduler/
│   │   ├── dispatcher.go
│   │   ├── retry.go
│   │   └── priority_queue.go
│   ├── worker/
│   │   ├── executor.go
│   │   ├── shell_executor.go
│   │   ├── http_executor.go
│   │   └── k8s_job_executor.go
│   ├── model/
│   │   ├── job.go
│   │   ├── execution.go
│   │   └── resource_spec.go
│   ├── store/
│   │   ├── mysql/
│   │   └── redis/
│   ├── telemetry/
│   │   ├── metrics.go
│   │   └── tracing.go
│   └── service/
│       ├── job_service.go
│       └── execution_service.go
├── api/
│   └── openapi.yaml
├── migrations/
│   ├── 001_init.sql
│   └── 002_ai_job_fields.sql
├── examples/
│   ├── training_job.json
│   ├── inference_job.json
│   └── benchmark_job.json
├── scripts/
│   ├── dev_up.sh
│   ├── seed_jobs.sh
│   ├── smoke_test.sh
│   └── load_test.sh
├── reports/
│   ├── architecture.md
│   ├── state_machine.md
│   └── observability.md
└── docs/
    ├── system_design.md
    ├── api_reference.md
    └── troubleshooting.md
```

---

# 我对你最推荐的执行顺序

## 第一阶段

先做这两个：

1. **llm-inference-benchmark**
2. **pytorch-profiling-lab**

这是补短板。

## 第二阶段

再把你现有 GO Cloud 改成：

3. **ai-job-orchestrator**

这是把你已有优势升级成 AI Infra 叙事。

---

# 你现在最该注意的一个点

这三个项目里，**最重要的不是“功能很多”**，而是这三件事：

* **有明确指标**
* **有优化前后对比**
* **有实验结论**

没有这三样，项目会像“搭了个 demo”。
有这三样，项目才像“做过性能分析和工程优化”。

---

# 你可以直接开工的最小方案

## 项目 1 最小版

* vLLM + Qwen2.5-1.5B
* benchmark client
* Prometheus + Grafana
* 2 组参数对比
* 1 份结果报告

## 项目 2 最小版

* ResNet / 小 Transformer
* `torch.profiler`
* 4 组实验
* trace + 图表 + 结论

## 项目 3 最小版

* GO Cloud 增加 training/inference job type
* K8s Job executor
* metrics + tracing
* 示例任务

---

下一步我可以继续给你一版 **这三个项目的“简历 bullet 成品版”**，你做完后几乎可以直接粘进简历。
