package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const projectTagColumns = "id, project_id, name, description, color"

var (
	ErrProjectTagNotFound  = errors.New("project tag not found")
	ErrProjectTagConflict  = errors.New("project tag already exists")
	ErrProjectTagNil       = errors.New("project tag is nil")
	ErrProjectTagUpdateNil = errors.New("project tag update is nil")
)

// ProjectTagUpdate contains the persisted fields that may be changed.
type ProjectTagUpdate struct {
	Name        *string
	Description *string
	Color       *string
}

type ProjectTagDao interface {
	Create(ctx context.Context, tag *ProjectTag) error
	ListByProjectID(ctx context.Context, projectID uint) ([]ProjectTag, error)
	FindByID(ctx context.Context, projectID, tagID uint) (*ProjectTag, error)
	Update(ctx context.Context, projectID, tagID uint, update *ProjectTagUpdate) (*ProjectTag, error)
	Delete(ctx context.Context, projectID, tagID uint) error
}

type GormProjectTagDao struct {
	db *gorm.DB
}

func NewGormProjectTagDao(db *gorm.DB) *GormProjectTagDao {
	return &GormProjectTagDao{db: db}
}

func (d *GormProjectTagDao) Create(ctx context.Context, tag *ProjectTag) error {
	if tag == nil {
		return ErrProjectTagNil
	}
	if err := d.ensureProjectExists(ctx, tag.ProjectID); err != nil {
		return err
	}
	result := d.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(tag)
	if result.Error != nil {
		return normalizeProjectTagWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProjectTagConflict
	}
	return nil
}

func (d *GormProjectTagDao) ListByProjectID(ctx context.Context, projectID uint) ([]ProjectTag, error) {
	if err := d.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}
	tags := make([]ProjectTag, 0)
	if err := d.db.WithContext(ctx).
		Select(projectTagColumns).
		Where("project_id = ?", projectID).
		Order("lower(trim(name)) ASC, id ASC").
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (d *GormProjectTagDao) FindByID(
	ctx context.Context,
	projectID uint,
	tagID uint,
) (*ProjectTag, error) {
	var tag ProjectTag
	err := d.db.WithContext(ctx).
		Select(projectTagColumns).
		Where("project_id = ? AND id = ?", projectID, tagID).
		First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProjectTagNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (d *GormProjectTagDao) Update(
	ctx context.Context,
	projectID uint,
	tagID uint,
	update *ProjectTagUpdate,
) (*ProjectTag, error) {
	if update == nil {
		return nil, ErrProjectTagUpdateNil
	}
	fields := projectTagUpdateFields(update)
	if len(fields) > 0 {
		result := d.db.WithContext(ctx).
			Model(&ProjectTag{}).
			Where("project_id = ? AND id = ?", projectID, tagID).
			Updates(fields)
		if result.Error != nil {
			return nil, normalizeProjectTagWriteError(result.Error)
		}
		if result.RowsAffected == 0 {
			return nil, ErrProjectTagNotFound
		}
	}
	return d.FindByID(ctx, projectID, tagID)
}

func (d *GormProjectTagDao) Delete(ctx context.Context, projectID, tagID uint) error {
	result := d.db.WithContext(ctx).
		Where("project_id = ? AND id = ?", projectID, tagID).
		Delete(&ProjectTag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectTagNotFound
	}
	return nil
}

func (d *GormProjectTagDao) ensureProjectExists(ctx context.Context, projectID uint) error {
	var project Project
	err := d.db.WithContext(ctx).Select("id").First(&project, projectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrProjectNotFound
	}
	return err
}

func projectTagUpdateFields(update *ProjectTagUpdate) map[string]any {
	fields := make(map[string]any)
	if update.Name != nil {
		fields["name"] = *update.Name
	}
	if update.Description != nil {
		fields["description"] = *update.Description
	}
	if update.Color != nil {
		fields["color"] = *update.Color
	}
	return fields
}

func normalizeProjectTagWriteError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrProjectTagConflict
	}
	var stateError interface{ SQLState() string }
	if errors.As(err, &stateError) && stateError.SQLState() == "23505" {
		return ErrProjectTagConflict
	}
	return err
}

var _ ProjectTagDao = (*GormProjectTagDao)(nil)
