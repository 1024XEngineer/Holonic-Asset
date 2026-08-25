package task

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

type taskStoreStub struct {
	createdTask   *Task
	status        Status
	statusUpdates []Status
	statusErr     error
	result        json.RawMessage
	resultStatus  Status
	resultCalls   int
	failure       string
	failureCalls  int
	listFilter    *ListFilter
	retriedID     uint
	retryStatus   Status
	deletedID     uint
}

func (s *taskStoreStub) CreateWithOutbox(_ context.Context, task *Task) (uint, error) {
	s.createdTask = task
	return 42, nil
}

func (s *taskStoreStub) RetryFailedTask(_ context.Context, taskID uint, completionStatus Status) error {
	s.retriedID = taskID
	s.retryStatus = completionStatus
	return nil
}

func (s *taskStoreStub) DeleteFailedTask(_ context.Context, taskID uint) error {
	s.deletedID = taskID
	return nil
}

func (*taskStoreStub) GetTaskByID(context.Context, uint) (*Task, error) {
	return &Task{ID: 42}, nil
}

func (s *taskStoreStub) ListTasks(_ context.Context, filter *ListFilter) ([]*Task, error) {
	s.listFilter = filter
	return []*Task{{ID: 42, Status: StatusPending}}, nil
}

func (s *taskStoreStub) UpdateTaskStatus(_ context.Context, _ uint, status Status) error {
	s.status = status
	s.statusUpdates = append(s.statusUpdates, status)
	return s.statusErr
}

func (s *taskStoreStub) UpdateTaskResult(_ context.Context, _ uint, status Status, result json.RawMessage) error {
	s.resultStatus = status
	s.result = result
	s.resultCalls++
	return nil
}

func (s *taskStoreStub) CompleteTask(_ context.Context, _ uint) error {
	s.status = StatusCompleted
	s.statusUpdates = append(s.statusUpdates, StatusCompleted)
	return s.statusErr
}

func (s *taskStoreStub) UpdateTaskFailure(_ context.Context, _ uint, errorMessage string) error {
	s.status = StatusFailed
	s.statusUpdates = append(s.statusUpdates, StatusFailed)
	s.failure = errorMessage
	s.failureCalls++
	return s.statusErr
}

func (*taskStoreStub) FetchPendingOutbox(context.Context, int) ([]OutboxRecord, error) {
	return nil, nil
}

func (*taskStoreStub) MarkOutboxPublished(context.Context, uint, int64) error {
	return nil
}

func TestManagerDelegatesTaskOperations(t *testing.T) {
	store := &taskStoreStub{}
	manager := &manager{store: store}
	message := &Task{Type: "example.v1"}

	id, err := manager.Publish(context.Background(), message)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if id != 42 || store.createdTask != message {
		t.Fatalf("unexpected create delegation: id=%d task=%p", id, store.createdTask)
	}

	if err := manager.RetryFailed(context.Background(), id, StatusAwaitingApplication); err != nil {
		t.Fatalf("retry failed task: %v", err)
	}
	if store.retriedID != id || store.retryStatus != StatusAwaitingApplication {
		t.Fatalf("unexpected retry delegation: id=%d status=%s", store.retriedID, store.retryStatus)
	}

	if err := manager.DeleteFailed(context.Background(), id); err != nil {
		t.Fatalf("delete failed task: %v", err)
	}
	if store.deletedID != id {
		t.Fatalf("unexpected delete delegation: id=%d", store.deletedID)
	}

	detail, err := manager.GetDetail(context.Background(), id)
	if err != nil {
		t.Fatalf("get task detail: %v", err)
	}
	if detail.ID != id {
		t.Fatalf("unexpected task detail: %+v", detail)
	}

	filter := &ListFilter{Statuses: []Status{StatusPending}, Types: []string{"example.v1"}, Limit: 20}
	tasks, err := manager.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if store.listFilter != filter || len(tasks) != 1 || tasks[0].ID != id || tasks[0].Status != StatusPending {
		t.Fatalf("unexpected task list: filter=%+v tasks=%+v", store.listFilter, tasks)
	}

	if err := manager.Cancel(context.Background(), id); err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if store.status != StatusCancelled {
		t.Fatalf("unexpected task status: %s", store.status)
	}

	if err := manager.Complete(context.Background(), id); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if store.status != StatusCompleted {
		t.Fatalf("unexpected completed task status: %s", store.status)
	}
}

func TestManagerRejectsNilPublish(t *testing.T) {
	manager := &manager{store: &taskStoreStub{}}

	if _, err := manager.Publish(context.Background(), nil); err == nil {
		t.Fatal("expected nil publish error")
	}
}

type executionStoreStub struct {
	statusErr  error
	resultErr  error
	failureErr error
	status     Status
	result     json.RawMessage
	resultCall int
	failure    string
	failureID  uint
}

func (s *executionStoreStub) UpdateTaskStatus(_ context.Context, _ uint, status Status) error {
	if s.statusErr != nil {
		return s.statusErr
	}
	s.status = status
	return nil
}

func (s *executionStoreStub) UpdateTaskResult(_ context.Context, _ uint, _ Status, result json.RawMessage) error {
	if s.resultErr != nil {
		return s.resultErr
	}
	s.result = result
	s.resultCall++
	return nil
}

func (s *executionStoreStub) UpdateTaskFailure(_ context.Context, taskID uint, message string) error {
	if s.failureErr != nil {
		return s.failureErr
	}
	s.failureID = taskID
	s.failure = message
	return nil
}

func (s *executionStoreStub) CompleteTask(context.Context, uint) error { return nil }

func TestNewManagerAndQueueValidateRequiredInputs(t *testing.T) {
	if _, err := NewManager(context.Background(), config.QueueConfig{}, nil); err == nil {
		t.Fatal("expected manager to reject a nil store")
	}

	store := &executionStoreStub{}
	tests := []struct {
		name string
		cfg  config.QueueConfig
		want string
	}{
		{name: "database URL", cfg: config.QueueConfig{MaxWorkers: 1}, want: "database URL"},
		{name: "worker count", cfg: config.QueueConfig{DatabaseURL: "postgres://localhost/test"}, want: "max workers"},
		{name: "worker limit", cfg: config.QueueConfig{DatabaseURL: "postgres://localhost/test", MaxWorkers: 10001}, want: "invalid number of workers"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := newQueue(context.Background(), tc.cfg, store); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newQueue error = %v, want message containing %q", err, tc.want)
			}
		})
	}
	if _, err := newQueue(context.Background(), config.QueueConfig{
		DatabaseURL: "postgres://localhost/test",
		MaxWorkers:  1,
	}, nil); err == nil || !strings.Contains(err.Error(), "task result store") {
		t.Fatalf("newQueue nil repo error = %v", err)
	}
	if _, err := NewManager(context.Background(), config.QueueConfig{
		DatabaseURL: "postgres://localhost/test",
	}, &taskStoreStub{}); err == nil || !strings.Contains(err.Error(), "max workers") {
		t.Fatalf("NewManager queue configuration error = %v", err)
	}
	if _, err := newQueue(context.Background(), config.QueueConfig{
		DatabaseURL: "://malformed",
		MaxWorkers:  1,
	}, store); err == nil || !strings.Contains(err.Error(), "create database pool") {
		t.Fatalf("newQueue pool creation error = %v", err)
	}
}

func TestManagerStartValidatesContextAndState(t *testing.T) {
	m := &manager{}
	var nilContext context.Context
	if err := m.Start(nilContext); err == nil {
		t.Fatal("expected nil start context to fail")
	}

	m = &manager{started: true}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("starting an already-started manager: %v", err)
	}

	m = &manager{stopped: true}
	if err := m.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "already stopped") {
		t.Fatalf("stopped manager error = %v", err)
	}
}

func TestManagerRegisterAndRunOutbox(t *testing.T) {
	store := &outboxDispatchStore{}
	m := &manager{
		queue:              &queue{registry: newRegistry()},
		dispatcher:         newDispatcher(store, &queuePublisherStub{}),
		outboxBatchSize:    3,
		outboxPollInterval: time.Hour,
	}
	m.Register("registered", HandlerFunc(func(context.Context, *Task) (any, error) { return struct{}{}, nil }))
	if _, ok := m.queue.registry.get("registered"); !ok {
		t.Fatal("manager did not register handler")
	}

	m.wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.runOutbox(ctx)
	if store.fetchLimit != 3 {
		t.Fatalf("outbox batch size = %d, want 3", store.fetchLimit)
	}
	m.dispatcher = newDispatcher(&outboxDispatchStore{fetchErr: errors.New("poll failed")}, &queuePublisherStub{})
	m.dispatchOutbox(context.Background())
}

func TestManagerStopIsIdempotentBeforeStart(t *testing.T) {
	cfg := config.QueueConfig{
		DatabaseURL: "postgres://localhost/test",
		MaxWorkers:  1,
		JobTimeout:  time.Second,
	}
	queue, err := newQueue(context.Background(), cfg, &executionStoreStub{})
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	m := &manager{queue: queue}
	if err := m.Stop(); err != nil {
		t.Fatalf("stop manager: %v", err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("stop manager twice: %v", err)
	}

	created, err := NewManager(context.Background(), cfg, &taskStoreStub{})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if err := created.Stop(); err != nil {
		t.Fatalf("stop newly created manager: %v", err)
	}

	queue, err = newQueue(context.Background(), cfg, &executionStoreStub{})
	if err != nil {
		t.Fatalf("create queue for cancelled start: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (&manager{queue: queue}).Start(ctx); err == nil {
		t.Fatal("expected cancelled queue start to fail")
	}
	if err := queue.stop(); err != nil {
		t.Fatalf("stop queue after failed start: %v", err)
	}

	queue, err = newQueue(context.Background(), cfg, &executionStoreStub{})
	if err != nil {
		t.Fatalf("create queue for cancel stop: %v", err)
	}
	cancelCalled := false
	started := &manager{queue: queue, started: true, cancel: func() { cancelCalled = true }}
	if err := started.Stop(); err != nil {
		t.Fatalf("stop started manager: %v", err)
	}
	if !cancelCalled {
		t.Fatal("stop should cancel the outbox context")
	}
}

func TestManagerStartRunsOutboxUntilContextCancellation(t *testing.T) {
	client := &queueClientStub{}
	store := &outboxDispatchStore{fetchSignal: make(chan struct{}, 2)}
	m := &manager{
		queue:              &queue{client: client, registry: newRegistry(), repo: &executionStoreStub{}},
		dispatcher:         newDispatcher(store, &queuePublisherStub{}),
		outboxBatchSize:    1,
		outboxPollInterval: time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := m.Start(ctx); err != nil {
		cancel()
		t.Fatalf("start manager: %v", err)
	}
	if !client.started {
		cancel()
		t.Fatal("manager did not start queue")
	}
	select {
	case <-store.fetchSignal:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("manager did not dispatch initial outbox batch")
	}
	select {
	case <-store.fetchSignal:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("manager did not poll outbox on ticker")
	}
	cancel()
	m.wg.Wait()
}
