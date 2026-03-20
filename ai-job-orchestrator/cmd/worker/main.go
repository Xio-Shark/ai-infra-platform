package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"ai-job-orchestrator/internal/model"
	"ai-job-orchestrator/internal/store"
	"ai-job-orchestrator/internal/telemetry"
	"ai-job-orchestrator/internal/worker"
)

func main() {
	_ = os.MkdirAll("data", 0o755)
	dsn := os.Getenv("JOB_DB_DSN")
	db, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mode := os.Getenv("WORKER_EXECUTOR")
	var exec worker.Executor = worker.ShellExecutor{}
	if mode == "http" {
		exec = worker.HTTPExecutor{}
	}
	if mode == "k8s" {
		exec = worker.K8sJobExecutor{}
	}

	interval := 500 * time.Millisecond
	ctx := context.Background()
	log.Printf("worker started executor=%s", mode)
	for {
		j, err := claimScheduled(ctx, db)
		if err != nil {
			log.Printf("worker claim error: %v", err)
			time.Sleep(interval)
			continue
		}
		if j == nil {
			time.Sleep(interval)
			continue
		}
		log.Printf("running job %s type=%s", j.ID, j.Type)
		runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err = exec.Run(runCtx, j)
		cancel()
		if err != nil {
			log.Printf("job %s failed: %v", j.ID, err)
			_ = store.UpdateJobStatus(ctx, db, j.ID, string(model.StatusFailed), err.Error())
			telemetry.JobTransitions.WithLabelValues("failed").Inc()
		} else {
			_ = store.UpdateJobStatus(ctx, db, j.ID, string(model.StatusSucceeded), "completed")
			telemetry.JobTransitions.WithLabelValues("succeeded").Inc()
		}
	}
}

func claimScheduled(ctx context.Context, db *sql.DB) (*model.Job, error) {
	j, err := store.ClaimNextScheduled(ctx, db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}
