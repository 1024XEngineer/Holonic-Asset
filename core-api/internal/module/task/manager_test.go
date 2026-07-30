package task

import (
	"context"
	"encoding/json"
	"testing"
)

type taskStoreStub struct {
	createdTask *Task
	status      Status
	result      json.RawMessage
}

func (s *taskStoreStub) CreateWithOutbox(_ context.Context, task *Task) (uint, error) {
	s.createdTask = task
	return 42, nil
}

func (*taskStoreStub) GetTaskByID(context.Context, uint) (*Task, error) {
	return &Task{ID: 42}, nil
}

func (*taskStoreStub) ListTasksByStatus(context.Context, Status) ([]*Task, error) {
	return []*Task{{ID: 42, Status: StatusPending}}, nil
}

func (s *taskStoreStub) UpdateTaskStatus(_ context.Context, _ uint, status Status) error {
	s.status = status
	return nil
}

func (s *taskStoreStub) UpdateTaskResult(_ context.Context, _ uint, result json.RawMessage) error {
	s.result = result
	return nil
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

	detail, err := manager.GetDetail(context.Background(), id)
	if err != nil {
		t.Fatalf("get task detail: %v", err)
	}
	if detail.ID != id {
		t.Fatalf("unexpected task detail: %+v", detail)
	}

	tasks, err := manager.ListByStatus(context.Background(), StatusPending)
	if err != nil {
		t.Fatalf("list tasks by status: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != id || tasks[0].Status != StatusPending {
		t.Fatalf("unexpected task list: %+v", tasks)
	}

	if err := manager.Cancel(context.Background(), id); err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if store.status != StatusCancelled {
		t.Fatalf("unexpected task status: %s", store.status)
	}
}

func TestManagerRejectsNilPublish(t *testing.T) {
	manager := &manager{store: &taskStoreStub{}}

	if _, err := manager.Publish(context.Background(), nil); err == nil {
		t.Fatal("expected nil publish error")
	}
}
