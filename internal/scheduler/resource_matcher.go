package scheduler

import (
	"fmt"
	"sort"
	"strings"

	"ai-infra-platform/internal/model"
)

// MatchStrategy defining how the scheduler picks a node for a job.
type MatchStrategy int

const (
	// StrategyBestFit picks the node with least remaining free GPUs after allocation.
	// Minimizes fragmentation.
	StrategyBestFit MatchStrategy = iota
	// StrategyFirstFit picks the first node that satisfies the request.
	// Minimizes scheduling latency.
	StrategyFirstFit
)

// NodeScore is used internally to rank candidate nodes.
type NodeScore struct {
	Node     model.Node
	Residual int // remaining free GPUs after allocation
}

// MatchNode selects the best worker node for the given resource request.
// Returns ErrNoFit if no node has enough resources.
func MatchNode(
	nodes []model.Node,
	spec model.ResourceSpec,
	strategy MatchStrategy,
) (model.Node, error) {
	gpuNeeded := spec.GPU
	gpuMemMiB := parseGPUMemoryMiB(spec.GPUMemory)

	var candidates []NodeScore
	for _, node := range nodes {
		if !node.IsSchedulable() {
			continue
		}
		if !node.CanFitGPU(gpuNeeded) {
			continue
		}
		if gpuMemMiB > 0 && !node.CanFitGPUMemory(gpuNeeded, gpuMemMiB) {
			continue
		}
		candidates = append(candidates, NodeScore{
			Node:     node,
			Residual: node.Resource.AvailGPU - gpuNeeded,
		})
	}
	if len(candidates) == 0 {
		return model.Node{}, fmt.Errorf(
			"no node fits: need %d GPU(s) with %d MiB each",
			gpuNeeded, gpuMemMiB,
		)
	}

	switch strategy {
	case StrategyFirstFit:
		return candidates[0].Node, nil
	default: // BestFit
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Residual < candidates[j].Residual
		})
		return candidates[0].Node, nil
	}
}

// parseGPUMemoryMiB converts gpu_memory string (e.g. "16Gi", "8192")
// to MiB. Returns 0 if empty or unparseable.
func parseGPUMemoryMiB(s string) int {
	if s == "" {
		return 0
	}
	var val int
	if strings.HasSuffix(s, "Mi") {
		if _, err := fmt.Sscanf(s, "%dMi", &val); err == nil {
			return val
		}
	}
	if strings.HasSuffix(s, "Gi") {
		if _, err := fmt.Sscanf(s, "%dGi", &val); err == nil {
			return val * 1024
		}
	}
	if _, err := fmt.Sscanf(s, "%d", &val); err == nil {
		return val
	}
	return 0
}
