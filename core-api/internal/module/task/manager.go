package task

import (
	"context"
)

type TaskManager interface {
	Create(ctx context.Context, task *Task) (uint, error)
	GetDetail(ctx context.Context, taskID uint) (*Task, error)
	ListByStatus(ctx context.Context, status Status) ([]*Task, error)
	UpdateStatus(ctx context.Context, taskID uint, status Status) error
}

type TaskManagerImpl struct {
	store TaskStore
}

func NewTaskManager(store TaskStore) *TaskManagerImpl {
	return &TaskManagerImpl{store: store}
}

func (m *TaskManagerImpl) Create(ctx context.Context, task *Task) (uint, error) {
	return m.store.CreateWithOutbox(ctx, task)
}

func (m *TaskManagerImpl) GetDetail(ctx context.Context, taskID uint) (*Task, error) {
	return m.store.GetTaskByID(ctx, taskID)
}

func (m *TaskManagerImpl) ListByStatus(ctx context.Context, status Status) ([]*Task, error) {
	return m.store.ListTasksByStatus(ctx, status)
}

func (m *TaskManagerImpl) UpdateStatus(ctx context.Context, taskID uint, status Status) error {
	return m.store.UpdateTaskStatus(ctx, taskID, status)
}

var _ TaskManager = (*TaskManagerImpl)(nil)
