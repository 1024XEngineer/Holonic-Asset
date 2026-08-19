package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Task struct {
	ID        uint
	Type      string
	Status    uint
	Payload   datatypes.JSON `gorm:"type:jsonb"`
	Result    datatypes.JSON `gorm:"type:jsonb"`
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TaskListFilter struct {
	Statuses []uint
	Types    []string
	BeforeID uint
	Limit    int
}

type TaskDao interface {
	Create(ctx context.Context, task *Task) error

	UpdateStatus(ctx context.Context, taskID uint, status uint) error
	UpdateStatusFrom(ctx context.Context, taskID uint, currentStatus uint, status uint) (bool, error)
	UpdateResult(ctx context.Context, taskID uint, status uint, result datatypes.JSON) error
	UpdateFailure(ctx context.Context, taskID uint, status uint, errorMessage string) error
	GetDetail(ctx context.Context, taskID uint) (*Task, error)
	List(ctx context.Context, filter TaskListFilter) ([]*Task, error)
}

type TaskDaoImpl struct {
	DB *gorm.DB
}

func NewTaskDao(db *gorm.DB) *TaskDaoImpl {
	return &TaskDaoImpl{DB: db}
}

func (d *TaskDaoImpl) Create(ctx context.Context, task *Task) error {
	return d.DB.WithContext(ctx).Create(task).Error
}

func (d *TaskDaoImpl) UpdateStatus(ctx context.Context, taskID uint, status uint) error {
	result := d.DB.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", taskID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("dao: update status task %d: %w", taskID, result.Error)
	}
	return nil
}

func (d *TaskDaoImpl) UpdateStatusFrom(
	ctx context.Context,
	taskID uint,
	currentStatus uint,
	status uint,
) (bool, error) {
	result := d.DB.WithContext(ctx).
		Model(&Task{}).
		Where("id = ? AND status = ?", taskID, currentStatus).
		Update("status", status)
	if result.Error != nil {
		return false, fmt.Errorf("dao: transition task %d status: %w", taskID, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (d *TaskDaoImpl) UpdateResult(ctx context.Context, taskID uint, status uint, result datatypes.JSON) error {
	query := d.DB.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status": status,
			"result": result,
		})
	if query.Error != nil {
		return fmt.Errorf("dao: update result task %d: %w", taskID, query.Error)
	}
	return nil
}

func (d *TaskDaoImpl) UpdateFailure(ctx context.Context, taskID uint, status uint, errorMessage string) error {
	query := d.DB.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status": status,
			"error":  errorMessage,
		})
	if query.Error != nil {
		return fmt.Errorf("dao: update failure task %d: %w", taskID, query.Error)
	}
	return nil
}

func (d *TaskDaoImpl) GetDetail(ctx context.Context, taskID uint) (*Task, error) {
	var task Task
	err := d.DB.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, fmt.Errorf("dao: get task %d: %w", taskID, err)
	}
	return &task, nil
}

func (d *TaskDaoImpl) List(ctx context.Context, filter TaskListFilter) ([]*Task, error) {
	var tasks []*Task
	if err := d.listQuery(ctx, filter).Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("dao: list tasks: %w", err)
	}
	return tasks, nil
}

func (d *TaskDaoImpl) listQuery(ctx context.Context, filter TaskListFilter) *gorm.DB {
	query := d.DB.WithContext(ctx)
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	}
	if len(filter.Types) > 0 {
		query = query.Where("type IN ?", filter.Types)
	}
	if filter.BeforeID > 0 {
		query = query.Where("id < ?", filter.BeforeID)
	}
	query = query.Order("id DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	return query
}
