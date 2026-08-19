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

// TaskExecutionStore persists status transitions owned by queue execution.
type TaskExecutionStore interface {
	UpdateTaskStatus(ctx context.Context, taskID uint, status Status) error
	UpdateTaskResult(ctx context.Context, taskID uint, status Status, result json.RawMessage) error
	UpdateTaskFailure(ctx context.Context, taskID uint, errorMessage string) error
	CompleteTask(ctx context.Context, taskID uint) error
}

// TaskStore provides task persistence and transactional outbox creation.
type TaskStore interface {
	CreateWithOutbox(ctx context.Context, task *Task) (uint, error)
	GetTaskByID(ctx context.Context, taskID uint) (*Task, error)
	ListTasks(ctx context.Context, filter *ListFilter) ([]*Task, error)
	TaskExecutionStore
	OutboxStore
}
