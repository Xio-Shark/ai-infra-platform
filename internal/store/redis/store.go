package redis

import (
	"context"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) CreateJob(_ context.Context, _ model.Job) error {
	return store.ErrNotImplemented
}

func (s *Store) UpdateJob(_ context.Context, _ model.Job) error {
	return store.ErrNotImplemented
}

func (s *Store) GetJob(_ context.Context, _ string) (model.Job, error) {
	return model.Job{}, store.ErrNotImplemented
}

func (s *Store) ListJobs(_ context.Context) ([]model.Job, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) CreateExecution(_ context.Context, _ model.Execution) error {
	return store.ErrNotImplemented
}

func (s *Store) UpdateExecution(_ context.Context, _ model.Execution) error {
	return store.ErrNotImplemented
}

func (s *Store) ListExecutions(_ context.Context, _ string) ([]model.Execution, error) {
	return nil, store.ErrNotImplemented
}
