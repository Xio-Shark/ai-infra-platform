package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"ai-job-orchestrator/internal/model"
	"ai-job-orchestrator/internal/store"
	"ai-job-orchestrator/internal/telemetry"
)

var ErrNotFound = errors.New("not found")

type JobService struct {
	DB *sql.DB
}

func (s *JobService) Create(ctx context.Context, req model.CreateJobRequest) (*model.Job, error) {
	switch req.JobType {
	case model.JobTraining, model.JobInference, model.JobEval, model.JobBenchmark:
	default:
		return nil, fmt.Errorf("invalid job_type")
	}
	j := &model.Job{
		ID:             uuid.NewString(),
		Status:         string(model.StatusPending),
		Type:           req.JobType,
		Payload:        req.Payload,
		ModelVersion:   req.ModelVersion,
		DatasetVersion: req.DatasetVersion,
		ImageTag:       req.ImageTag,
		ResourceSpec:   req.ResourceSpec,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := store.CreateJob(ctx, s.DB, j); err != nil {
		return nil, err
	}
	telemetry.JobsCreated.WithLabelValues(string(req.JobType)).Inc()
	return j, nil
}

func (s *JobService) Get(ctx context.Context, id string) (*model.Job, error) {
	return store.GetJob(ctx, s.DB, id)
}

func (s *JobService) ListExecutions(ctx context.Context, id string) ([]model.Execution, error) {
	return store.ListExecutions(ctx, s.DB, id)
}

func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
