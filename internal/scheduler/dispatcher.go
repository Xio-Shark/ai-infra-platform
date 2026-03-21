package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
)

type Dispatcher struct {
	repo    store.Repository
	metrics *telemetry.Metrics
	tracer  *telemetry.Tracer
}

func NewDispatcher(repo store.Repository, metrics *telemetry.Metrics, tracer *telemetry.Tracer) *Dispatcher {
	return &Dispatcher{repo: repo, metrics: metrics, tracer: tracer}
}

func (d *Dispatcher) SchedulePending(ctx context.Context, limit int) ([]model.Job, error) {
	jobs, err := d.repo.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	candidates := make([]model.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.CanSchedule() {
			candidates = append(candidates, job)
		}
	}
	queue := NewPriorityQueue(candidates)
	scheduled := make([]model.Job, 0, limit)
	for queue.Len() > 0 && len(scheduled) < limit {
		job := heap.Pop(queue).(model.Job)
		next, err := d.markScheduled(ctx, job)
		if err != nil {
			return nil, err
		}
		scheduled = append(scheduled, next)
	}
	return scheduled, nil
}

func (d *Dispatcher) ScheduleJob(ctx context.Context, id string) (model.Job, error) {
	job, err := d.repo.GetJob(ctx, id)
	if err != nil {
		return model.Job{}, fmt.Errorf("get job %s: %w", id, err)
	}
	return d.markScheduled(ctx, job)
}

func (d *Dispatcher) markScheduled(ctx context.Context, job model.Job) (model.Job, error) {
	if !job.CanSchedule() {
		return model.Job{}, fmt.Errorf("job %s cannot be scheduled from status %s", job.ID, job.Status)
	}
	job.Status = model.JobStatusScheduled
	job.UpdatedAt = time.Now().UTC()
	if err := d.repo.UpdateJob(ctx, job); err != nil {
		return model.Job{}, fmt.Errorf("update scheduled job %s: %w", job.ID, err)
	}
	d.metrics.IncScheduled()
	d.tracer.Add(job.TraceID, "job.schedule", map[string]string{"job_id": job.ID})
	return job, nil
}
