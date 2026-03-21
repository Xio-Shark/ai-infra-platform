# 排障

## `go test` 提示无法写入 Go cache

本仓库默认将 cache 写入项目内：

```bash
mkdir -p .tmp/go-cache .tmp/go-mod-cache
GOCACHE=$(pwd)/.tmp/go-cache GOMODCACHE=$(pwd)/.tmp/go-mod-cache go test ./...
```

## `k8s-dry-run` 没有真正执行任务

这是预期行为。该执行器只生成 manifest，用于显式 dry-run 调试。如果需要真实提交，需要额外接入 Kubernetes client。

## `k8s-apply` 没有执行

- 只有在 `ALLOW_K8S_APPLY=true` 时才会启用
- 检查 `kubectl` 是否存在于 PATH
- 如需指定 kubeconfig，设置 `KUBECONFIG=/path/to/config`

## Shell 执行器失败

- 检查 `command` 是否非空
- 检查命令在当前系统是否存在
- 查看 `GET /jobs/{id}/executions` 的 `logs` 和 `error`
