package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ai-job-orchestrator/internal/model"
	"ai-job-orchestrator/internal/service"
)

type Jobs struct {
	Svc *service.JobService
}

func (h *Jobs) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		service.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	j, err := h.Svc.Create(r.Context(), req)
	if err != nil {
		service.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	service.JSON(w, http.StatusCreated, j)
}

func (h *Jobs) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	j, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			service.JSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		service.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	service.JSON(w, http.StatusOK, j)
}

func (h *Jobs) Executions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := h.Svc.ListExecutions(r.Context(), id)
	if err != nil {
		service.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	service.JSON(w, http.StatusOK, rows)
}
