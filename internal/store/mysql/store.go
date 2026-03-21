package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"ai-infra-platform/internal/model"
	"ai-infra-platform/internal/store"
)

type Store struct {
	db *sql.DB
}

type scanner func(dest ...any) error

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) CreateJob(ctx context.Context, job model.Job) error {
	row, err := encodeJob(job)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, createJobSQL, row...)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (s *Store) UpdateJob(ctx context.Context, job model.Job) error {
	row, err := encodeJob(job)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, updateJobSQL, append(row[1:], job.ID)...)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return ensureUpdated(result)
}

func (s *Store) GetJob(ctx context.Context, id string) (model.Job, error) {
	record, err := scanJob(s.db.QueryRowContext(ctx, getJobSQL, id).Scan)
	if err != nil {
		return model.Job{}, err
	}
	return decodeJob(record)
}

func (s *Store) ListJobs(ctx context.Context) ([]model.Job, error) {
	rows, err := s.db.QueryContext(ctx, listJobsSQL)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	items := make([]model.Job, 0)
	for rows.Next() {
		record, scanErr := scanJob(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		job, decodeErr := decodeJob(record)
		if decodeErr != nil {
			return nil, decodeErr
		}
		items = append(items, job)
	}
	return items, rows.Err()
}

func (s *Store) CreateExecution(ctx context.Context, execution model.Execution) error {
	row := encodeExecution(execution)
	_, err := s.db.ExecContext(ctx, createExecutionSQL, row...)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}
	return nil
}

func (s *Store) UpdateExecution(ctx context.Context, execution model.Execution) error {
	row := encodeExecution(execution)
	result, err := s.db.ExecContext(ctx, updateExecutionSQL, append(row[1:], execution.ID)...)
	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}
	return ensureUpdated(result)
}

func (s *Store) ListExecutions(ctx context.Context, jobID string) ([]model.Execution, error) {
	rows, err := s.db.QueryContext(ctx, listExecutionsSQL, jobID)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()
	items := make([]model.Execution, 0)
	for rows.Next() {
		execution, scanErr := scanExecution(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, execution)
	}
	return items, rows.Err()
}

type jobRecord struct {
	ID             string
	Name           string
	Type           string
	Status         string
	Executor       string
	Priority       int
	ModelVersion   string
	DatasetVersion string
	ImageTag       string
	ResourceSpec   []byte
	Command        []byte
	Environment    []byte
	Metadata       []byte
	MaxRetries     int
	RetryCount     int
	LastError      string
	TraceID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func encodeJob(job model.Job) ([]any, error) {
	spec, err := json.Marshal(job.ResourceSpec)
	if err != nil {
		return nil, fmt.Errorf("marshal resource spec: %w", err)
	}
	command, err := json.Marshal(job.Command)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}
	environment, err := json.Marshal(job.Environment)
	if err != nil {
		return nil, fmt.Errorf("marshal environment: %w", err)
	}
	metadata, err := json.Marshal(job.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return []any{job.ID, job.Name, string(job.Type), string(job.Status), job.Executor, job.Priority, job.ModelVersion, job.DatasetVersion, job.ImageTag, spec, command, environment, metadata, job.MaxRetries, job.RetryCount, job.LastError, job.TraceID, job.CreatedAt, job.UpdatedAt}, nil
}

func decodeJob(record jobRecord) (model.Job, error) {
	var spec model.ResourceSpec
	var command []string
	environment := map[string]string{}
	metadata := map[string]string{}
	if err := json.Unmarshal(record.ResourceSpec, &spec); err != nil {
		return model.Job{}, fmt.Errorf("decode resource spec: %w", err)
	}
	if err := json.Unmarshal(record.Command, &command); err != nil {
		return model.Job{}, fmt.Errorf("decode command: %w", err)
	}
	if len(record.Environment) > 0 {
		if err := json.Unmarshal(record.Environment, &environment); err != nil {
			return model.Job{}, fmt.Errorf("decode environment: %w", err)
		}
	}
	if len(record.Metadata) > 0 {
		if err := json.Unmarshal(record.Metadata, &metadata); err != nil {
			return model.Job{}, fmt.Errorf("decode metadata: %w", err)
		}
	}
	return model.Job{ID: record.ID, Name: record.Name, Type: model.JobType(record.Type), Status: model.JobStatus(record.Status), Executor: record.Executor, Priority: record.Priority, ModelVersion: record.ModelVersion, DatasetVersion: record.DatasetVersion, ImageTag: record.ImageTag, ResourceSpec: spec, Command: command, Environment: environment, Metadata: metadata, MaxRetries: record.MaxRetries, RetryCount: record.RetryCount, LastError: record.LastError, TraceID: record.TraceID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}, nil
}

func scanJob(scan scanner) (jobRecord, error) {
	var record jobRecord
	err := scan(&record.ID, &record.Name, &record.Type, &record.Status, &record.Executor, &record.Priority, &record.ModelVersion, &record.DatasetVersion, &record.ImageTag, &record.ResourceSpec, &record.Command, &record.Environment, &record.Metadata, &record.MaxRetries, &record.RetryCount, &record.LastError, &record.TraceID, &record.CreatedAt, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return jobRecord{}, store.ErrNotFound
	}
	if err != nil {
		return jobRecord{}, fmt.Errorf("scan job: %w", err)
	}
	return record, nil
}

func encodeExecution(execution model.Execution) []any {
	return []any{execution.ID, execution.JobID, execution.Attempt, string(execution.Status), execution.Executor, execution.TraceID, execution.Logs, execution.Manifest, execution.ExitCode, execution.Error, execution.StartedAt, sql.NullTime{Time: execution.FinishedAt, Valid: !execution.FinishedAt.IsZero()}}
}

func scanExecution(scan scanner) (model.Execution, error) {
	var execution model.Execution
	var status string
	var finishedAt sql.NullTime
	err := scan(&execution.ID, &execution.JobID, &execution.Attempt, &status, &execution.Executor, &execution.TraceID, &execution.Logs, &execution.Manifest, &execution.ExitCode, &execution.Error, &execution.StartedAt, &finishedAt)
	if err != nil {
		return model.Execution{}, fmt.Errorf("scan execution: %w", err)
	}
	execution.Status = model.ExecutionStatus(status)
	if finishedAt.Valid {
		execution.FinishedAt = finishedAt.Time
	}
	return execution, nil
}

func ensureUpdated(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return store.ErrNotFound
	}
	return nil
}

const createJobSQL = `INSERT INTO jobs (id, name, job_type, status, executor, priority, model_version, dataset_version, image_tag, resource_spec, command_json, environment_json, metadata_json, max_retries, retry_count, last_error, trace_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
const updateJobSQL = `UPDATE jobs SET name=?, job_type=?, status=?, executor=?, priority=?, model_version=?, dataset_version=?, image_tag=?, resource_spec=?, command_json=?, environment_json=?, metadata_json=?, max_retries=?, retry_count=?, last_error=?, trace_id=?, created_at=?, updated_at=? WHERE id=?`
const getJobSQL = `SELECT id, name, job_type, status, executor, priority, model_version, dataset_version, image_tag, resource_spec, command_json, environment_json, metadata_json, max_retries, retry_count, last_error, trace_id, created_at, updated_at FROM jobs WHERE id = ?`
const listJobsSQL = `SELECT id, name, job_type, status, executor, priority, model_version, dataset_version, image_tag, resource_spec, command_json, environment_json, metadata_json, max_retries, retry_count, last_error, trace_id, created_at, updated_at FROM jobs ORDER BY priority DESC, created_at ASC`
const createExecutionSQL = `INSERT INTO executions (id, job_id, attempt, status, executor, trace_id, logs, manifest, exit_code, error, started_at, finished_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
const updateExecutionSQL = `UPDATE executions SET job_id=?, attempt=?, status=?, executor=?, trace_id=?, logs=?, manifest=?, exit_code=?, error=?, started_at=?, finished_at=? WHERE id=?`
const listExecutionsSQL = `SELECT id, job_id, attempt, status, executor, trace_id, logs, manifest, exit_code, error, started_at, finished_at FROM executions WHERE job_id = ? ORDER BY attempt ASC`
