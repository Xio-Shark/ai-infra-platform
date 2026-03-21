package service_test

import (
	"context"
	"testing"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

func TestJobLifecycleWithShellExecutor(t *testing.T) {
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	registry := worker.NewRegistry(worker.ShellExecutor{}, worker.K8sJobExecutor{})
	jobService := service.NewJobService(repo, metrics, tracer)
	executionService := service.NewExecutionService(repo, dispatcher, registry, metrics, tracer)

	job, err := jobService.CreateJob(context.Background(), model.CreateJobInput{
		Name:       "shell-success",
		Type:       model.JobTypeTraining,
		Executor:   "shell",
		Command:    []string{"/bin/sh", "-lc", "echo hello"},
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	execution, err := executionService.RunJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("RunJob() error = %v", err)
	}
	if execution.Status != model.ExecutionStatusSucceeded {
		t.Fatalf("execution status = %s, want %s", execution.Status, model.ExecutionStatusSucceeded)
	}
	stored, err := jobService.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if stored.Status != model.JobStatusSucceeded {
		t.Fatalf("job status = %s, want %s", stored.Status, model.JobStatusSucceeded)
	}
}

func TestRetryPath(t *testing.T) {
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	registry := worker.NewRegistry(worker.ShellExecutor{})
	jobService := service.NewJobService(repo, metrics, tracer)
	executionService := service.NewExecutionService(repo, dispatcher, registry, metrics, tracer)

	job, err := jobService.CreateJob(context.Background(), model.CreateJobInput{
		Name:       "shell-failure",
		Type:       model.JobTypeBenchmark,
		Executor:   "shell",
		Command:    []string{"/bin/sh", "-lc", "exit 7"},
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if _, err := executionService.RunJob(context.Background(), job.ID); err == nil {
		t.Fatal("RunJob() error = nil, want failure")
	}
	stored, err := jobService.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if stored.Status != model.JobStatusRetrying {
		t.Fatalf("job status = %s, want %s", stored.Status, model.JobStatusRetrying)
	}
	retried, err := jobService.RetryJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("RetryJob() error = %v", err)
	}
	if retried.Status != model.JobStatusPending {
		t.Fatalf("retry status = %s, want %s", retried.Status, model.JobStatusPending)
	}
}
