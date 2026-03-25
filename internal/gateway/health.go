package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ai-infra-platform/internal/model"
)

// HealthChecker periodically probes backend /health endpoints
// and marks them as healthy or unhealthy.
type HealthChecker struct {
	mu       sync.RWMutex
	backends []model.Backend
	client   *http.Client
	interval time.Duration
	cancel   context.CancelFunc
}

// NewHealthChecker creates a checker with the given probe interval.
func NewHealthChecker(
	backends []model.Backend,
	interval, timeout time.Duration,
) *HealthChecker {
	return &HealthChecker{
		backends: backends,
		client:   &http.Client{Timeout: timeout},
		interval: interval,
	}
}

// Start begins background health probing. Call Stop() to terminate.
func (hc *HealthChecker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	hc.cancel = cancel

	// Initial probe
	hc.probeAll()

	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hc.probeAll()
			}
		}
	}()
}

// Stop terminates background probing.
func (hc *HealthChecker) Stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
}

// GetHealthyBackends returns currently healthy backends.
func (hc *HealthChecker) GetHealthyBackends() []model.Backend {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	healthy := make([]model.Backend, 0, len(hc.backends))
	for _, b := range hc.backends {
		if b.Healthy {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

// GetAllBackends returns all backends with their health status.
func (hc *HealthChecker) GetAllBackends() []model.Backend {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	cp := make([]model.Backend, len(hc.backends))
	copy(cp, hc.backends)
	return cp
}

func (hc *HealthChecker) probeAll() {
	var wg sync.WaitGroup
	for i := range hc.backends {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			healthy := hc.probe(hc.backends[idx].Endpoint)
			hc.mu.Lock()
			hc.backends[idx].Healthy = healthy
			hc.mu.Unlock()
		}(i)
	}
	wg.Wait()
}

func (hc *HealthChecker) probe(endpoint string) bool {
	url := fmt.Sprintf("%s/health", endpoint)
	resp, err := hc.client.Get(url)
	if err != nil {
		log.Printf("[health] probe %s failed: %v", endpoint, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
