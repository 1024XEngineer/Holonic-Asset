package task

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

const (
	defaultOutboxBatchSize    = 100
	defaultOutboxPollInterval = time.Second
)

// Manager is the single entry point for task registration, lifecycle, and
// task state operations. Queue consumption and transactional outbox dispatch
// are managed internally.
type Manager interface {
	Register(taskType string, handler Handler)
	Start(ctx context.Context) error
	Stop() error
	Publish(ctx context.Context, task *Task) (uint, error)
	GetDetail(ctx context.Context, taskID uint) (*Task, error)
	List(ctx context.Context, filter *ListFilter) ([]*Task, error)
	Cancel(ctx context.Context, taskID uint) error
	Complete(ctx context.Context, taskID uint) error
}

type manager struct {
	store      TaskStore
	queue      *queue
	dispatcher *dispatcher

	outboxBatchSize    int
	outboxPollInterval time.Duration

	stateMu sync.Mutex
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	stopOnce sync.Once
	stopErr  error
}

// NewManager creates the task module and owns all task execution internals.
func NewManager(ctx context.Context, cfg config.QueueConfig, store TaskStore) (Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("task: task store is required")
	}

	queue, err := newQueue(ctx, cfg, store)
	if err != nil {
		return nil, err
	}

	return &manager{
		store:              store,
		queue:              queue,
		dispatcher:         newDispatcher(store, queue),
		outboxBatchSize:    defaultOutboxBatchSize,
		outboxPollInterval: defaultOutboxPollInterval,
	}, nil
}

func (m *manager) Register(taskType string, handler Handler) {
	m.queue.Register(taskType, handler)
}

func (m *manager) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("task: start context is required")
	}

	m.stateMu.Lock()
	if m.stopped {
		m.stateMu.Unlock()
		return fmt.Errorf("task: manager is already stopped")
	}
	if m.started {
		m.stateMu.Unlock()
		return nil
	}

	if err := m.queue.start(ctx); err != nil {
		m.stateMu.Unlock()
		return err
	}

	outboxCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.started = true
	m.wg.Add(1)
	m.stateMu.Unlock()

	go func() {
		defer cancel()
		m.runOutbox(outboxCtx)
	}()
	return nil
}

func (m *manager) Stop() error {
	m.stopOnce.Do(func() {
		m.stateMu.Lock()
		m.stopped = true
		cancel := m.cancel
		m.stateMu.Unlock()

		if cancel != nil {
			cancel()
		}
		m.wg.Wait()
		m.stopErr = m.queue.stop()
	})
	return m.stopErr
}

func (m *manager) Publish(ctx context.Context, task *Task) (uint, error) {
	if task == nil {
		return 0, fmt.Errorf("task: cannot publish nil task")
	}
	return m.store.CreateWithOutbox(ctx, task)
}

func (m *manager) GetDetail(ctx context.Context, taskID uint) (*Task, error) {
	return m.store.GetTaskByID(ctx, taskID)
}

func (m *manager) List(ctx context.Context, filter *ListFilter) ([]*Task, error) {
	return m.store.ListTasks(ctx, filter)
}

func (m *manager) Cancel(ctx context.Context, taskID uint) error {
	return m.store.UpdateTaskStatus(ctx, taskID, StatusCancelled)
}

func (m *manager) Complete(ctx context.Context, taskID uint) error {
	return m.store.CompleteTask(ctx, taskID)
}

func (m *manager) runOutbox(ctx context.Context) {
	defer m.wg.Done()

	m.dispatchOutbox(ctx)
	ticker := time.NewTicker(m.outboxPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.dispatchOutbox(ctx)
		}
	}
}

func (m *manager) dispatchOutbox(ctx context.Context) {
	if _, err := m.dispatcher.run(ctx, m.outboxBatchSize); err != nil {
		log.Printf("task: dispatch outbox: %v", err)
	}
}

var _ Manager = (*manager)(nil)
