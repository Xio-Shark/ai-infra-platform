package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ai-job-orchestrator/internal/api/handlers"
	"ai-job-orchestrator/internal/api/middleware"
	"ai-job-orchestrator/internal/service"
)

func NewRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestLog)

	jh := &handlers.Jobs{Svc: &service.JobService{DB: db}}
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/jobs", jh.Create)
		r.Get("/jobs/{id}", jh.Get)
		r.Get("/jobs/{id}/executions", jh.Executions)
	})
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())
	return r
}
