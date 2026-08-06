package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type TaskRepository = domain.TaskStore

type TaskRepositoryImpl struct {
	DB        *gorm.DB
	TaskDao   dao.TaskDao
	OutboxDao dao.OutboxDao
}

func NewTaskRepository(db *gorm.DB) *TaskRepositoryImpl {
	return &TaskRepositoryImpl{
		DB:        db,
		TaskDao:   dao.NewTaskDao(db),
		OutboxDao: dao.NewOutboxDao(db),
	}
}

var _ domain.TaskStore = (*TaskRepositoryImpl)(nil)

func (r *TaskRepositoryImpl) CreateWithOutbox(ctx context.Context, task *domain.Task) (uint, error) {
	task.Status = domain.StatusPending

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		daoTask := &dao.Task{
			Type:      task.Type,
			Status:    uint(task.Status),
			Payload:   datatypes.JSON(task.Payload),
			Result:    datatypes.JSON(task.Result),
			Error:     task.Error,
			CreatedAt: task.CreatedAt,
			UpdatedAt: task.UpdatedAt,
		}
		if err := tx.WithContext(ctx).Create(daoTask).Error; err != nil {
			return fmt.Errorf("repo: insert task: %w", err)
		}
		task.ID = daoTask.ID

		payload, err := json.Marshal(task)
		if err != nil {
			return fmt.Errorf("repo: marshal task %q: %w", task.Type, err)
		}

		outbox := &dao.Outbox{
			TaskID:   task.ID,
			TaskType: task.Type,
			Payload:  datatypes.JSON(payload),
			Status:   0,
		}
		if err := r.OutboxDao.Insert(ctx, tx, outbox); err != nil {
			return fmt.Errorf("repo: insert outbox for task %d: %w", task.ID, err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return task.ID, nil
}

func (r *TaskRepositoryImpl) UpdateTaskStatus(ctx context.Context, taskID uint, status domain.Status) error {
	return r.TaskDao.UpdateStatus(ctx, taskID, uint(status))
}

func (r *TaskRepositoryImpl) UpdateTaskResult(ctx context.Context, taskID uint, result json.RawMessage) error {
	return r.TaskDao.UpdateResult(ctx, taskID, uint(domain.StatusCompleted), datatypes.JSON(result))
}

func (r *TaskRepositoryImpl) GetTaskByID(ctx context.Context, taskID uint) (*domain.Task, error) {
	dt, err := r.TaskDao.GetDetail(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("repo: get task %d: %w", taskID, err)
	}
	return taskFromDAO(dt), nil
}

func (r *TaskRepositoryImpl) ListTasks(
	ctx context.Context,
	filter *domain.ListFilter,
) ([]*domain.Task, error) {
	if filter == nil {
		return nil, fmt.Errorf("repo: task list filter is required")
	}

	statuses := make([]uint, len(filter.Statuses))
	for i, status := range filter.Statuses {
		statuses[i] = uint(status)
	}
	items, err := r.TaskDao.List(ctx, dao.TaskListFilter{
		Statuses: statuses,
		Types:    filter.Types,
		BeforeID: filter.BeforeID,
		Limit:    filter.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("repo: list tasks: %w", err)
	}

	tasks := make([]*domain.Task, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, taskFromDAO(item))
	}
	return tasks, nil
}

func taskFromDAO(item *dao.Task) *domain.Task {
	return &domain.Task{
		ID:        item.ID,
		Type:      item.Type,
		Status:    domain.Status(item.Status),
		Payload:   json.RawMessage(item.Payload),
		Result:    json.RawMessage(item.Result),
		Error:     item.Error,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func (r *TaskRepositoryImpl) FetchPendingOutbox(ctx context.Context, limit int) ([]domain.OutboxRecord, error) {
	records, err := r.OutboxDao.FetchPending(ctx, limit)
	if err != nil {
		return nil, err
	}

	outbox := make([]domain.OutboxRecord, 0, len(records))
	for _, record := range records {
		outbox = append(outbox, domain.OutboxRecord{
			ID:      record.ID,
			Payload: json.RawMessage(record.Payload),
		})
	}
	return outbox, nil
}

func (r *TaskRepositoryImpl) MarkOutboxPublished(ctx context.Context, outboxID uint, queueID int64) error {
	return r.OutboxDao.MarkPublished(ctx, outboxID, queueID)
}
