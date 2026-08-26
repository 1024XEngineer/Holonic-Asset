package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrProjectNil       = errors.New("project is nil")
	ErrProjectUpdateNil = errors.New("project update is nil")
)

type Project struct {
	ID             uint `gorm:"primaryKey"`
	UserID         uint `gorm:"index"`
	Name           string
	GameType       string
	Perspective    string `gorm:"not null;default:Top-Down"`
	TargetPlatform string
	Description    string
	Reference      string
	Style          string
}

// ProjectUpdate contains the fields that may be changed without replacing a Project.
type ProjectUpdate struct {
	ID             uint
	Name           *string
	GameType       *string
	Perspective    *string
	TargetPlatform *string
	Description    *string
	Reference      *string
	Style          *string
}

type ProjectDao interface {
	CreateProject(ctx context.Context, project *Project) (uint, error)
	FindByID(ctx context.Context, id uint) (*Project, error)
	FindByUserID(ctx context.Context, userID uint) ([]*Project, error)
	Update(ctx context.Context, update *ProjectUpdate) error
	Delete(ctx context.Context, id uint) error
}

type GormProjectDao struct {
	db *gorm.DB
}

func NewGormProjectDao(db *gorm.DB) *GormProjectDao {
	return &GormProjectDao{db: db}
}

func (d *GormProjectDao) CreateProject(ctx context.Context, project *Project) (uint, error) {
	if project == nil {
		return 0, ErrProjectNil
	}
	if err := d.db.WithContext(ctx).Create(project).Error; err != nil {
		return 0, err
	}
	return project.ID, nil
}

func (d *GormProjectDao) FindByID(ctx context.Context, id uint) (*Project, error) {
	var project Project
	err := d.db.WithContext(ctx).First(&project, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (d *GormProjectDao) FindByUserID(ctx context.Context, userID uint) ([]*Project, error) {
	var projects []*Project
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id ASC").
		Find(&projects).Error
	return projects, err
}

func (d *GormProjectDao) Update(ctx context.Context, update *ProjectUpdate) error {
	if update == nil {
		return ErrProjectUpdateNil
	}

	fields := projectUpdateFields(update)
	if len(fields) == 0 {
		return nil
	}
	result := d.db.WithContext(ctx).Model(&Project{}).Where("id = ?", update.ID).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (d *GormProjectDao) Delete(ctx context.Context, id uint) error {
	result := d.db.WithContext(ctx).Delete(&Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func projectUpdateFields(update *ProjectUpdate) map[string]any {
	fields := make(map[string]any)
	if update.Name != nil {
		fields["name"] = *update.Name
	}
	if update.GameType != nil {
		fields["game_type"] = *update.GameType
	}
	if update.Perspective != nil {
		fields["perspective"] = *update.Perspective
	}
	if update.TargetPlatform != nil {
		fields["target_platform"] = *update.TargetPlatform
	}
	if update.Description != nil {
		fields["description"] = *update.Description
	}
	if update.Reference != nil {
		fields["reference"] = *update.Reference
	}
	if update.Style != nil {
		fields["style"] = *update.Style
	}
	return fields
}

var _ ProjectDao = (*GormProjectDao)(nil)
