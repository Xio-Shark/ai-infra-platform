package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"ai-job-orchestrator/internal/model"
)

func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		dsn = "file:./data/jobs.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrationsDir() string {
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d
	}
	return "migrations"
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var version string
		err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version = ?`, name).Scan(&version)
		if err == sql.ErrNoRows {
			body, rerr := os.ReadFile(filepath.Join(dir, name))
			if rerr != nil {
				return rerr
			}
			if _, err := db.Exec(string(body)); err != nil {
				return fmt.Errorf("migrate %s: %w", name, err)
			}
			if _, err := db.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, name); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func CreateJob(ctx context.Context, db *sql.DB, j *model.Job) error {
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, `
INSERT INTO jobs(id,status,job_type,payload,model_version,dataset_version,image_tag,resource_spec,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		j.ID, j.Status, string(j.Type), j.Payload, j.ModelVersion, j.DatasetVersion, j.ImageTag, j.ResourceSpec, now, now)
	return err
}

func GetJob(ctx context.Context, db *sql.DB, id string) (*model.Job, error) {
	row := db.QueryRowContext(ctx, `
SELECT id,status,job_type,IFNULL(payload,''),IFNULL(model_version,''),IFNULL(dataset_version,''),IFNULL(image_tag,''),IFNULL(resource_spec,''),created_at,updated_at
FROM jobs WHERE id=?`, id)
	var j model.Job
	var created, updated int64
	if err := row.Scan(&j.ID, &j.Status, &j.Type, &j.Payload, &j.ModelVersion, &j.DatasetVersion, &j.ImageTag, &j.ResourceSpec, &created, &updated); err != nil {
		return nil, err
	}
	j.CreatedAt = time.Unix(created, 0).UTC()
	j.UpdatedAt = time.Unix(updated, 0).UTC()
	return &j, nil
}

func ClaimNextPending(ctx context.Context, db *sql.DB) (*model.Job, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE status=? ORDER BY created_at LIMIT 1`, string(model.StatusPending))
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, updated_at=? WHERE id=?`, string(model.StatusScheduled), now, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetJob(ctx, db, id)
}

func ClaimNextScheduled(ctx context.Context, db *sql.DB) (*model.Job, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE status=? ORDER BY updated_at LIMIT 1`, string(model.StatusScheduled))
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status=?, updated_at=? WHERE id=?`, string(model.StatusRunning), now, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetJob(ctx, db, id)
}

func UpdateJobStatus(ctx context.Context, db *sql.DB, id, status, message string) error {
	now := time.Now().Unix()
	if _, err := db.ExecContext(ctx, `UPDATE jobs SET status=?, updated_at=? WHERE id=?`, status, now, id); err != nil {
		return err
	}
	if message != "" {
		_, _ = db.ExecContext(ctx, `INSERT INTO executions(id,job_id,phase,message,created_at) VALUES(?,?,?,?,?)`,
			fmt.Sprintf("%s-%d", id, now), id, status, message, now)
	}
	return nil
}

func ListExecutions(ctx context.Context, db *sql.DB, jobID string) ([]model.Execution, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,job_id,phase,IFNULL(message,''),created_at FROM executions WHERE job_id=? ORDER BY created_at`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Execution
	for rows.Next() {
		var e model.Execution
		var ts int64
		if err := rows.Scan(&e.ID, &e.JobID, &e.Phase, &e.Message, &ts); err != nil {
			return nil, err
		}
		e.CreatedAt = time.Unix(ts, 0).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
