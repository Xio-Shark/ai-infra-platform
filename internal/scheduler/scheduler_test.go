package scheduler

import (
	"context"
	"testing"
	"time"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
)

func TestDispatcher_ScheduleJob(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := NewDispatcher(repo, metrics, tracer)

	job := model.Job{
		ID:        "job-1",
		Name:      "test",
		Status:    model.JobStatusPending,
		Priority:  5,
		CreatedAt: time.Now(),
	}
	_ = repo.CreateJob(ctx, job)

	scheduled, err := dispatcher.ScheduleJob(ctx, "job-1")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if scheduled.Status != model.JobStatusScheduled {
		t.Errorf("status: want scheduled, got %s", scheduled.Status)
	}
}

func TestDispatcher_SchedulePending(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := NewDispatcher(repo, metrics, tracer)

	for i, name := range []string{"a", "b", "c"} {
		_ = repo.CreateJob(ctx, model.Job{
			ID:        name,
			Name:      name,
			Status:    model.JobStatusPending,
			Priority:  i,
			CreatedAt: time.Now(),
		})
	}

	scheduled, err := dispatcher.SchedulePending(ctx, 2)
	if err != nil {
		t.Fatalf("schedule pending: %v", err)
	}
	if len(scheduled) != 2 {
		t.Errorf("count: want 2, got %d", len(scheduled))
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
