package model

import "fmt"

type ResourceSpec struct {
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
	GPU       int    `json:"gpu"`
	GPUMemory string `json:"gpu_memory"`
}

func (spec ResourceSpec) Validate() error {
	if spec.GPU < 0 {
		return fmt.Errorf("%w: gpu cannot be negative", ErrInvalidJob)
	}
	return nil
}
