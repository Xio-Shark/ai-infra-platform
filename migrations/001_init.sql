CREATE TABLE jobs (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  job_type VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  executor VARCHAR(64) NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  model_version VARCHAR(128) NOT NULL DEFAULT '',
  dataset_version VARCHAR(128) NOT NULL DEFAULT '',
  image_tag VARCHAR(255) NOT NULL DEFAULT '',
  resource_spec JSON NOT NULL,
  command_json JSON NOT NULL,
  environment_json JSON NOT NULL,
  metadata_json JSON NOT NULL,
  max_retries INT NOT NULL DEFAULT 0,
  retry_count INT NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL,
  trace_id VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE executions (
  id VARCHAR(64) PRIMARY KEY,
  job_id VARCHAR(64) NOT NULL,
  attempt INT NOT NULL,
  status VARCHAR(32) NOT NULL,
  executor VARCHAR(64) NOT NULL,
  trace_id VARCHAR(64) NOT NULL,
  logs TEXT NOT NULL,
  manifest TEXT NOT NULL,
  exit_code INT NOT NULL,
  error TEXT NOT NULL,
  started_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NULL,
  CONSTRAINT fk_job FOREIGN KEY (job_id) REFERENCES jobs(id)
);
