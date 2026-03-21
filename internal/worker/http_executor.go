package worker

import (
	"context"
	"fmt"

	"ai-infra-platform/internal/model"
)

type HTTPExecutor struct{}

func (e HTTPExecutor) Name() string { return "http" }

func (e HTTPExecutor) Execute(_ context.Context, job model.Job) (Result, error) {
	return Result{}, fmt.Errorf("http executor is not implemented in the MVP for job %s", job.ID)
}
