package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/store"
	"ai-infra-platform/internal/telemetry"
	"ai-infra-platform/internal/worker"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: worker <job-json-file>")
	}
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	registry := worker.NewRegistry(worker.ShellExecutor{}, worker.K8sJobExecutor{Namespace: "default"})
	jobService := service.NewJobService(repo, metrics, tracer)
	executionService := service.NewExecutionService(repo, dispatcher, registry, metrics, tracer)
	input := loadWorkerExample(os.Args[1])
	job, err := jobService.CreateJob(context.Background(), input)
	if err != nil {
		log.Fatalf("create job: %v", err)
	}
	execution, err := executionService.RunJob(context.Background(), job.ID)
	if err != nil {
		log.Fatalf("run job: %v", err)
	}
	log.Printf("execution %s finished with status %s", execution.ID, execution.Status)
}

func loadWorkerExample(path string) model.CreateJobInput {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	var input model.CreateJobInput
	if err := json.Unmarshal(data, &input); err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}
	return input
}
