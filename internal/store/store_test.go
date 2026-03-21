package store

import (
	"context"
	"testing"
	"time"

	"ai-infra-platform/internal/model"
)

func TestMemoryStore_JobCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	job := model.Job{
		ID:        "job-1",
		Name:      "test",
		Status:    model.JobStatusPending,
		Priority:  5,
		CreatedAt: time.Now(),
	}

	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.GetJob(ctx, "job-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("name: want test, got %s", got.Name)
	}

	got.Status = model.JobStatusRunning
	if err := s.UpdateJob(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, _ := s.GetJob(ctx, "job-1")
	if updated.Status != model.JobStatusRunning {
		t.Errorf("status: want running, got %s", updated.Status)
	}

	_, err = s.GetJob(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = s.UpdateJob(ctx, model.Job{ID: "nonexistent"})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_ListJobs_Priority(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	now := time.Now()

	jobs := []model.Job{
		{ID: "low", Priority: 1, CreatedAt: now},
		{ID: "high", Priority: 10, CreatedAt: now},
		{ID: "med", Priority: 5, CreatedAt: now},
	}
	for _, j := range jobs {
		_ = s.CreateJob(ctx, j)
	}

	list, err := s.ListJobs(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("count: want 3, got %d", len(list))
	}
	if list[0].ID != "high" {
		t.Errorf("first should be high priority, got %s", list[0].ID)
	}
}

func TestMemoryStore_ExecutionCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	exec := model.Execution{
		ID:      "exec-1",
		JobID:   "job-1",
		Attempt: 1,
		Status:  model.ExecutionStatusRunning,
	}
	if err := s.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("create: %v", err)
	}

	exec.Status = model.ExecutionStatusSucceeded
	if err := s.UpdateExecution(ctx, exec); err != nil {
		t.Fatalf("update: %v", err)
	}

	list, err := s.ListExecutions(ctx, "job-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count: want 1, got %d", len(list))
	}
	if list[0].Status != model.ExecutionStatusSucceeded {
		t.Errorf("status: want succeeded, got %s", list[0].Status)
	}

	err = s.UpdateExecution(ctx, model.Execution{ID: "nonexistent"})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
