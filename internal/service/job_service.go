package service

import (
	"context"
	"fmt"
	"time"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
)

type JobService struct {
	repo    store.Repository
	metrics *telemetry.Metrics
	tracer  *telemetry.Tracer
}

func NewJobService(repo store.Repository, metrics *telemetry.Metrics, tracer *telemetry.Tracer) *JobService {
	return &JobService{repo: repo, metrics: metrics, tracer: tracer}
}

func (s *JobService) CreateJob(ctx context.Context, input model.CreateJobInput) (model.Job, error) {
	if err := input.Validate(); err != nil {
		return model.Job{}, err
	}
	now := time.Now().UTC()
	traceID := s.tracer.NewTrace("job.submit")
	job := model.Job{
		ID:             nextID("job"),
		Name:           input.Name,
		Type:           input.Type,
		Status:         model.JobStatusPending,
		Executor:       input.Executor,
		Priority:       input.Priority,
		ModelVersion:   input.ModelVersion,
		DatasetVersion: input.DatasetVersion,
		ImageTag:       input.ImageTag,
		ResourceSpec:   input.ResourceSpec,
		Command:        append([]string(nil), input.Command...),
		Environment:    cloneMap(input.Environment),
		Metadata:       cloneMap(input.Metadata),
		MaxRetries:     input.MaxRetries,
		TraceID:        traceID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return model.Job{}, fmt.Errorf("create job: %w", err)
	}
	s.metrics.IncSubmitted()
	s.tracer.Add(traceID, "job.persisted", map[string]string{"job_id": job.ID})
	return job, nil
}

func (s *JobService) GetJob(ctx context.Context, id string) (model.Job, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return model.Job{}, fmt.Errorf("get job %s: %w", id, err)
	}
	return job, nil
}

func (s *JobService) ListJobs(ctx context.Context) ([]model.Job, error) {
	items, err := s.repo.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return items, nil
}

func (s *JobService) CancelJob(ctx context.Context, id string) (model.Job, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return model.Job{}, err
	}
	if !job.CanCancel() {
		return model.Job{}, fmt.Errorf("job %s cannot be cancelled from status %s", id, job.Status)
	}
	job.Status = model.JobStatusCancelled
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return model.Job{}, fmt.Errorf("cancel job %s: %w", id, err)
	}
	s.metrics.IncCancelled()
	s.tracer.Add(job.TraceID, "job.cancelled", map[string]string{"job_id": job.ID})
	return job, nil
}

func (s *JobService) RetryJob(ctx context.Context, id string) (model.Job, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return model.Job{}, err
	}
	if !job.CanRetry() {
		return model.Job{}, fmt.Errorf("job %s cannot be retried from status %s", id, job.Status)
	}
	job.Status = model.JobStatusPending
	job.LastError = ""
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return model.Job{}, fmt.Errorf("retry job %s: %w", id, err)
	}
	s.metrics.IncRetried()
	s.tracer.Add(job.TraceID, "job.retry_requested", map[string]string{"job_id": job.ID})
	return job, nil
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
