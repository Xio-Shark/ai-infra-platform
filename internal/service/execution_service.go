package service

import (
	"context"
	"fmt"
	"time"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

type ExecutionService struct {
	repo       store.Repository
	dispatcher *scheduler.Dispatcher
	registry   *worker.Registry
	metrics    *telemetry.Metrics
	tracer     *telemetry.Tracer
}

func NewExecutionService(
	repo store.Repository,
	dispatcher *scheduler.Dispatcher,
	registry *worker.Registry,
	metrics *telemetry.Metrics,
	tracer *telemetry.Tracer,
) *ExecutionService {
	return &ExecutionService{
		repo:       repo,
		dispatcher: dispatcher,
		registry:   registry,
		metrics:    metrics,
		tracer:     tracer,
	}
}

func (s *ExecutionService) RunJob(ctx context.Context, id string) (model.Execution, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return model.Execution{}, fmt.Errorf("get job %s: %w", id, err)
	}
	if job.Status != model.JobStatusScheduled {
		job, err = s.dispatcher.ScheduleJob(ctx, id)
		if err != nil {
			return model.Execution{}, err
		}
	}
	execution := model.Execution{
		ID:        nextID("exec"),
		JobID:     job.ID,
		Attempt:   job.RetryCount + 1,
		Status:    model.ExecutionStatusRunning,
		Executor:  job.Executor,
		TraceID:   job.TraceID,
		StartedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return model.Execution{}, fmt.Errorf("create execution: %w", err)
	}
	job.Status = model.JobStatusRunning
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return model.Execution{}, fmt.Errorf("mark job running: %w", err)
	}
	s.metrics.IncRunning()
	s.tracer.Add(job.TraceID, "job.run_started", map[string]string{"job_id": job.ID})
	executor, err := s.registry.Get(job.Executor)
	if err != nil {
		return execution, s.finishFailed(ctx, job, execution, fmt.Errorf("load executor: %w", err), worker.Result{})
	}
	result, runErr := executor.Execute(ctx, job)
	if runErr != nil {
		return execution, s.finishFailed(ctx, job, execution, runErr, result)
	}
	execution.Status = model.ExecutionStatusSucceeded
	execution.Logs = result.Logs
	execution.Manifest = result.Manifest
	execution.ExitCode = result.ExitCode
	execution.FinishedAt = time.Now().UTC()
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return model.Execution{}, fmt.Errorf("update execution %s: %w", execution.ID, err)
	}
	job.Status = model.JobStatusSucceeded
	job.UpdatedAt = execution.FinishedAt
	job.LastError = ""
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return model.Execution{}, fmt.Errorf("mark job succeeded: %w", err)
	}
	s.metrics.IncSucceeded()
	s.metrics.ObserveRuntime(execution.FinishedAt.Sub(execution.StartedAt))
	s.tracer.Add(job.TraceID, "job.run_succeeded", map[string]string{"job_id": job.ID})
	return execution, nil
}

func (s *ExecutionService) DispatchPending(ctx context.Context, limit int) ([]model.Execution, error) {
	scheduledJobs, err := s.dispatcher.SchedulePending(ctx, limit)
	if err != nil {
		return nil, err
	}
	executions := make([]model.Execution, 0, len(scheduledJobs))
	for _, job := range scheduledJobs {
		execution, runErr := s.RunJob(ctx, job.ID)
		if runErr != nil {
			return executions, runErr
		}
		executions = append(executions, execution)
	}
	return executions, nil
}

func (s *ExecutionService) ListExecutions(ctx context.Context, jobID string) ([]model.Execution, error) {
	items, err := s.repo.ListExecutions(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("list executions for job %s: %w", jobID, err)
	}
	return items, nil
}

func (s *ExecutionService) finishFailed(
	ctx context.Context,
	job model.Job,
	execution model.Execution,
	runErr error,
	result worker.Result,
) error {
	execution.Status = model.ExecutionStatusFailed
	execution.Logs = result.Logs
	execution.Manifest = result.Manifest
	execution.ExitCode = result.ExitCode
	execution.Error = runErr.Error()
	execution.FinishedAt = time.Now().UTC()
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return fmt.Errorf("update failed execution %s: %w", execution.ID, err)
	}
	if scheduler.CanRetry(job) {
		job = scheduler.MarkRetrying(job, runErr.Error())
		s.metrics.IncRetried()
		s.tracer.Add(job.TraceID, "job.run_failed_retryable", map[string]string{"job_id": job.ID})
	} else {
		job.Status = model.JobStatusFailed
		job.LastError = runErr.Error()
		job.UpdatedAt = execution.FinishedAt
		s.metrics.IncFailed()
		s.tracer.Add(job.TraceID, "job.run_failed", map[string]string{"job_id": job.ID})
	}
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("update failed job %s: %w", job.ID, err)
	}
	s.metrics.ObserveRuntime(execution.FinishedAt.Sub(execution.StartedAt))
	return fmt.Errorf("execute job %s: %w", job.ID, runErr)
}
