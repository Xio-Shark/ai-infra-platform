package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ai-infra-platform/internal/model"
)

type K8sApplyExecutor struct {
	Namespace  string
	Kubeconfig string
	AllowApply bool
}

func (e K8sApplyExecutor) Name() string { return "k8s-apply" }

func (e K8sApplyExecutor) Execute(ctx context.Context, job model.Job) (Result, error) {
	if !e.AllowApply {
		return Result{}, fmt.Errorf("k8s-apply is disabled; set ALLOW_K8S_APPLY=true to opt in")
	}
	manifest, err := buildJobManifest(job, e.Namespace)
	if err != nil {
		return Result{}, err
	}
	command := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	command.Stdin = strings.NewReader(manifest)
	command.Env = kubectlEnv(e.Kubeconfig)
	output, runErr := command.CombinedOutput()
	result := Result{Logs: string(output), Manifest: manifest, ExitCode: exitCode(runErr)}
	if runErr != nil {
		return result, fmt.Errorf("kubectl apply failed: %w", runErr)
	}
	return result, nil
}

func kubectlEnv(kubeconfig string) []string {
	env := os.Environ()
	if kubeconfig == "" {
		return env
	}
	return append(env, "KUBECONFIG="+kubeconfig)
}
