package task

import (
	"context"
	"encoding/json"
)

// OutboxRecord is the task-level representation of a pending outbox entry.
// Persistence details remain inside the repository implementation.
type OutboxRecord struct {
	ID      uint
	Payload json.RawMessage
}

// OutboxStore provides the persistence operations needed by the outbox
// dispatcher.
type OutboxStore interface {
	FetchPendingOutbox(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkOutboxPublished(ctx context.Context, outboxID uint, queueID int64) error
}

// TaskResultStore persists a completed task result and its completed status.
type TaskResultStore interface {
	UpdateTaskResult(ctx context.Context, taskID uint, result json.RawMessage) error
}

// TaskStore provides task persistence and transactional outbox creation.
type TaskStore interface {
	CreateWithOutbox(ctx context.Context, task *Task) (uint, error)
	GetTaskByID(ctx context.Context, taskID uint) (*Task, error)
	ListTasksByStatus(ctx context.Context, status Status) ([]*Task, error)
	UpdateTaskStatus(ctx context.Context, taskID uint, status Status) error
	TaskResultStore
	OutboxStore
}
