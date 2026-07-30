package task

import (
	"context"
	"fmt"
	"sync"
)

// registry maps task types to handlers and is safe for concurrent access.
type registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func newRegistry() *registry {
	return &registry{handlers: make(map[string]Handler)}
}

func (r *registry) register(taskType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = h
}

func (r *registry) get(taskType string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[taskType]
	return h, ok
}

func (r *registry) dispatch(ctx context.Context, message *Task) (any, error) {
	if message == nil {
		return nil, fmt.Errorf("task: cannot dispatch nil task")
	}

	handler, ok := r.get(message.Type)
	if !ok {
		return nil, fmt.Errorf("task: no handler registered for type %q", message.Type)
	}
	return handler.Handle(ctx, message)
}
