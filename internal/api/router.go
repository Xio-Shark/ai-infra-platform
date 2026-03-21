package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/scheduler"
	"ai-infra-platform/internal/service"
	"ai-infra-platform/internal/telemetry"
)

type Router struct {
	jobService       *service.JobService
	executionService *service.ExecutionService
	dispatcher       *scheduler.Dispatcher
	metrics          *telemetry.Metrics
	tracer           *telemetry.Tracer
}

func NewRouter(
	jobService *service.JobService,
	executionService *service.ExecutionService,
	dispatcher *scheduler.Dispatcher,
	metrics *telemetry.Metrics,
	tracer *telemetry.Tracer,
) http.Handler {
	router := &Router{
		jobService:       jobService,
		executionService: executionService,
		dispatcher:       dispatcher,
		metrics:          metrics,
		tracer:           tracer,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", router.handleHealth)
	mux.HandleFunc("/metrics", router.handleMetrics)
	mux.HandleFunc("/jobs", router.handleJobs)
	mux.HandleFunc("/jobs/", router.handleJobByID)
	mux.HandleFunc("/dispatch/once", router.handleDispatchOnce)
	return mux
}

func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(r.metrics.RenderPrometheus()))
}

func (r *Router) handleJobs(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		var input model.CreateJobInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		job, err := r.jobService.CreateJob(req.Context(), input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, job)
	case http.MethodGet:
		jobs, err := r.jobService.ListJobs(req.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (r *Router) handleJobByID(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/jobs/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "job id is required")
		return
	}
	jobID := parts[0]
	if len(parts) == 1 {
		if req.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		job, err := r.jobService.GetJob(req.Context(), jobID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	switch parts[1] {
	case "schedule":
		r.handleScheduleJob(w, req, jobID)
	case "run":
		r.handleRunJob(w, req, jobID)
	case "retry":
		r.handleRetryJob(w, req, jobID)
	case "cancel":
		r.handleCancelJob(w, req, jobID)
	case "executions":
		r.handleListExecutions(w, req, jobID)
	case "trace":
		r.handleTrace(w, req, jobID)
	default:
		writeError(w, http.StatusNotFound, "route not found")
	}
}

func (r *Router) handleScheduleJob(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := r.dispatcher.ScheduleJob(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (r *Router) handleRunJob(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	execution, err := r.executionService.RunJob(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, execution)
}

func (r *Router) handleRetryJob(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := r.jobService.RetryJob(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (r *Router) handleCancelJob(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := r.jobService.CancelJob(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (r *Router) handleListExecutions(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	items, err := r.executionService.ListExecutions(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r *Router) handleTrace(w http.ResponseWriter, req *http.Request, jobID string) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := r.jobService.GetJob(req.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, r.tracer.Snapshot(job.TraceID))
}

func (r *Router) handleDispatchOnce(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 1
	if raw := req.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	executions, err := r.executionService.DispatchPending(req.Context(), limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, executions)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
