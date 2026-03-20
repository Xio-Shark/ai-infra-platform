package worker

import (
	"context"
	"fmt"
	"log"

	"ai-job-orchestrator/internal/model"
)

// K8sJobExecutor is a stub: logs what would be submitted; wire client-go for real clusters.
type K8sJobExecutor struct{}

func (K8sJobExecutor) Run(ctx context.Context, job *model.Job) error {
	_ = ctx
	log.Printf("[k8s stub] would create Job for %s image=%s spec=%s", job.ID, job.ImageTag, job.ResourceSpec)
	return fmt.Errorf("k8s executor not implemented; use shell executor for local demo")
}
