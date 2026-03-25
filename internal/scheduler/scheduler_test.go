package scheduler

import (
	"context"
	"testing"
	"time"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
)

func setupDispatcher(t *testing.T) (*Dispatcher, *store.MemoryStore) {
	t.Helper()
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	return NewDispatcher(repo, metrics, tracer), repo
}

func TestDispatcher_ScheduleJob_CPUOnly(t *testing.T) {
	ctx := context.Background()
	d, repo := setupDispatcher(t)

	job := model.Job{
		ID:       "cpu-1",
		Name:     "cpu-task",
		Status:   model.JobStatusPending,
		Priority: 5,
		ResourceSpec: model.ResourceSpec{GPU: 0},
		CreatedAt: time.Now(),
	}
	_ = repo.CreateJob(ctx, job)

	scheduled, err := d.ScheduleJob(ctx, "cpu-1")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if scheduled.Status != model.JobStatusScheduled {
		t.Errorf("status: want scheduled, got %s", scheduled.Status)
	}
}

func TestDispatcher_ScheduleJob_GPU_BestFit(t *testing.T) {
	ctx := context.Background()
	d, repo := setupDispatcher(t)

	// Register two GPU nodes
	_ = repo.RegisterNode(ctx, makeNode("big", model.NodeStatusOnline, 8, 8, 81920))
	_ = repo.RegisterNode(ctx, makeNode("small", model.NodeStatusOnline, 2, 2, 81920))

	// Create a GPU job needing 2 GPUs
	job := model.Job{
		ID:       "gpu-1",
		Name:     "train",
		Status:   model.JobStatusPending,
		Priority: 5,
		ResourceSpec: model.ResourceSpec{GPU: 2},
		CreatedAt: time.Now(),
	}
	_ = repo.CreateJob(ctx, job)

	scheduled, err := d.ScheduleJob(ctx, "gpu-1")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if scheduled.Status != model.JobStatusScheduled {
		t.Errorf("status: want scheduled, got %s", scheduled.Status)
	}
	// Best-Fit should pick "small" (residual=0)
	if scheduled.Metadata["assigned_node"] != "small" {
		t.Errorf("node: want small, got %s", scheduled.Metadata["assigned_node"])
	}

	// Verify GPU was allocated
	node, _ := repo.GetNode(ctx, "small")
	if node.Resource.AvailGPU != 0 {
		t.Errorf("avail GPU: want 0, got %d", node.Resource.AvailGPU)
	}
}

func TestDispatcher_ScheduleJob_GPU_NoFit(t *testing.T) {
	ctx := context.Background()
	d, repo := setupDispatcher(t)

	_ = repo.RegisterNode(ctx, makeNode("tiny", model.NodeStatusOnline, 1, 1, 81920))

	job := model.Job{
		ID:       "gpu-big",
		Name:     "big-train",
		Status:   model.JobStatusPending,
		Priority: 5,
		ResourceSpec: model.ResourceSpec{GPU: 4},
		CreatedAt: time.Now(),
	}
	_ = repo.CreateJob(ctx, job)

	_, err := d.ScheduleJob(ctx, "gpu-big")
	if err == nil {
		t.Error("expected error for no fit")
	}
}

func TestDispatcher_SchedulePending_ResourceAware(t *testing.T) {
	ctx := context.Background()
	d, repo := setupDispatcher(t)

	// 1 node with 4 GPUs
	_ = repo.RegisterNode(ctx, makeNode("n1", model.NodeStatusOnline, 4, 4, 81920))

	// 3 jobs: each needs 2 GPUs → only 2 should get scheduled
	for i, name := range []string{"j1", "j2", "j3"} {
		_ = repo.CreateJob(ctx, model.Job{
			ID:           name,
			Name:         name,
			Status:       model.JobStatusPending,
			Priority:     10 - i,
			ResourceSpec: model.ResourceSpec{GPU: 2},
			CreatedAt:    time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	scheduled, err := d.SchedulePending(ctx, 10)
	if err != nil {
		t.Fatalf("schedule pending: %v", err)
	}
	if len(scheduled) != 2 {
		t.Errorf("count: want 2, got %d", len(scheduled))
	}

	// Check GPU fully allocated
	node, _ := repo.GetNode(ctx, "n1")
	if node.Resource.AvailGPU != 0 {
		t.Errorf("avail GPU: want 0, got %d", node.Resource.AvailGPU)
	}
}

func TestDispatcher_ReleaseJobResources(t *testing.T) {
	ctx := context.Background()
	d, repo := setupDispatcher(t)

	_ = repo.RegisterNode(ctx, makeNode("n1", model.NodeStatusOnline, 4, 4, 81920))

	job := model.Job{
		ID:           "rel-1",
		Name:         "release-test",
		Status:       model.JobStatusPending,
		Priority:     5,
		ResourceSpec: model.ResourceSpec{GPU: 2},
		CreatedAt:    time.Now(),
	}
	_ = repo.CreateJob(ctx, job)

	scheduled, err := d.ScheduleJob(ctx, "rel-1")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}

	// Release resources
	if err := d.ReleaseJobResources(ctx, scheduled); err != nil {
		t.Fatalf("release: %v", err)
	}

	node, _ := repo.GetNode(ctx, "n1")
	if node.Resource.AvailGPU != 4 {
		t.Errorf("avail GPU after release: want 4, got %d", node.Resource.AvailGPU)
	}
}

func TestCanRetry(t *testing.T) {
	retriable := model.Job{
		Status:     model.JobStatusFailed,
		MaxRetries: 3,
		RetryCount: 1,
	}
	if !CanRetry(retriable) {
		t.Error("expected retriable")
	}

	exhausted := model.Job{
		Status:     model.JobStatusFailed,
		MaxRetries: 2,
		RetryCount: 2,
	}
	if CanRetry(exhausted) {
		t.Error("expected not retriable (exhausted)")
	}
}
