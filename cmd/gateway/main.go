package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ai-infra-platform/internal/gateway"
	"ai-infra-platform/internal/model"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := gateway.DefaultConfig()
	cfg.ListenAddr = envOrDefault("GATEWAY_LISTEN", cfg.ListenAddr)
	cfg.DefaultRPS = intEnvOrDefault("GATEWAY_DEFAULT_RPS", cfg.DefaultRPS)
	cfg.RequestTimeout = durationEnvOrDefault("GATEWAY_REQUEST_TIMEOUT", cfg.RequestTimeout)
	cfg.HealthInterval = durationEnvOrDefault("GATEWAY_HEALTH_INTERVAL", cfg.HealthInterval)
	cfg.HealthTimeout = durationEnvOrDefault("GATEWAY_HEALTH_TIMEOUT", cfg.HealthTimeout)
	cfg.Backends = mustLoadBackends()
	cfg.RateLimits = mustLoadRateLimits()

	gw := gateway.New(cfg)
	gw.Start()
	defer gw.Stop()

	if len(cfg.Backends) == 0 {
		log.Println("gateway started without GATEWAY_BACKENDS_JSON; /gateway/health will stay degraded until backends are configured")
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           gw.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("received signal %s, shutting down gateway...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("gateway graceful shutdown failed: %v", err)
		}
		close(done)
	}()

	log.Printf("gateway listening on %s with %d backend(s)", cfg.ListenAddr, len(cfg.Backends))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gateway stopped: %v", err)
	}
	<-done
	log.Println("gateway exited")
}

func mustLoadBackends() []model.Backend {
	backends, err := parseBackendsJSON(strings.TrimSpace(os.Getenv("GATEWAY_BACKENDS_JSON")))
	if err != nil {
		log.Fatalf("load gateway backends: %v", err)
	}
	return backends
}

func mustLoadRateLimits() map[string]int {
	limits, err := parseRateLimitsJSON(strings.TrimSpace(os.Getenv("GATEWAY_RATE_LIMITS_JSON")))
	if err != nil {
		log.Fatalf("load gateway rate limits: %v", err)
	}
	return limits
}

func parseBackendsJSON(raw string) ([]model.Backend, error) {
	if raw == "" {
		return nil, nil
	}

	var backends []model.Backend
	if err := json.Unmarshal([]byte(raw), &backends); err != nil {
		return nil, fmt.Errorf("invalid GATEWAY_BACKENDS_JSON: %w", err)
	}
	for i := range backends {
		if backends[i].ID == "" {
			return nil, fmt.Errorf("backend[%d] id is required", i)
		}
		if backends[i].Endpoint == "" {
			return nil, fmt.Errorf("backend[%d] endpoint is required", i)
		}
		if backends[i].Weight <= 0 {
			backends[i].Weight = 1
		}
	}
	return backends, nil
}

func parseRateLimitsJSON(raw string) (map[string]int, error) {
	if raw == "" {
		return nil, nil
	}

	var limits map[string]int
	if err := json.Unmarshal([]byte(raw), &limits); err != nil {
		return nil, fmt.Errorf("invalid GATEWAY_RATE_LIMITS_JSON: %w", err)
	}
	return limits, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func intEnvOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		log.Fatalf("%s must be an integer: %v", key, err)
	}
	return value
}

func durationEnvOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		log.Fatalf("%s must be a valid duration: %v", key, err)
	}
	return value
}
