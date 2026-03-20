package worker

import (
	"context"
	"os/exec"
	"strings"

	"ai-job-orchestrator/internal/model"
)

type ShellExecutor struct{}

func (ShellExecutor) Run(ctx context.Context, job *model.Job) error {
	cmdStr := strings.TrimSpace(job.Payload)
	if cmdStr == "" {
		cmdStr = "echo noop"
	}
	c := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	return c.Run()
}
