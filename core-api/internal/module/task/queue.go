package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	riverqueue "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

const queueTaskKind = "holonic_task"

type queueTaskArgs struct {
	Task Task `json:"task"`
}

func (queueTaskArgs) Kind() string { return queueTaskKind }

type queueWorker struct {
	riverqueue.WorkerDefaults[queueTaskArgs]
	queue *TaskQueue
}

func (w *queueWorker) Work(ctx context.Context, job *riverqueue.Job[queueTaskArgs]) error {
	return w.queue.dispatch(ctx, &job.Args.Task)
}

// Queue combines task registration, production, and consumption.
type Queue interface {
	Register(taskType string, h Handler)
	Producer
	Consumer
}

// TaskQueue is the ready-to-use task queue implementation.
type TaskQueue struct {
	client   *riverqueue.Client[pgx.Tx]
	dbPool   *pgxpool.Pool
	registry *registry
	repo     TaskResultStore
}

// NewQueue creates a ready-to-use task queue using the module's configuration.
func NewQueue(ctx context.Context, cfg config.QueueConfig, repo TaskResultStore) (Queue, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("task: database URL is required")
	}
	if cfg.MaxWorkers < 1 {
		return nil, fmt.Errorf("task: max workers must be at least 1")
	}
	if repo == nil {
		return nil, fmt.Errorf("task: task result store is required")
	}

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("task: create database pool: %w", err)
	}

	queue := &TaskQueue{dbPool: dbPool, registry: newRegistry(), repo: repo}
	workers := riverqueue.NewWorkers()
	riverqueue.AddWorker(workers, &queueWorker{queue: queue})

	riverConfig := &riverqueue.Config{
		Queues: map[string]riverqueue.QueueConfig{
			riverqueue.QueueDefault: {
				MaxWorkers: cfg.MaxWorkers,
			},
		},
		Workers: workers,
	}
	if cfg.JobTimeout > 0 {
		riverConfig.JobTimeout = cfg.JobTimeout
	}

	client, err := riverqueue.NewClient(riverpgxv5.New(dbPool), riverConfig)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("task: create queue: %w", err)
	}
	queue.client = client
	return queue, nil
}

// Producer publishes tasks to the configured queue.
type Producer interface {
	Publish(ctx context.Context, task *Task) error
}

// Consumer receives tasks and delegates processing to registered handlers.
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

func (q *TaskQueue) Register(taskType string, h Handler) {
	q.registry.register(taskType, h)
}

func (q *TaskQueue) Publish(ctx context.Context, message *Task) error {
	if message == nil {
		return fmt.Errorf("task: cannot publish nil task")
	}

	result, err := q.client.Insert(ctx, queueTaskArgs{Task: *message}, &riverqueue.InsertOpts{
		UniqueOpts: riverqueue.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return fmt.Errorf("task: publish task %q: %w", message.Type, err)
	}
	if result.UniqueSkippedAsDuplicate {
		return nil
	}
	return nil
}

func (q *TaskQueue) Start(ctx context.Context) error {
	if err := q.client.Start(ctx); err != nil {
		return fmt.Errorf("task: start queue: %w", err)
	}
	return nil
}

func (q *TaskQueue) Stop() error {
	err := q.client.Stop(context.Background())
	q.dbPool.Close()
	return err
}

func (q *TaskQueue) dispatch(ctx context.Context, message *Task) error {
	if message == nil {
		return fmt.Errorf("task: cannot dispatch nil task")
	}

	data, err := q.registry.dispatch(ctx, message)
	if err != nil {
		return err
	}

	result, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("task: encode result for task %d: %w", message.ID, err)
	}
	if err := q.repo.UpdateTaskResult(ctx, message.ID, result); err != nil {
		return fmt.Errorf("task: persist result for task %d: %w", message.ID, err)
	}

	message.Result = result
	message.Status = StatusCompleted
	return nil
}

var (
	_ Queue = (*TaskQueue)(nil)
)
