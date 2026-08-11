package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	riverqueue "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

const queueTaskKind = "holonic_task"

type queueTaskArgs struct {
	Task Task `json:"task"`
}

func (queueTaskArgs) Kind() string { return queueTaskKind }

type queueWorker struct {
	riverqueue.WorkerDefaults[queueTaskArgs]
	queue *queue
}

func (w *queueWorker) Work(ctx context.Context, job *riverqueue.Job[queueTaskArgs]) error {
	return w.queue.dispatch(ctx, &job.Args.Task)
}

type queue struct {
	client   *riverqueue.Client[pgx.Tx]
	dbPool   *pgxpool.Pool
	registry *registry
	repo     TaskExecutionStore
}

func newQueue(ctx context.Context, cfg config.QueueConfig, repo TaskExecutionStore) (*queue, error) {
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

	queue := &queue{dbPool: dbPool, registry: newRegistry(), repo: repo}
	workers := riverqueue.NewWorkers()
	riverqueue.AddWorker(workers, &queueWorker{queue: queue})

	riverConfig := &riverqueue.Config{
		Queues: map[string]riverqueue.QueueConfig{
			riverqueue.QueueDefault: {
				MaxWorkers: cfg.MaxWorkers,
			},
		},
		Workers:      workers,
		ErrorHandler: &queueErrorHandler{repo: repo},
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

func (q *queue) Register(taskType string, h Handler) {
	q.registry.register(taskType, h)
}

func (q *queue) publish(ctx context.Context, message *Task) error {
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

func (q *queue) start(ctx context.Context) error {
	if err := q.client.Start(ctx); err != nil {
		return fmt.Errorf("task: start queue: %w", err)
	}
	return nil
}

func (q *queue) stop() error {
	err := q.client.Stop(context.Background())
	q.dbPool.Close()
	return err
}

func (q *queue) dispatch(ctx context.Context, message *Task) error {
	if message == nil {
		return fmt.Errorf("task: cannot dispatch nil task")
	}
	if err := q.repo.UpdateTaskStatus(ctx, message.ID, StatusProcessing); err != nil {
		return fmt.Errorf("task: mark task %d as processing: %w", message.ID, err)
	}
	message.Status = StatusProcessing

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

// queueErrorHandler records handler failures immediately. Generator handlers
// can return provider, validation, and local-tool errors that are not safe to
// blindly retry (for example, a paid video request or missing ffmpeg).
type queueErrorHandler struct {
	repo TaskExecutionStore
}

func (h *queueErrorHandler) HandleError(
	ctx context.Context,
	job *rivertype.JobRow,
	jobErr error,
) *riverqueue.ErrorHandlerResult {
	h.markFailed(ctx, job, jobErr, true)
	return &riverqueue.ErrorHandlerResult{SetCancelled: true}
}

func (h *queueErrorHandler) HandlePanic(
	ctx context.Context,
	job *rivertype.JobRow,
	panicValue any,
	_ string,
) *riverqueue.ErrorHandlerResult {
	h.markFailed(ctx, job, fmt.Errorf("task panicked: %v", panicValue), true)
	return &riverqueue.ErrorHandlerResult{SetCancelled: true}
}

func (h *queueErrorHandler) markFailed(
	ctx context.Context,
	job *rivertype.JobRow,
	failure error,
	force bool,
) {
	if job == nil || job.Kind != queueTaskKind || (!force && job.Attempt < job.MaxAttempts) {
		return
	}

	var args queueTaskArgs
	if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
		log.Printf("task: decode failed queue job %d: %v", job.ID, err)
		return
	}
	if args.Task.ID == 0 {
		log.Printf("task: failed queue job %d has no task ID", job.ID)
		return
	}
	errorMessage := "task execution failed"
	if failure != nil {
		errorMessage = failure.Error()
	}
	// The worker context has already expired on a job timeout. Detach the
	// persistence write from it so the failed transition can still reach the DB.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := h.repo.UpdateTaskFailure(persistCtx, args.Task.ID, errorMessage); err != nil {
		log.Printf("task: mark task %d as failed: %v", args.Task.ID, err)
	}
}
