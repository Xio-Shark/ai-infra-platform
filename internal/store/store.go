package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"ai-infra-platform/internal/model"
)

var ErrNotFound = errors.New("resource not found")
var ErrNotImplemented = errors.New("store backend is not implemented in the MVP")

type Repository interface {
	CreateJob(ctx context.Context, job model.Job) error
	UpdateJob(ctx context.Context, job model.Job) error
	GetJob(ctx context.Context, id string) (model.Job, error)
	ListJobs(ctx context.Context) ([]model.Job, error)
	CreateExecution(ctx context.Context, execution model.Execution) error
	UpdateExecution(ctx context.Context, execution model.Execution) error
	ListExecutions(ctx context.Context, jobID string) ([]model.Execution, error)

	// Node management
	RegisterNode(ctx context.Context, node model.Node) error
	UpdateNode(ctx context.Context, node model.Node) error
	GetNode(ctx context.Context, id string) (model.Node, error)
	ListOnlineNodes(ctx context.Context) ([]model.Node, error)
	AllocateGPU(ctx context.Context, nodeID string, count int) error
	ReleaseGPU(ctx context.Context, nodeID string, count int) error
}

type MemoryStore struct {
	mu         sync.RWMutex
	jobs       map[string]model.Job
	executions map[string]model.Execution
	jobIndex   map[string][]string
	nodes      map[string]model.Node
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:       make(map[string]model.Job),
		executions: make(map[string]model.Execution),
		jobIndex:   make(map[string][]string),
		nodes:      make(map[string]model.Node),
	}
}

func (s *MemoryStore) CreateJob(_ context.Context, job model.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	return nil
}

func (s *MemoryStore) UpdateJob(_ context.Context, job model.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; !ok {
		return ErrNotFound
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *MemoryStore) GetJob(_ context.Context, id string) (model.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return model.Job{}, ErrNotFound
	}
	return job, nil
}

func (s *MemoryStore) ListJobs(_ context.Context) ([]model.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		items = append(items, job)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].Priority > items[j].Priority
	})
	return items, nil
}

func (s *MemoryStore) CreateExecution(_ context.Context, execution model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[execution.ID] = execution
	s.jobIndex[execution.JobID] = append(s.jobIndex[execution.JobID], execution.ID)
	return nil
}

func (s *MemoryStore) UpdateExecution(_ context.Context, execution model.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.executions[execution.ID]; !ok {
		return ErrNotFound
	}
	s.executions[execution.ID] = execution
	return nil
}

func (s *MemoryStore) ListExecutions(_ context.Context, jobID string) ([]model.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.jobIndex[jobID]
	items := make([]model.Execution, 0, len(ids))
	for _, id := range ids {
		items = append(items, s.executions[id])
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Attempt < items[j].Attempt
	})
	return items, nil
}

// ---- Node management ----

func (s *MemoryStore) RegisterNode(_ context.Context, node model.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryStore) UpdateNode(_ context.Context, node model.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[node.ID]; !ok {
		return ErrNotFound
	}
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryStore) GetNode(_ context.Context, id string) (model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[id]
	if !ok {
		return model.Node{}, ErrNotFound
	}
	return node, nil
}

func (s *MemoryStore) ListOnlineNodes(_ context.Context) ([]model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]model.Node, 0)
	for _, n := range s.nodes {
		if n.IsSchedulable() {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (s *MemoryStore) AllocateGPU(_ context.Context, nodeID string, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[nodeID]
	if !ok {
		return ErrNotFound
	}
	if node.Resource.AvailGPU < count {
		return fmt.Errorf("insufficient GPU: avail=%d, requested=%d", node.Resource.AvailGPU, count)
	}
	node.Resource.AvailGPU -= count
	s.nodes[nodeID] = node
	return nil
}

func (s *MemoryStore) ReleaseGPU(_ context.Context, nodeID string, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[nodeID]
	if !ok {
		return ErrNotFound
	}
	node.Resource.AvailGPU += count
	if node.Resource.AvailGPU > node.Resource.TotalGPU {
		node.Resource.AvailGPU = node.Resource.TotalGPU
	}
	s.nodes[nodeID] = node
	return nil
}
