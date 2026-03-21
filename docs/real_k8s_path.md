# Real Kubernetes Path

## 目标

在保留 `k8s-dry-run` 为默认安全路径的前提下，提供一个显式 opt-in 的真实提交骨架。

## 当前路径

- `k8s-dry-run`
  只生成 manifest，不调用集群
- `k8s-apply`
  调用 `kubectl apply -f -` 的可选执行器

## 安全边界

- 默认不会启用真实提交
- 只有设置 `ALLOW_K8S_APPLY=true` 才会执行 `k8s-apply`
- kubeconfig 通过标准 `KUBECONFIG` 注入，不写入源码
- 建议在接真实集群前增加人工 review 或 `kubectl diff`
