package task

import "context"

// Handler is implemented by business modules that consume task messages.
type Handler interface {
	Handle(ctx context.Context, task *Task) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, task *Task) error

func (f HandlerFunc) Handle(ctx context.Context, task *Task) error {
	return f(ctx, task)
}
