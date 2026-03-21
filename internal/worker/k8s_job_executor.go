package worker

import (
	"context"

	"ai-infra-platform/internal/model"
)

type K8sJobExecutor struct {
	Namespace string
}

func (e K8sJobExecutor) Name() string { return "k8s-dry-run" }

func (e K8sJobExecutor) Execute(_ context.Context, job model.Job) (Result, error) {
	manifest, err := buildJobManifest(job, e.Namespace)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Logs:     "dry-run only: manifest generated, job not submitted to Kubernetes",
		Manifest: manifest,
		ExitCode: 0,
	}, nil
}
