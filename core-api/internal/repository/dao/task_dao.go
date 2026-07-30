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

type TaskDao interface {
	Create(ctx context.Context, task *Task) error

	UpdateStatus(ctx context.Context, taskID uint, status uint) error
	UpdateResult(ctx context.Context, taskID uint, status uint, result datatypes.JSON) error
	GetDetail(ctx context.Context, taskID uint) (*Task, error)
	ListByStatus(ctx context.Context, status uint) ([]*Task, error)
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

func (d *TaskDaoImpl) GetDetail(ctx context.Context, taskID uint) (*Task, error) {
	var task Task
	err := d.DB.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, fmt.Errorf("dao: get task %d: %w", taskID, err)
	}
	return &task, nil
}

func (d *TaskDaoImpl) ListByStatus(ctx context.Context, status uint) ([]*Task, error) {
	var tasks []*Task
	err := d.DB.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at ASC, id ASC").
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("dao: list tasks by status %d: %w", status, err)
	}
	return tasks, nil
}
