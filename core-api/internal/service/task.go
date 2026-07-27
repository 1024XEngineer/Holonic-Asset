package service

import (
	"context"
	"encoding/json"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

type TaskService interface {
	Create(ctx context.Context, task *domain.Task) (uint, error)
	GetDetail(ctx context.Context, taskID uint) (*domain.Task, error)
	UpdateStatus(ctx context.Context, taskID uint, status domain.Status) error
	UpdateResult(ctx context.Context, taskID uint, result json.RawMessage) error
}

type TaskServiceImpl struct {
	TaskRepository repository.TaskRepository
}

func NewTaskService(r repository.TaskRepository) *TaskServiceImpl {
	return &TaskServiceImpl{TaskRepository: r}
}

func (s *TaskServiceImpl) Create(ctx context.Context, task *domain.Task) (uint, error) {
	return s.TaskRepository.CreateWithOutbox(ctx, task)
}

func (s *TaskServiceImpl) GetDetail(ctx context.Context, taskID uint) (*domain.Task, error) {
	return s.TaskRepository.GetTaskByID(ctx, taskID)
}

func (s *TaskServiceImpl) UpdateStatus(ctx context.Context, taskID uint, status domain.Status) error {
	return s.TaskRepository.UpdateTaskStatus(ctx, taskID, status)
}

func (s *TaskServiceImpl) UpdateResult(ctx context.Context, taskID uint, result json.RawMessage) error {
	return s.TaskRepository.UpdateTaskResult(ctx, taskID, result)
}
