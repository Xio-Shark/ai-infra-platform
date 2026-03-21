CREATE INDEX idx_jobs_status_priority_created_at
  ON jobs (status, priority, created_at);

CREATE INDEX idx_jobs_trace_id
  ON jobs (trace_id);

CREATE INDEX idx_executions_job_attempt
  ON executions (job_id, attempt);
