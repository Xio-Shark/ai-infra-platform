package scheduler

import (
	"context"
	"database/sql"
	"errors"

	"ai-job-orchestrator/internal/model"
	"ai-job-orchestrator/internal/store"
)

func Tick(ctx context.Context, db *sql.DB) (*model.Job, error) {
	j, err := store.ClaimNextPending(ctx, db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}
