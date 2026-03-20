package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"ai-job-orchestrator/internal/scheduler"
	"ai-job-orchestrator/internal/store"
	"ai-job-orchestrator/internal/telemetry"
)

func main() {
	_ = os.MkdirAll("data", 0o755)
	dsn := os.Getenv("JOB_DB_DSN")
	db, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	interval := 500 * time.Millisecond
	if v := os.Getenv("SCHEDULER_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			interval = time.Duration(ms) * time.Millisecond
		}
	}
	log.Printf("scheduler running every %s", interval)
	ctx := context.Background()
	for {
		j, err := scheduler.Tick(ctx, db)
		if err != nil {
			log.Printf("scheduler error: %v", err)
		} else if j != nil {
			log.Printf("scheduled job %s", j.ID)
			telemetry.JobTransitions.WithLabelValues("scheduled").Inc()
		}
		time.Sleep(interval)
	}
}
