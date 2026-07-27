package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Outbox struct {
	ID        uint           `gorm:"primaryKey"`
	TaskID    uint           `gorm:"not null;index:idx_outbox_task"`
	Status    uint           `gorm:"not null;default:0;index:idx_outbox_status"`
	TaskType  string         `gorm:"column:job_kind;not null"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null"`
	QueueID   int64          `gorm:"column:job_id;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OutboxDao interface {
	Insert(ctx context.Context, tx *gorm.DB, record *Outbox) error
	FetchPending(ctx context.Context, limit int) ([]*Outbox, error)
	MarkPublished(ctx context.Context, id uint, queueID int64) error
}

type OutboxDaoImpl struct {
	DB *gorm.DB
}

func NewOutboxDao(db *gorm.DB) *OutboxDaoImpl {
	return &OutboxDaoImpl{DB: db}
}

func (d *OutboxDaoImpl) Insert(ctx context.Context, tx *gorm.DB, record *Outbox) error {
	return tx.WithContext(ctx).Create(record).Error
}

func (d *OutboxDaoImpl) FetchPending(ctx context.Context, limit int) ([]*Outbox, error) {
	var records []*Outbox
	err := d.DB.WithContext(ctx).
		Where("status = 0").
		Order("id ASC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("dao: fetch pending outbox: %w", err)
	}
	return records, nil
}

func (d *OutboxDaoImpl) MarkPublished(ctx context.Context, id uint, queueID int64) error {
	result := d.DB.WithContext(ctx).
		Model(&Outbox{}).
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{
			"status": 1,
			"job_id": queueID,
		})
	if result.Error != nil {
		return fmt.Errorf("dao: mark outbox published %d: %w", id, result.Error)
	}
	return nil
}
