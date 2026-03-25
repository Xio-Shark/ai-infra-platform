package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-infra-platform/internal/model"
)

// --- Helpers ---

func mockBackendServer(t *testing.T, response string, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(response))
	}))
}

func mockSSEServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
}

func makeGateway(t *testing.T, backends []model.Backend) *Gateway {
	t.Helper()
	cfg := Config{
		Backends:       backends,
		DefaultRPS:     1000,
		RequestTimeout: 5 * time.Second,
		HealthInterval: 1 * time.Hour, // don't auto-probe in tests
		HealthTimeout:  1 * time.Second,
	}
	gw := New(cfg)
	// Force initial health check
	gw.health.probeAll()
	return gw
}

// --- Tests ---

func TestGateway_RouteToCorrectBackend(t *testing.T) {
	srv := mockBackendServer(t, `{"id":"chatcmpl-1","choices":[]}`, http.StatusOK)
	defer srv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "b1", Endpoint: srv.URL, Models: []string{"qwen-7b"}, Weight: 1},
	})

	body := `{"model":"qwen-7b","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] != "chatcmpl-1" {
		t.Errorf("response id: want chatcmpl-1, got %v", resp["id"])
	}
}

func TestGateway_MissingModel(t *testing.T) {
	gw := makeGateway(t, nil)

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", w.Code)
	}
}

func TestGateway_NoHealthyBackend(t *testing.T) {
	// Server that fails health check
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failSrv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "sick", Endpoint: failSrv.URL, Models: []string{"qwen-7b"}, Weight: 1},
	})

	body := `{"model":"qwen-7b","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: want 503, got %d", w.Code)
	}
}

func TestGateway_Failover(t *testing.T) {
	// First backend returns 500
	badSrv := mockBackendServer(t, `{"error":"internal"}`, http.StatusInternalServerError)
	defer badSrv.Close()

	// Second backend returns 200
	goodSrv := mockBackendServer(t, `{"id":"chatcmpl-ok","choices":[]}`, http.StatusOK)
	defer goodSrv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "bad", Endpoint: badSrv.URL, Models: []string{"qwen-7b"}, Weight: 100},
		{ID: "good", Endpoint: goodSrv.URL, Models: []string{"qwen-7b"}, Weight: 1},
	})

	body := `{"model":"qwen-7b","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	gw.Handler().ServeHTTP(w, req)

	// Should failover to good backend
	if w.Code != http.StatusOK {
		t.Errorf("status: want 200 (failover), got %d", w.Code)
	}
}

func TestGateway_SSEStreaming(t *testing.T) {
	chunks := []string{
		"data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n",
		"data: {\"id\":\"2\",\"choices\":[{\"delta\":{\"content\":\" World\"}}]}\n\n",
		"data: [DONE]\n\n",
	}
	srv := mockSSEServer(t, chunks)
	defer srv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "sse", Endpoint: srv.URL, Models: []string{"gpt-4"}, Weight: 1},
	})

	body := `{"model":"gpt-4","messages":[],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "Hello") || !strings.Contains(respBody, "[DONE]") {
		t.Errorf("SSE chunks not forwarded correctly: %s", respBody)
	}
}

func TestGateway_RateLimit(t *testing.T) {
	srv := mockBackendServer(t, `{"id":"ok"}`, http.StatusOK)
	defer srv.Close()

	cfg := Config{
		Backends:       []model.Backend{
			{ID: "b1", Endpoint: srv.URL, Models: []string{"qwen-7b"}, Weight: 1},
		},
		RateLimits:     map[string]int{"qwen-7b": 1}, // 1 RPS
		DefaultRPS:     1000,
		RequestTimeout: 5 * time.Second,
		HealthInterval: 1 * time.Hour,
		HealthTimeout:  1 * time.Second,
	}
	gw := New(cfg)
	gw.health.probeAll()

	body := `{"model":"qwen-7b","messages":[]}`

	// First request should pass
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w1 := httptest.NewRecorder()
	gw.Handler().ServeHTTP(w1, req1)

	// Rapid second request should be rate-limited
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w2 := httptest.NewRecorder()
	gw.Handler().ServeHTTP(w2, req2)

	// Third request — definitely limited
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	w3 := httptest.NewRecorder()
	gw.Handler().ServeHTTP(w3, req3)

	// At least one of w2/w3 should be 429
	if w2.Code != http.StatusTooManyRequests && w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected at least one 429, got w2=%d w3=%d", w2.Code, w3.Code)
	}
}

func TestGateway_HealthEndpoint(t *testing.T) {
	srv := mockBackendServer(t, `{}`, http.StatusOK)
	defer srv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "b1", Endpoint: srv.URL, Models: []string{"m1"}, Weight: 1},
	})

	req := httptest.NewRequest(http.MethodGet, "/gateway/health", nil)
	w := httptest.NewRecorder()
	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health status: want 200, got %d", w.Code)
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status: want ok, got %v", resp["status"])
	}
}

func TestGateway_BackendsEndpoint(t *testing.T) {
	srv := mockBackendServer(t, `{}`, http.StatusOK)
	defer srv.Close()

	gw := makeGateway(t, []model.Backend{
		{ID: "b1", Endpoint: srv.URL, Models: []string{"m1"}, Weight: 1},
	})

	req := httptest.NewRequest(http.MethodGet, "/gateway/backends", nil)
	w := httptest.NewRecorder()
	gw.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("backends status: want 200, got %d", w.Code)
	}

	bodyBytes, _ := io.ReadAll(w.Body)
	var backends []model.Backend
	if err := json.Unmarshal(bodyBytes, &backends); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(backends) != 1 || backends[0].ID != "b1" {
		t.Errorf("backends: want [b1], got %+v", backends)
	}
}

// --- Limiter unit tests ---

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(2) // 2 RPS, capacity=4

	// Should allow initial burst
	if !tb.Allow() {
		t.Error("first request should be allowed")
	}
	if !tb.Allow() {
		t.Error("second request should be allowed (within capacity)")
	}
}

func TestTokenBucket_Exhaustion(t *testing.T) {
	tb := NewTokenBucket(1) // 1 RPS, capacity=2

	// Drain all tokens
	tb.Allow()
	tb.Allow()
	if tb.Allow() {
		t.Error("third request should be denied (exhausted)")
	}
}

func TestRateLimiterRegistry_PerModel(t *testing.T) {
	reg := NewRateLimiterRegistry(1000, map[string]int{"slow-model": 1})

	// "fast-model" uses default 1000 RPS — should pass easily
	if !reg.Allow("fast-model") {
		t.Error("fast-model should be allowed")
	}

	// "slow-model" limited to 1 RPS
	reg.Allow("slow-model")
	reg.Allow("slow-model")
	if reg.Allow("slow-model") {
		t.Error("slow-model should be rate-limited after burst")
	}
}

func TestExtractModel(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{`{"model":"gpt-4","messages":[]}`, "gpt-4"},
		{`{"messages":[]}`, ""},
		{`invalid json`, ""},
	}
	for _, tc := range tests {
		got := extractModel([]byte(tc.body))
		if got != tc.want {
			t.Errorf("extractModel(%q) = %q, want %q", tc.body, got, tc.want)
		}
	}
}
