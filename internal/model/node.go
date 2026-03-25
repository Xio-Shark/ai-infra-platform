package model

import (
	"fmt"
	"time"
)

// NodeStatus represents the health state of a worker node.
type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusDrain   NodeStatus = "drain"
)

// GPUInfo describes a single GPU device on a node.
type GPUInfo struct {
	Index       int    `json:"index"`
	Model       string `json:"model"`
	TotalMemMiB int    `json:"total_mem_mib"`
	UsedMemMiB  int    `json:"used_mem_mib"`
	Utilization int    `json:"utilization"` // 0-100%
}

// FreeMemMiB returns available GPU memory.
func (g GPUInfo) FreeMemMiB() int {
	return g.TotalMemMiB - g.UsedMemMiB
}

// NodeResource aggregates a node's allocatable resources.
type NodeResource struct {
	CPUCores   int       `json:"cpu_cores"`
	MemoryMiB  int       `json:"memory_mib"`
	GPUs       []GPUInfo `json:"gpus"`
	TotalGPU   int       `json:"total_gpu"`
	AvailGPU   int       `json:"avail_gpu"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Node represents a registered worker node in the cluster.
type Node struct {
	ID         string       `json:"id"`
	Hostname   string       `json:"hostname"`
	Status     NodeStatus   `json:"status"`
	Labels     map[string]string `json:"labels,omitempty"`
	Resource   NodeResource `json:"resource"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt  time.Time    `json:"created_at"`
}

// IsSchedulable checks whether this node can receive new jobs.
func (n Node) IsSchedulable() bool {
	return n.Status == NodeStatusOnline
}

// CanFitGPU returns true if the node has enough free GPUs.
func (n Node) CanFitGPU(requested int) bool {
	return n.Resource.AvailGPU >= requested
}

// CanFitGPUMemory checks if any `count` GPUs have at least minMemMiB free.
func (n Node) CanFitGPUMemory(count, minMemMiB int) bool {
	if minMemMiB <= 0 {
		return n.CanFitGPU(count)
	}
	matched := 0
	for _, gpu := range n.Resource.GPUs {
		if gpu.FreeMemMiB() >= minMemMiB {
			matched++
		}
	}
	return matched >= count
}

// Validate performs basic sanity checks on the node data.
func (n Node) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("node id is required")
	}
	if n.Hostname == "" {
		return fmt.Errorf("node hostname is required")
	}
	return nil
}
