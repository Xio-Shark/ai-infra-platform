package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-infra-platform/internal/api"
	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

func TestRouterCreateAndRunJob(t *testing.T) {
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	registry := worker.NewRegistry(worker.K8sJobExecutor{Namespace: "test"})
	jobService := service.NewJobService(repo, metrics, tracer)
	executionService := service.NewExecutionService(repo, dispatcher, registry, metrics, tracer)
	handler := api.NewRouter(jobService, executionService, dispatcher, metrics, tracer)

	body, _ := json.Marshal(model.CreateJobInput{
		Name:       "dry-run-job",
		Type:       model.JobTypeInference,
		Executor:   "k8s-dry-run",
		Command:    []string{"python", "serve.py"},
		ImageTag:   "demo:latest",
		MaxRetries: 0,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}
	var job model.Job
	if err := json.Unmarshal(createRec.Body.Bytes(), &job); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/jobs/"+job.ID+"/run", nil)
	runRec := httptest.NewRecorder()
	handler.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run status = %d, want %d body=%s", runRec.Code, http.StatusOK, runRec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", metricsRec.Code, http.StatusOK)
	}
	if got := metricsRec.Body.String(); got == "" {
		t.Fatal("metrics body is empty")
	}
}
