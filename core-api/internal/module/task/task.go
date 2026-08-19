package task

import (
	"encoding/json"
	"time"
)

// Task is the queue-neutral task envelope. Business modules own Type and Payload semantics.
type Task struct {
	ID               uint            `json:"id"`
	Type             string          `json:"type"`
	Status           Status          `json:"status"`
	CompletionStatus Status          `json:"completionStatus,omitempty"`
	Payload          json.RawMessage `json:"payload"`
	Result           json.RawMessage `json:"result,omitempty"`
	Error            string          `json:"error,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// ListFilter selects tasks using queue-neutral task envelope fields.
type ListFilter struct {
	Statuses []Status
	Types    []string
	BeforeID uint
	Limit    int
}

type Status uint

const (
	StatusPending Status = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
	StatusCancelled
	StatusAwaitingApplication
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
	case StatusAwaitingApplication:
		return "awaiting_application"
	default:
		return "unknown"
	}
}
