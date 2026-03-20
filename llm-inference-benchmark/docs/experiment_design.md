# 实验设计（模板）

1. **Baseline**：默认引擎参数，记录 QPS、P95、TTFT、tokens/s。
2. **调参轮次**：每次只改 1～2 个变量（如 `max_num_seqs` / batch tokens），其余固定。
3. **对比**：同 workload（并发、prompt 长度、输出长度），保存 `run_benchmark --output-json` 结果与 Grafana 截图。
