// Package river provides a River implementation for the generic task module.
package river

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	riverqueue "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

const riverTaskKind = "holonic_task"

type taskArgs struct {
	Task task.Task `json:"task"`
}

func (taskArgs) Kind() string { return riverTaskKind }

type worker struct {
	riverqueue.WorkerDefaults[taskArgs]
	queue *Queue
}

func (w *worker) Work(ctx context.Context, job *riverqueue.Job[taskArgs]) error {
	return w.queue.dispatch(ctx, &job.Args.Task)
}

// Queue implements the task producer and consumer contracts with River.
type Queue struct {
	client *riverqueue.Client[pgx.Tx]

	processMu sync.RWMutex
	process   task.ProcessFunc
}

// New creates a River-backed task queue. Start must be called before consuming.
func New(dbPool *pgxpool.Pool, config *riverqueue.Config) (*Queue, error) {
	if config == nil {
		return nil, fmt.Errorf("river: config is required")
	}

	queue := &Queue{}
	workers := riverqueue.NewWorkers()
	riverqueue.AddWorker(workers, &worker{queue: queue})

	clientConfig := *config
	clientConfig.Workers = workers

	client, err := riverqueue.NewClient(riverpgxv5.New(dbPool), &clientConfig)
	if err != nil {
		return nil, fmt.Errorf("river: create client: %w", err)
	}
	queue.client = client
	return queue, nil
}

func (q *Queue) Publish(ctx context.Context, message *task.Task) error {
	return publish(ctx, q.client, message)
}

func (q *Queue) Start(ctx context.Context, process task.ProcessFunc) error {
	if process == nil {
		return fmt.Errorf("river: process function is required")
	}

	q.processMu.Lock()
	q.process = process
	q.processMu.Unlock()

	if err := q.client.Start(ctx); err != nil {
		return fmt.Errorf("river: start consumer: %w", err)
	}
	return nil
}

func (q *Queue) Stop() error {
	return q.client.Stop(context.Background())
}

func (q *Queue) dispatch(ctx context.Context, message *task.Task) error {
	q.processMu.RLock()
	process := q.process
	q.processMu.RUnlock()
	if process == nil {
		return fmt.Errorf("river: consumer has not been started")
	}
	return process(ctx, message)
}

var (
	_ task.Producer = (*Queue)(nil)
	_ task.Consumer = (*Queue)(nil)
)

// Producer is a publish-only River adapter kept for composition compatibility.
type Producer struct {
	client *riverqueue.Client[pgx.Tx]
}

func NewProducer(client *riverqueue.Client[pgx.Tx]) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Publish(ctx context.Context, message *task.Task) error {
	return publish(ctx, p.client, message)
}

var _ task.Producer = (*Producer)(nil)

func publish(
	ctx context.Context,
	client *riverqueue.Client[pgx.Tx],
	message *task.Task,
) error {
	if message == nil {
		return fmt.Errorf("river: cannot publish nil task")
	}

	result, err := client.Insert(ctx, taskArgs{Task: *message}, &riverqueue.InsertOpts{
		UniqueOpts: riverqueue.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return fmt.Errorf("river: publish task %q: %w", message.Type, err)
	}
	if result.UniqueSkippedAsDuplicate {
		return nil
	}
	return nil
}

// BuildClient keeps the existing IOC entry point while using Queue internally.
func BuildClient(
	_ context.Context,
	dbPool *pgxpool.Pool,
	config *riverqueue.Config,
	registry *task.Registry,
) (*riverqueue.Client[pgx.Tx], error) {
	queue, err := New(dbPool, config)
	if err != nil {
		return nil, err
	}
	if registry != nil {
		queue.process = registry.Dispatch
	}
	return queue.client, nil
}
