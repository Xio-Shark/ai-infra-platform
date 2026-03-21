package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"ai-infra-platform/internal/model"
)

type ShellExecutor struct{}

func (e ShellExecutor) Name() string { return "shell" }

func (e ShellExecutor) Execute(ctx context.Context, job model.Job) (Result, error) {
	if len(job.Command) == 0 {
		return Result{}, fmt.Errorf("job %s has empty command", job.ID)
	}
	command := exec.CommandContext(ctx, job.Command[0], job.Command[1:]...)
	command.Env = append(os.Environ(), toEnv(job.Environment)...)
	output, err := command.CombinedOutput()
	result := Result{Logs: string(output), ExitCode: exitCode(err)}
	if err != nil {
		return result, fmt.Errorf("shell executor failed: %w", err)
	}
	return result, nil
}

func toEnv(values map[string]string) []string {
	items := make([]string, 0, len(values))
	for key, value := range values {
		items = append(items, key+"="+value)
	}
	return items
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if ok := errorAs(err, &exitErr); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}

func errorAs(err error, target any) bool {
	return errors.As(err, target)
}
