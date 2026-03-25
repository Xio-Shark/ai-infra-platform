package scheduler

import (
	"context"
	"time"

	"ai-infra-platform/internal/model"
)

// ResourceProvider abstracts how GPU/node resource info is collected.
// Production: NVMLProvider (nvidia-smi / go-nvml).
// Testing:    MockProvider with configurable data.
type ResourceProvider interface {
	// CollectNodeResource gathers current resource snapshot for a node.
	CollectNodeResource(ctx context.Context) (model.NodeResource, error)
}

// MockProvider returns pre-configured resource data for local/test use.
type MockProvider struct {
	GPUs      []model.GPUInfo
	CPUCores  int
	MemoryMiB int
}

// NewMockProvider creates a mock with the given GPU configuration.
func NewMockProvider(cpuCores, memoryMiB int, gpus []model.GPUInfo) *MockProvider {
	return &MockProvider{
		GPUs:      gpus,
		CPUCores:  cpuCores,
		MemoryMiB: memoryMiB,
	}
}

func (m *MockProvider) CollectNodeResource(_ context.Context) (model.NodeResource, error) {
	avail := 0
	for _, g := range m.GPUs {
		if g.Utilization < 95 { // <95% util = available
			avail++
		}
	}
	return model.NodeResource{
		CPUCores:  m.CPUCores,
		MemoryMiB: m.MemoryMiB,
		GPUs:      m.GPUs,
		TotalGPU:  len(m.GPUs),
		AvailGPU:  avail,
		UpdatedAt: time.Now().UTC(),
	}, nil
}
