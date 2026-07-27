// Package river adapts River to the generic task module.
package river

import (
	"context"
	"fmt"

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
	registry *task.Registry
}

func (w *worker) Work(ctx context.Context, job *riverqueue.Job[taskArgs]) error {
	handler, ok := w.registry.Get(job.Args.Task.Type)
	if !ok {
		return fmt.Errorf("river: no task handler for type %q", job.Args.Task.Type)
	}
	return handler.Handle(ctx, &job.Args.Task)
}

type Producer struct {
	client *riverqueue.Client[pgx.Tx]
}

func NewProducer(client *riverqueue.Client[pgx.Tx]) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Publish(ctx context.Context, message *task.Task) error {
	result, err := p.client.Insert(ctx, taskArgs{Task: *message}, &riverqueue.InsertOpts{
		UniqueOpts: riverqueue.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return fmt.Errorf("river: publish task %q: %w", message.Type, err)
	}
	if result.UniqueSkippedAsDuplicate {
		return nil
	}
	_ = result.Job.ID
	return nil
}

var _ task.Producer = (*Producer)(nil)

func BuildClient(
	ctx context.Context,
	dbPool *pgxpool.Pool,
	config *riverqueue.Config,
	registry *task.Registry,
) (*riverqueue.Client[pgx.Tx], error) {
	workers := riverqueue.NewWorkers()
	if registry != nil {
		riverqueue.AddWorker(workers, &worker{registry: registry})
	}
	config.Workers = workers

	client, err := riverqueue.NewClient(riverpgxv5.New(dbPool), config)
	if err != nil {
		return nil, fmt.Errorf("river: create client: %w", err)
	}
	return client, nil
}
