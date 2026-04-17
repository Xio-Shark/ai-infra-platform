package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ai-infra-platform/internal/gateway"
	"ai-infra-platform/internal/model"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("load gateway config: %v", err)
	}

	gw := gateway.New(cfg)
	gw.Start()
	defer gw.Stop()

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           gw.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("gateway graceful shutdown failed: %v", err)
		}
	}()

	if len(cfg.Backends) == 0 {
		log.Printf("gateway starting without configured backends; set GATEWAY_BACKENDS_JSON to enable routing")
	}

	log.Printf("gateway listening on %s with %d backend(s)", cfg.ListenAddr, len(cfg.Backends))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("gateway stopped: %v", err)
	}
}

func loadConfigFromEnv() (gateway.Config, error) {
	cfg := gateway.DefaultConfig()
	cfg.ListenAddr = envOrDefault("GATEWAY_LISTEN", cfg.ListenAddr)

	if raw := os.Getenv("GATEWAY_BACKENDS_JSON"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.Backends); err != nil {
			return gateway.Config{}, err
		}
	}

	if raw := os.Getenv("GATEWAY_RATE_LIMITS_JSON"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.RateLimits); err != nil {
			return gateway.Config{}, err
		}
	}

	cfg.DefaultRPS = intEnvOrDefault("GATEWAY_DEFAULT_RPS", cfg.DefaultRPS)
	cfg.RequestTimeout = durationEnvOrDefault("GATEWAY_REQUEST_TIMEOUT", cfg.RequestTimeout)
	cfg.HealthInterval = durationEnvOrDefault("GATEWAY_HEALTH_INTERVAL", cfg.HealthInterval)
	cfg.HealthTimeout = durationEnvOrDefault("GATEWAY_HEALTH_TIMEOUT", cfg.HealthTimeout)

	normalizeBackends(cfg.Backends)
	return cfg, nil
}

func normalizeBackends(backends []model.Backend) {
	for i := range backends {
		if backends[i].Weight <= 0 {
			backends[i].Weight = 1
		}
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func intEnvOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("parse %s as int: %v", key, err)
	}
	return parsed
}

func durationEnvOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("parse %s as duration: %v", key, err)
	}
	return parsed
}
