package model

import "time"

type JobType string

const (
	JobTraining  JobType = "training"
	JobInference JobType = "inference"
	JobEval      JobType = "eval"
	JobBenchmark JobType = "benchmark"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusScheduled  JobStatus = "scheduled"
	StatusRunning    JobStatus = "running"
	StatusSucceeded  JobStatus = "succeeded"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
	StatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	Type            JobType   `json:"job_type"`
	Payload         string    `json:"payload,omitempty"`
	ModelVersion    string    `json:"model_version,omitempty"`
	DatasetVersion  string    `json:"dataset_version,omitempty"`
	ImageTag        string    `json:"image_tag,omitempty"`
	ResourceSpec    string    `json:"resource_spec,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateJobRequest struct {
	JobType         JobType `json:"job_type"` // training | inference | eval | benchmark
	Payload         string  `json:"payload"`
	ModelVersion    string  `json:"model_version"`
	DatasetVersion  string  `json:"dataset_version"`
	ImageTag        string  `json:"image_tag"`
	ResourceSpec    string  `json:"resource_spec"`
}
