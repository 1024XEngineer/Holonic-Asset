package task

import (
	"encoding/json"
	"time"
)

// Task is the queue-neutral task envelope. Business modules own Type and Payload semantics.
type Task struct {
	ID        uint            `json:"id"`
	Type      string          `json:"type"`
	Status    Status          `json:"status"`
	Payload   json.RawMessage `json:"payload"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type Status uint

const (
	StatusPending Status = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
	StatusCancelled
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusProcessing:
		return "processing"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}
