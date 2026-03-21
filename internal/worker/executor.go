package worker

import (
	"context"
	"fmt"

	"ai-infra-platform/internal/model"
)

type Result struct {
	Logs     string
	Manifest string
	ExitCode int
}

type Executor interface {
	Name() string
	Execute(ctx context.Context, job model.Job) (Result, error)
}

type Registry struct {
	items map[string]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	registry := &Registry{items: make(map[string]Executor)}
	for _, executor := range executors {
		registry.items[executor.Name()] = executor
	}
	return registry
}

func (r *Registry) Get(name string) (Executor, error) {
	executor, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("executor %q is not registered", name)
	}
	return executor, nil
}
