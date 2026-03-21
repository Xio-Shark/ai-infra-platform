package model

import (
	"errors"
	"fmt"
	"time"
)

type JobType string
type JobStatus string

const (
	JobTypeTraining  JobType = "training"
	JobTypeInference JobType = "inference"
	JobTypeBenchmark JobType = "benchmark"
)

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusScheduled JobStatus = "scheduled"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
	JobStatusCancelled JobStatus = "cancelled"
)

var ErrInvalidJob = errors.New("invalid job")

type Job struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           JobType           `json:"job_type"`
	Status         JobStatus         `json:"status"`
	Executor       string            `json:"executor"`
	Priority       int               `json:"priority"`
	ModelVersion   string            `json:"model_version"`
	DatasetVersion string            `json:"dataset_version"`
	ImageTag       string            `json:"image_tag"`
	ResourceSpec   ResourceSpec      `json:"resource_spec"`
	Command        []string          `json:"command"`
	Environment    map[string]string `json:"environment"`
	Metadata       map[string]string `json:"metadata"`
	MaxRetries     int               `json:"max_retries"`
	RetryCount     int               `json:"retry_count"`
	LastError      string            `json:"last_error,omitempty"`
	TraceID        string            `json:"trace_id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type CreateJobInput struct {
	Name           string            `json:"name"`
	Type           JobType           `json:"job_type"`
	Executor       string            `json:"executor"`
	Priority       int               `json:"priority"`
	ModelVersion   string            `json:"model_version"`
	DatasetVersion string            `json:"dataset_version"`
	ImageTag       string            `json:"image_tag"`
	ResourceSpec   ResourceSpec      `json:"resource_spec"`
	Command        []string          `json:"command"`
	Environment    map[string]string `json:"environment"`
	Metadata       map[string]string `json:"metadata"`
	MaxRetries     int               `json:"max_retries"`
}

func (input CreateJobInput) Validate() error {
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidJob)
	}
	switch input.Type {
	case JobTypeTraining, JobTypeInference, JobTypeBenchmark:
	default:
		return fmt.Errorf("%w: unsupported job_type %q", ErrInvalidJob, input.Type)
	}
	if input.Executor == "" {
		return fmt.Errorf("%w: executor is required", ErrInvalidJob)
	}
	if len(input.Command) == 0 {
		return fmt.Errorf("%w: command is required", ErrInvalidJob)
	}
	if err := input.ResourceSpec.Validate(); err != nil {
		return err
	}
	if input.MaxRetries < 0 {
		return fmt.Errorf("%w: max_retries cannot be negative", ErrInvalidJob)
	}
	return nil
}

func (job Job) CanSchedule() bool {
	return job.Status == JobStatusPending || job.Status == JobStatusRetrying
}

func (job Job) CanRun() bool {
	return job.Status == JobStatusScheduled
}

func (job Job) CanCancel() bool {
	return job.Status == JobStatusPending || job.Status == JobStatusScheduled || job.Status == JobStatusRetrying
}

func (job Job) CanRetry() bool {
	return job.Status == JobStatusFailed || job.Status == JobStatusRetrying
}

func (job Job) IsTerminal() bool {
	return job.Status == JobStatusSucceeded || job.Status == JobStatusFailed || job.Status == JobStatusCancelled
}
