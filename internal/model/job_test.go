package model

import (
	"testing"
)

func TestCreateJobInput_Validate(t *testing.T) {
	valid := CreateJobInput{
		Name:     "test-job",
		Type:     JobTypeTraining,
		Executor: "shell",
		Command:  []string{"echo", "hello"},
		ResourceSpec: ResourceSpec{
			CPU:    "1",
			Memory: "1Gi",
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}

	tests := []struct {
		name   string
		modify func(input *CreateJobInput)
	}{
		{"empty name", func(in *CreateJobInput) { in.Name = "" }},
		{"bad job type", func(in *CreateJobInput) { in.Type = "unknown" }},
		{"empty executor", func(in *CreateJobInput) { in.Executor = "" }},
		{"empty command", func(in *CreateJobInput) { in.Command = nil }},
		{"negative retries", func(in *CreateJobInput) { in.MaxRetries = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.modify(&input)
			if err := input.Validate(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestJob_StateMachine(t *testing.T) {
	tests := []struct {
		status      JobStatus
		canSchedule bool
		canRun      bool
		canCancel   bool
		canRetry    bool
		isTerminal  bool
	}{
		{JobStatusPending, true, false, true, false, false},
		{JobStatusScheduled, false, true, true, false, false},
		{JobStatusRunning, false, false, false, false, false},
		{JobStatusSucceeded, false, false, false, false, true},
		{JobStatusFailed, false, false, false, true, true},
		{JobStatusRetrying, true, false, true, true, false},
		{JobStatusCancelled, false, false, false, false, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			job := Job{Status: tt.status}
			if got := job.CanSchedule(); got != tt.canSchedule {
				t.Errorf("CanSchedule: want %v, got %v", tt.canSchedule, got)
			}
			if got := job.CanRun(); got != tt.canRun {
				t.Errorf("CanRun: want %v, got %v", tt.canRun, got)
			}
			if got := job.CanCancel(); got != tt.canCancel {
				t.Errorf("CanCancel: want %v, got %v", tt.canCancel, got)
			}
			if got := job.CanRetry(); got != tt.canRetry {
				t.Errorf("CanRetry: want %v, got %v", tt.canRetry, got)
			}
			if got := job.IsTerminal(); got != tt.isTerminal {
				t.Errorf("IsTerminal: want %v, got %v", tt.isTerminal, got)
			}
		})
	}
}

func TestJobType_Values(t *testing.T) {
	types := []JobType{JobTypeTraining, JobTypeInference, JobTypeBenchmark}
	for _, jt := range types {
		if jt == "" {
			t.Error("job type should not be empty")
		}
	}
}
