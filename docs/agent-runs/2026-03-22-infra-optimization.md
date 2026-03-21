# 任务日志 — 2026-03-22 Infra 项目优化

## 基本信息

- **任务名称**：AI Infra Platform 优化 + RTX 4050 GPU 压测实验
- **项目类型**：C（成熟项目优化/重构）+ 次类型 E（性能专项）
- **风险等级**：R1（低风险局部改动——补文档 + 运行压测，不改核心代码）

## 目标

1. 在 Windows RTX 4050 Laptop GPU 上运行 Qwen2.5-1.5B 压测实验，与已有 CPU 数据对比
2. 将 infra 项目纳入投递体系（工作区 README、简历证据导航）
3. 补齐其他主项目已有的标准化文档
4. 清理 Redis store 代码异味

## 约束

- 不改变核心代码逻辑
- 压测仅在本地 Ollama 上进行

## 方案摘要

四阶段执行：GPU 压测 → 纳入投递体系 → 标准化文档 → 代码清理

## 关键决策

- Redis store 保留为 roadmap 占位，加注释标注，不删除
- GPU warm baseline (QPS=2.48) 作为对比基准值，排除冷启动影响

## 变更文件

### 新增

| 文件 | 说明 |
|------|------|
| `docs/简历条目对照表.md` | 简历 bullet → 代码/测试/验证映射 |
| `docs/API请求与响应样例.md` | 12 条 API 固定输入输出样例 |
| `docs/异常场景样例.md` | 8 个异常/边界场景 |
| `docs/工程证据索引.md` | 量化证据汇总 |
| `reports/generated/bench_20260322_025612.*` | GPU c=3 冷启动 |
| `reports/generated/bench_20260322_025641.*` | GPU c=10 |
| `reports/generated/bench_20260322_025723.*` | GPU c=20 |
| `reports/generated/bench_20260322_025745.*` | GPU c=3 热稳态 |

### 修改

| 文件 | 说明 |
|------|------|
| `reports/benchmark_comparison.md` | 重写为 CPU vs GPU 完整对比报告 |
| `README.md` | 加 GPU 亮点、Redis roadmap、文档导航 |
| `internal/store/redis/store.go` | 加 roadmap 占位注释 |
| 工作区 `README.md` | 项目地图新增 AI Infra Platform |

## GPU 实验结果

| 场景 | QPS | P50 | tokens/s | vs CPU |
|------|-----|-----|----------|--------|
| c=3 GPU (warm) | 2.48 | 1,162 ms | 48.65 | QPS +119%, P50 -56% |
| c=10 GPU | 2.36 | 4,058 ms | 17.84 | QPS +109% |
| c=20 GPU | 2.44 | 7,848 ms | 9.92 | — |

## 风险与未覆盖项

- GPU 实验仅在 Ollama 上执行，未在 vLLM / TGI 上验证
- Redis store 仍未实现，如面试被追问需用 roadmap 说法应对
- 简历中 infra 项目的 bullet 尚未整合到定向简历文件

## 回滚方式

- 所有改动为文档和注释，`git revert` 即可
- 压测报告为追加文件，不影响原有报告

## 后续建议

1. 将 `resume_bullet.md` 中的 4 条 bullet 整合到后端/AI 方向简历
2. 更新 `简历与仓库证据导航.md` 和 `投递优先级清单`
3. 如有 vLLM / continuous batching 环境，补充真并发扩展实验
