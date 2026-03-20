package model

import "time"

type Execution struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Phase     string    `json:"phase"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
