package service

import (
	"context"
	"database/sql"

	"ai-job-orchestrator/internal/model"
	"ai-job-orchestrator/internal/store"
)

type ExecutionService struct {
	DB *sql.DB
}

func (s *ExecutionService) ListByJob(ctx context.Context, jobID string) ([]model.Execution, error) {
	return store.ListExecutions(ctx, s.DB, jobID)
}
