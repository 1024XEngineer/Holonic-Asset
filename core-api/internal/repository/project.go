package repository

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectRepository struct {
	projectDao dao.ProjectDao
}

func NewProjectRepository(projectDao dao.ProjectDao) domain.Store {
	return &projectRepository{projectDao: projectDao}
}

var _ domain.Store = (*projectRepository)(nil)

func (r *projectRepository) Insert(ctx context.Context, project *domain.Project) error {
	id, err := r.projectDao.CreateProject(ctx, convertProjectToDao(project))
	if err != nil {
		return err
	}
	project.ID = id
	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, projectID uint) (*domain.Project, error) {
	project, err := r.projectDao.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return convertProjectToDomain(project), nil
}

func (r *projectRepository) FindByUserID(ctx context.Context, userID uint) ([]*domain.Project, error) {
	projects, err := r.projectDao.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, convertProjectToDomain(project))
	}
	return result, nil
}

func (r *projectRepository) Update(ctx context.Context, update *domain.ProjectUpdate) error {
	return r.projectDao.Update(ctx, convertProjectUpdateToDao(update))
}

func (r *projectRepository) Remove(ctx context.Context, projectID uint) error {
	return r.projectDao.Delete(ctx, projectID)
}

func convertProjectToDao(project *domain.Project) *dao.Project {
	if project == nil {
		return nil
	}
	return &dao.Project{
		ID:             project.ID,
		UserID:         project.UserID,
		Name:           project.Name,
		GameType:       string(project.GameType),
		ViewType:       string(project.ViewType),
		TargetPlatform: string(project.TargetPlatform),
		Description:    project.Description,
		Reference:      project.Reference,
		Style:          project.Style,
	}
}

func convertProjectUpdateToDao(update *domain.ProjectUpdate) *dao.ProjectUpdate {
	if update == nil {
		return nil
	}
	converted := &dao.ProjectUpdate{
		ID:          update.ID,
		Name:        update.Name,
		Description: update.Description,
		Reference:   update.Reference,
		Style:       update.Style,
	}
	if update.GameType != nil {
		value := string(*update.GameType)
		converted.GameType = &value
	}
	if update.ViewType != nil {
		value := string(*update.ViewType)
		converted.ViewType = &value
	}
	if update.TargetPlatform != nil {
		value := string(*update.TargetPlatform)
		converted.TargetPlatform = &value
	}
	return converted
}

func convertProjectToDomain(project *dao.Project) *domain.Project {
	if project == nil {
		return nil
	}
	return &domain.Project{
		ID:             project.ID,
		UserID:         project.UserID,
		Name:           project.Name,
		GameType:       domain.GameType(project.GameType),
		ViewType:       domain.ViewType(project.ViewType),
		TargetPlatform: domain.PlatformType(project.TargetPlatform),
		Description:    project.Description,
		Reference:      project.Reference,
		Style:          project.Style,
	}
}
