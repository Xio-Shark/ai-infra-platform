package scheduler

import (
	"time"

	"ai-infra-platform/internal/model"
)

func CanRetry(job model.Job) bool {
	return job.RetryCount < job.MaxRetries
}

func MarkRetrying(job model.Job, errText string) model.Job {
	job.RetryCount++
	job.Status = model.JobStatusRetrying
	job.LastError = errText
	job.UpdatedAt = time.Now().UTC()
	return job
}
