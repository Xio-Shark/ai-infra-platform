package worker

import (
	"context"

	"ai-job-orchestrator/internal/model"
)

type Executor interface {
	Run(ctx context.Context, job *model.Job) error
}
