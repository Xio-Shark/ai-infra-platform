package model

import "time"

type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSucceeded ExecutionStatus = "succeeded"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

type Execution struct {
	ID         string          `json:"id"`
	JobID      string          `json:"job_id"`
	Attempt    int             `json:"attempt"`
	Status     ExecutionStatus `json:"status"`
	Executor   string          `json:"executor"`
	TraceID    string          `json:"trace_id"`
	Logs       string          `json:"logs,omitempty"`
	Manifest   string          `json:"manifest,omitempty"`
	ExitCode   int             `json:"exit_code"`
	Error      string          `json:"error,omitempty"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at,omitempty"`
}
