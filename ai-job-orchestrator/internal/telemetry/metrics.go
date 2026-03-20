package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_jobs_created_total",
			Help: "Jobs created by type",
		},
		[]string{"job_type"},
	)
	JobTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_job_status_transitions_total",
			Help: "Job status transitions",
		},
		[]string{"to_status"},
	)
)
