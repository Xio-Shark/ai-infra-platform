package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ai-job-orchestrator/internal/model"
)

// HTTPExecutor calls Payload URL with GET (demo only).
type HTTPExecutor struct {
	Client *http.Client
}

func (h HTTPExecutor) Run(ctx context.Context, job *model.Job) error {
	if h.Client == nil {
		h.Client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.Payload, nil)
	if err != nil {
		return err
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return nil
}
