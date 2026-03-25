package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"ai-infra-platform/internal/model"
)

// Gateway is the AI inference reverse-proxy gateway.
// Core responsibilities:
//  1. Parse model from OpenAI-compatible request body
//  2. Rate-limit per model (token bucket)
//  3. Select healthy backend via weighted random
//  4. Reverse proxy with SSE streaming support
//  5. Failover to next backend on 5xx / timeout
type Gateway struct {
	health  *HealthChecker
	limiter *RateLimiterRegistry
	client  *http.Client
	config  Config
}

// New creates a gateway from config.
func New(cfg Config) *Gateway {
	hc := NewHealthChecker(cfg.Backends, cfg.HealthInterval, cfg.HealthTimeout)
	return &Gateway{
		health:  hc,
		limiter: NewRateLimiterRegistry(cfg.DefaultRPS, cfg.RateLimits),
		client:  &http.Client{Timeout: cfg.RequestTimeout},
		config:  cfg,
	}
}

// Handler returns the HTTP handler for the gateway.
func (gw *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", gw.handleChatCompletions)
	mux.HandleFunc("/v1/completions", gw.handleCompletions)
	mux.HandleFunc("/gateway/health", gw.handleGatewayHealth)
	mux.HandleFunc("/gateway/backends", gw.handleListBackends)
	return mux
}

// Start begins background health checking.
func (gw *Gateway) Start() { gw.health.Start() }

// Stop terminates background health checking.
func (gw *Gateway) Stop() { gw.health.Stop() }

// --- HTTP handlers ---

func (gw *Gateway) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gw.proxyInference(w, r, "/v1/chat/completions")
}

func (gw *Gateway) handleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gw.proxyInference(w, r, "/v1/completions")
}

func (gw *Gateway) handleGatewayHealth(w http.ResponseWriter, _ *http.Request) {
	healthy := gw.health.GetHealthyBackends()
	status := "ok"
	if len(healthy) == 0 {
		status = "degraded"
	}
	writeGatewayJSON(w, http.StatusOK, map[string]any{
		"status":          status,
		"healthy_backends": len(healthy),
		"total_backends":   len(gw.config.Backends),
	})
}

func (gw *Gateway) handleListBackends(w http.ResponseWriter, _ *http.Request) {
	writeGatewayJSON(w, http.StatusOK, gw.health.GetAllBackends())
}

// --- Core proxy logic ---

// proxyInference is the central routing engine.
func (gw *Gateway) proxyInference(w http.ResponseWriter, r *http.Request, path string) {
	// 1. Read body to extract model
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	modelName := extractModel(body)
	if modelName == "" {
		writeGatewayError(w, http.StatusBadRequest, "model field is required")
		return
	}

	// 2. Rate limit
	if !gw.limiter.Allow(modelName) {
		writeGatewayError(w, http.StatusTooManyRequests,
			fmt.Sprintf("rate limit exceeded for model %s", modelName))
		return
	}

	// 3. Find healthy backends for this model
	candidates := gw.findBackends(modelName)
	if len(candidates) == 0 {
		writeGatewayError(w, http.StatusServiceUnavailable,
			fmt.Sprintf("no healthy backend for model %s", modelName))
		return
	}

	// 4. Try each candidate (failover)
	isStream := extractStream(body)
	for i, backend := range candidates {
		targetURL := backend.Endpoint + path
		proxyReq, reqErr := http.NewRequestWithContext(
			r.Context(), http.MethodPost, targetURL,
			strings.NewReader(string(body)),
		)
		if reqErr != nil {
			continue
		}
		copyHeaders(proxyReq.Header, r.Header)
		proxyReq.Header.Set("Content-Type", "application/json")

		resp, respErr := gw.client.Do(proxyReq)
		if respErr != nil {
			log.Printf("[gateway] backend %s failed: %v", backend.ID, respErr)
			continue
		}

		// 5xx → failover to next
		if resp.StatusCode >= 500 && i < len(candidates)-1 {
			resp.Body.Close()
			log.Printf("[gateway] backend %s returned %d, failover", backend.ID, resp.StatusCode)
			continue
		}

		// 5. Forward response (SSE streaming or regular JSON)
		if isStream && resp.StatusCode == http.StatusOK {
			gw.streamResponse(w, resp)
		} else {
			gw.forwardResponse(w, resp)
		}
		return
	}

	writeGatewayError(w, http.StatusBadGateway, "all backends failed")
}

// findBackends returns healthy backends supporting the model, weighted-random shuffled.
func (gw *Gateway) findBackends(modelName string) []model.Backend {
	healthy := gw.health.GetHealthyBackends()
	var matched []model.Backend
	for _, b := range healthy {
		if b.SupportsModel(modelName) {
			matched = append(matched, b)
		}
	}
	weightedShuffle(matched)
	return matched
}

// streamResponse forwards SSE chunks in real-time.
func (gw *Gateway) streamResponse(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if ok {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}
}

// forwardResponse forwards a regular JSON response.
func (gw *Gateway) forwardResponse(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// --- Helpers ---

func extractModel(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

func extractStream(body []byte) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &req)
	return req.Stream
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		lower := strings.ToLower(k)
		if lower == "host" || lower == "content-length" {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// weightedShuffle reorders backends by weight (higher weight = more likely first).
func weightedShuffle(backends []model.Backend) {
	if len(backends) <= 1 {
		return
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range backends {
		j := i + rng.Intn(len(backends)-i)
		// Bias toward higher weight
		if backends[j].Weight > backends[i].Weight {
			backends[i], backends[j] = backends[j], backends[i]
		}
	}
}

func writeGatewayJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeGatewayError(w http.ResponseWriter, status int, message string) {
	writeGatewayJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "gateway_error",
		},
	})
}
