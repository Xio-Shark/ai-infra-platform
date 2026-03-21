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
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: scheduler <job-json-file>")
	}
	repo := store.NewMemoryStore()
	metrics := telemetry.NewMetrics()
	tracer := telemetry.NewTracer()
	jobService := service.NewJobService(repo, metrics, tracer)
	dispatcher := scheduler.NewDispatcher(repo, metrics, tracer)
	job := loadExample(os.Args[1])
	created, err := jobService.CreateJob(context.Background(), job)
	if err != nil {
		log.Fatalf("create job: %v", err)
	}
	scheduled, err := dispatcher.ScheduleJob(context.Background(), created.ID)
	if err != nil {
		log.Fatalf("schedule job: %v", err)
	}
	log.Printf("scheduled job %s with status %s", scheduled.ID, scheduled.Status)
}

func loadExample(path string) model.CreateJobInput {
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
