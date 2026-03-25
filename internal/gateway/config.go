package gateway

import (
	"time"

	"ai-infra-platform/internal/model"
)

// Config holds the full gateway configuration.
type Config struct {
	ListenAddr     string            `json:"listen_addr"`
	Backends       []model.Backend   `json:"backends"`
	RateLimits     map[string]int    `json:"rate_limits"`     // model -> requests/sec
	DefaultRPS     int               `json:"default_rps"`     // fallback rate limit
	RequestTimeout time.Duration     `json:"request_timeout"`
	HealthInterval time.Duration     `json:"health_interval"` // probe interval
	HealthTimeout  time.Duration     `json:"health_timeout"`
}

// DefaultConfig returns sane defaults for local development.
func DefaultConfig() Config {
	return Config{
		ListenAddr:     ":9090",
		DefaultRPS:     100,
		RequestTimeout: 60 * time.Second,
		HealthInterval: 10 * time.Second,
		HealthTimeout:  3 * time.Second,
	}
}
