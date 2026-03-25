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

// Dispatcher schedules jobs to worker nodes with resource awareness.
// Core scheduling flow:
//  1. Fetch pending jobs, rank by priority (heap)
//  2. For each job, query online nodes from store
//  3. Best-Fit match: find the node with least residual GPU after allocation
//  4. Atomically allocate GPU on chosen node
//  5. Mark job as scheduled with assigned node ID
type Dispatcher struct {
	repo     store.Repository
	metrics  *telemetry.Metrics
	tracer   *telemetry.Tracer
	strategy MatchStrategy
}

func NewDispatcher(
	repo store.Repository,
	metrics *telemetry.Metrics,
	tracer *telemetry.Tracer,
) *Dispatcher {
	return &Dispatcher{
		repo:     repo,
		metrics:  metrics,
		tracer:   tracer,
		strategy: StrategyBestFit,
	}
}

// SetStrategy allows switching between BestFit and FirstFit.
func (d *Dispatcher) SetStrategy(s MatchStrategy) {
	d.strategy = s
}

// SchedulePending fetches all schedulable jobs and attempts resource-aware
// placement on available nodes. Jobs that require GPUs are matched to nodes
// via Best-Fit; CPU-only jobs (GPU=0) are scheduled without node constraint.
func (d *Dispatcher) SchedulePending(ctx context.Context, limit int) ([]model.Job, error) {
	jobs, err := d.repo.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	candidates := filterSchedulable(jobs)
	queue := NewPriorityQueue(candidates)

	scheduled := make([]model.Job, 0, limit)
	for queue.Len() > 0 && len(scheduled) < limit {
		job := heap.Pop(queue).(model.Job)
		placed, placeErr := d.placeJob(ctx, job)
		if placeErr != nil {
			// No fit for this job — skip, try next
			d.tracer.Add(job.TraceID, "job.schedule.skip",
				map[string]string{"job_id": job.ID, "reason": placeErr.Error()})
			continue
		}
		scheduled = append(scheduled, placed)
	}
	return scheduled, nil
}

// ScheduleJob schedules a single job by ID with resource matching.
func (d *Dispatcher) ScheduleJob(ctx context.Context, id string) (model.Job, error) {
	job, err := d.repo.GetJob(ctx, id)
	if err != nil {
		return model.Job{}, fmt.Errorf("get job %s: %w", id, err)
	}
	return d.placeJob(ctx, job)
}

// placeJob is the core scheduling decision: match resources → allocate → mark.
func (d *Dispatcher) placeJob(ctx context.Context, job model.Job) (model.Job, error) {
	if !job.CanSchedule() {
		return model.Job{}, fmt.Errorf(
			"job %s cannot be scheduled from status %s", job.ID, job.Status)
	}

	gpuNeeded := job.ResourceSpec.GPU
	if gpuNeeded > 0 {
		return d.placeGPUJob(ctx, job)
	}
	return d.markScheduled(ctx, job, "")
}

// placeGPUJob handles GPU-requiring jobs: find fitting node, allocate, assign.
func (d *Dispatcher) placeGPUJob(ctx context.Context, job model.Job) (model.Job, error) {
	nodes, err := d.repo.ListOnlineNodes(ctx)
	if err != nil {
		return model.Job{}, fmt.Errorf("list online nodes: %w", err)
	}

	target, err := MatchNode(nodes, job.ResourceSpec, d.strategy)
	if err != nil {
		return model.Job{}, fmt.Errorf("match node for job %s: %w", job.ID, err)
	}

	if err := d.repo.AllocateGPU(ctx, target.ID, job.ResourceSpec.GPU); err != nil {
		return model.Job{}, fmt.Errorf("allocate GPU on %s: %w", target.ID, err)
	}

	return d.markScheduled(ctx, job, target.ID)
}

func (d *Dispatcher) markScheduled(
	ctx context.Context, job model.Job, nodeID string,
) (model.Job, error) {
	job.Status = model.JobStatusScheduled
	job.UpdatedAt = time.Now().UTC()
	if nodeID != "" {
		if job.Metadata == nil {
			job.Metadata = make(map[string]string)
		}
		job.Metadata["assigned_node"] = nodeID
	}
	if err := d.repo.UpdateJob(ctx, job); err != nil {
		return model.Job{}, fmt.Errorf("update scheduled job %s: %w", job.ID, err)
	}
	d.metrics.IncScheduled()
	attrs := map[string]string{"job_id": job.ID}
	if nodeID != "" {
		attrs["node_id"] = nodeID
	}
	d.tracer.Add(job.TraceID, "job.schedule", attrs)
	return job, nil
}

// ReleaseJobResources returns GPU resources to the node when a job completes.
func (d *Dispatcher) ReleaseJobResources(ctx context.Context, job model.Job) error {
	nodeID := job.Metadata["assigned_node"]
	if nodeID == "" || job.ResourceSpec.GPU == 0 {
		return nil
	}
	return d.repo.ReleaseGPU(ctx, nodeID, job.ResourceSpec.GPU)
}

func filterSchedulable(jobs []model.Job) []model.Job {
	out := make([]model.Job, 0, len(jobs))
	for _, j := range jobs {
		if j.CanSchedule() {
			out = append(out, j)
		}
	}
	return out
}
