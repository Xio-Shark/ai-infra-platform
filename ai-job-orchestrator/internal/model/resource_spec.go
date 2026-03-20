package model

// ResourceSpec is stored as JSON string on Job; parse in callers when needed.
type ResourceSpec struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	GPU    int    `json:"gpu,omitempty"`
}
