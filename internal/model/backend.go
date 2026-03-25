package model

// Backend represents a downstream LLM inference endpoint.
type Backend struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"` // e.g. "http://localhost:8000"
	Models   []string `json:"models"`   // models served, e.g. ["qwen2.5-7b"]
	Weight   int      `json:"weight"`   // load-balancing weight (higher = more traffic)
	Healthy  bool     `json:"healthy"`
}

// SupportsModel checks if this backend can serve the given model.
func (b Backend) SupportsModel(model string) bool {
	for _, m := range b.Models {
		if m == model {
			return true
		}
	}
	return false
}
