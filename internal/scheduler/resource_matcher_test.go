package scheduler

import (
	"testing"
	"time"

	"ai-infra-platform/internal/model"
)

func makeNode(id string, status model.NodeStatus, totalGPU, availGPU int, memPerGPU int) model.Node {
	gpus := make([]model.GPUInfo, totalGPU)
	for i := range gpus {
		used := 0
		if i >= availGPU {
			used = memPerGPU // fully used
		}
		gpus[i] = model.GPUInfo{
			Index:       i,
			Model:       "A100-80G",
			TotalMemMiB: memPerGPU,
			UsedMemMiB:  used,
		}
	}
	return model.Node{
		ID:       id,
		Hostname: id,
		Status:   status,
		Resource: model.NodeResource{
			TotalGPU:  totalGPU,
			AvailGPU:  availGPU,
			GPUs:      gpus,
			UpdatedAt: time.Now(),
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}
}

func TestMatchNode_BestFit(t *testing.T) {
	nodes := []model.Node{
		makeNode("big", model.NodeStatusOnline, 8, 8, 81920),
		makeNode("mid", model.NodeStatusOnline, 4, 4, 81920),
		makeNode("small", model.NodeStatusOnline, 2, 2, 81920),
	}
	spec := model.ResourceSpec{GPU: 2}

	got, err := MatchNode(nodes, spec, StrategyBestFit)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	// Best-Fit should pick "small" (residual=0, least waste)
	if got.ID != "small" {
		t.Errorf("want node small, got %s", got.ID)
	}
}

func TestMatchNode_FirstFit(t *testing.T) {
	nodes := []model.Node{
		makeNode("big", model.NodeStatusOnline, 8, 8, 81920),
		makeNode("small", model.NodeStatusOnline, 2, 2, 81920),
	}
	spec := model.ResourceSpec{GPU: 2}

	got, err := MatchNode(nodes, spec, StrategyFirstFit)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	// FirstFit takes first candidate in order
	if got.ID != "big" {
		t.Errorf("want node big, got %s", got.ID)
	}
}

func TestMatchNode_NoFit(t *testing.T) {
	nodes := []model.Node{
		makeNode("tiny", model.NodeStatusOnline, 1, 1, 81920),
	}
	spec := model.ResourceSpec{GPU: 4}

	_, err := MatchNode(nodes, spec, StrategyBestFit)
	if err == nil {
		t.Error("expected error for no fit")
	}
}

func TestMatchNode_SkipsOffline(t *testing.T) {
	nodes := []model.Node{
		makeNode("offline", model.NodeStatusOffline, 8, 8, 81920),
		makeNode("drain", model.NodeStatusDrain, 4, 4, 81920),
		makeNode("ok", model.NodeStatusOnline, 2, 2, 81920),
	}
	spec := model.ResourceSpec{GPU: 2}

	got, err := MatchNode(nodes, spec, StrategyBestFit)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if got.ID != "ok" {
		t.Errorf("want node ok, got %s", got.ID)
	}
}

func TestMatchNode_GPUMemoryFilter(t *testing.T) {
	// "small-mem" has enough GPU count but not enough memory
	nodes := []model.Node{
		makeNode("small-mem", model.NodeStatusOnline, 4, 4, 8192), // 8Gi each
		makeNode("big-mem", model.NodeStatusOnline, 4, 4, 81920),  // 80Gi each
	}
	spec := model.ResourceSpec{GPU: 2, GPUMemory: "40Gi"}

	got, err := MatchNode(nodes, spec, StrategyBestFit)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if got.ID != "big-mem" {
		t.Errorf("want node big-mem, got %s", got.ID)
	}
}

func TestParseGPUMemoryMiB(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"16Gi", 16384},
		{"8192Mi", 8192},
		{"4096", 4096},
		{"invalid", 0},
	}
	for _, tc := range tests {
		got := parseGPUMemoryMiB(tc.input)
		if got != tc.want {
			t.Errorf("parseGPUMemoryMiB(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
