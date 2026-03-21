package telemetry

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	mu              sync.Mutex
	submitted       int
	scheduled       int
	running         int
	succeeded       int
	failed          int
	retried         int
	cancelled       int
	executionCount  int
	totalRuntimeSec float64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncSubmitted() { m.add(func() { m.submitted++ }) }
func (m *Metrics) IncScheduled() { m.add(func() { m.scheduled++ }) }
func (m *Metrics) IncRunning()   { m.add(func() { m.running++ }) }
func (m *Metrics) IncSucceeded() { m.add(func() { m.succeeded++ }) }
func (m *Metrics) IncFailed()    { m.add(func() { m.failed++ }) }
func (m *Metrics) IncRetried()   { m.add(func() { m.retried++ }) }
func (m *Metrics) IncCancelled() { m.add(func() { m.cancelled++ }) }

func (m *Metrics) ObserveRuntime(duration time.Duration) {
	m.add(func() {
		m.executionCount++
		m.totalRuntimeSec += duration.Seconds()
	})
}

func (m *Metrics) RenderPrometheus() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := []string{
		"# HELP ai_jobs_submitted_total Total submitted jobs.",
		"# TYPE ai_jobs_submitted_total counter",
		fmt.Sprintf("ai_jobs_submitted_total %d", m.submitted),
		"# HELP ai_jobs_scheduled_total Total scheduled jobs.",
		"# TYPE ai_jobs_scheduled_total counter",
		fmt.Sprintf("ai_jobs_scheduled_total %d", m.scheduled),
		"# HELP ai_jobs_running_total Total running jobs observed.",
		"# TYPE ai_jobs_running_total counter",
		fmt.Sprintf("ai_jobs_running_total %d", m.running),
		"# HELP ai_jobs_succeeded_total Total succeeded jobs.",
		"# TYPE ai_jobs_succeeded_total counter",
		fmt.Sprintf("ai_jobs_succeeded_total %d", m.succeeded),
		"# HELP ai_jobs_failed_total Total failed jobs.",
		"# TYPE ai_jobs_failed_total counter",
		fmt.Sprintf("ai_jobs_failed_total %d", m.failed),
		"# HELP ai_jobs_retried_total Total retries requested.",
		"# TYPE ai_jobs_retried_total counter",
		fmt.Sprintf("ai_jobs_retried_total %d", m.retried),
		"# HELP ai_jobs_cancelled_total Total cancelled jobs.",
		"# TYPE ai_jobs_cancelled_total counter",
		fmt.Sprintf("ai_jobs_cancelled_total %d", m.cancelled),
		"# HELP ai_job_runtime_seconds_sum Sum of job runtime seconds.",
		"# TYPE ai_job_runtime_seconds_sum counter",
		fmt.Sprintf("ai_job_runtime_seconds_sum %.6f", m.totalRuntimeSec),
		"# HELP ai_job_runtime_seconds_count Count of observed executions.",
		"# TYPE ai_job_runtime_seconds_count counter",
		fmt.Sprintf("ai_job_runtime_seconds_count %d", m.executionCount),
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *Metrics) add(update func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	update()
}
