package task

import (
	"context"
	"fmt"
	"sync"
)

// Registry maps task types to handlers and is safe for concurrent access.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

func (r *Registry) Register(taskType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = h
}

func (r *Registry) Get(taskType string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[taskType]
	return h, ok
}

// Dispatch resolves the handler for a task type and executes it.
// Business modules only register handlers; dispatching remains task infrastructure logic.
func (r *Registry) Dispatch(ctx context.Context, message *Task) error {
	if message == nil {
		return fmt.Errorf("task: cannot dispatch nil task")
	}

	handler, ok := r.Get(message.Type)
	if !ok {
		return fmt.Errorf("task: no handler registered for type %q", message.Type)
	}
	return handler.Handle(ctx, message)
}
