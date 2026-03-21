package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-infra-platform/internal/api"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/store"
	mysqlstore "ai-infra-platform/internal/store/mysql"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

const shutdownTimeout = 10 * time.Second

func main() {
	repo := buildRepository()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	registry := worker.NewRegistry(
		worker.ShellExecutor{},
		worker.K8sJobExecutor{Namespace: envOrDefault("K8S_NAMESPACE", "default")},
		worker.K8sApplyExecutor{
			Namespace:  envOrDefault("K8S_NAMESPACE", "default"),
			Kubeconfig: os.Getenv("KUBECONFIG"),
			AllowApply: os.Getenv("ALLOW_K8S_APPLY") == "true",
		},
		worker.HTTPExecutor{},
		worker.BenchmarkExecutor{},
	)
	jobService := service.NewJobService(repo, metrics, tracer)
	executionService := service.NewExecutionService(repo, dispatcher, registry, metrics, tracer)
	handler := api.NewRouter(jobService, executionService, dispatcher, metrics, tracer)

	addr := envOrDefault("API_ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 优雅关闭：监听 SIGINT/SIGTERM
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("received signal %s, shutting down...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		close(done)
	}()

	log.Printf("api-server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api-server stopped: %v", err)
	}
	<-done
	log.Println("api-server exited")
}

func buildRepository() store.Repository {
	switch envOrDefault("STORE_BACKEND", "memory") {
	case "memory":
		return store.NewMemoryStore()
	case "mysql":
		dsn := os.Getenv("MYSQL_DSN")
		if dsn == "" {
			log.Fatal("MYSQL_DSN is required when STORE_BACKEND=mysql")
		}
		repo, err := mysqlstore.Open(dsn)
		if err != nil {
			log.Fatalf("open mysql store: %v", err)
		}
		return repo
	default:
		log.Fatalf("unsupported STORE_BACKEND=%s", os.Getenv("STORE_BACKEND"))
		return nil
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
