package main

import (
	"log"
	"net/http"
	"os"

	"ai-infra-platform/internal/api"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/store"
	mysqlstore "ai-infra-platform/internal/store/mysql"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

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
	log.Printf("api-server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("api-server stopped: %v", err)
	}
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
