package task

import "context"

// Producer publishes tasks to the configured queue.
type Producer interface {
	Publish(ctx context.Context, task *Task) error
}

// ProcessFunc handles a task received by a Consumer.
type ProcessFunc func(ctx context.Context, task *Task) error

// Consumer receives tasks and delegates processing to ProcessFunc.
type Consumer interface {
	Start(ctx context.Context, fn ProcessFunc) error
	Stop() error
}

// Queue combines task production and consumption.
type Queue interface {
	Producer
	Consumer
}
