package repository

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

func (r *AssetRepositoryImpl) projectTagDao() (dao.ProjectTagDao, error) {
	if r.ProjectTagDao != nil {
		return r.ProjectTagDao, nil
	}
	if db := r.transactionDB(); db != nil {
		return dao.NewGormProjectTagDao(db), nil
	}
	return nil, fmt.Errorf("repository: project tag storage is required")
}

func (r *AssetRepositoryImpl) CreateProjectTag(ctx context.Context, tag *domain.ProjectTag) error {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return err
	}
	value := convertProjectTagToDao(tag)
	if err := projectTagDao.Create(ctx, value); err != nil {
		return normalizeProjectTagError(err)
	}
	tag.ID = value.ID
	return nil
}

func (r *AssetRepositoryImpl) ListProjectTags(ctx context.Context, projectID uint) ([]domain.ProjectTag, error) {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return nil, err
	}
	tags, err := projectTagDao.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, normalizeProjectTagError(err)
	}
	result := make([]domain.ProjectTag, len(tags))
	for index := range tags {
		result[index] = convertProjectTagToDomain(&tags[index])
	}
	return result, nil
}

func (r *AssetRepositoryImpl) GetProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
) (*domain.ProjectTag, error) {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return nil, err
	}
	tag, err := projectTagDao.FindByID(ctx, projectID, tagID)
	if err != nil {
		return nil, normalizeProjectTagError(err)
	}
	value := convertProjectTagToDomain(tag)
	return &value, nil
}

func (r *AssetRepositoryImpl) UpdateProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
	update *domain.ProjectTagUpdate,
) (*domain.ProjectTag, error) {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return nil, err
	}
	tag, err := projectTagDao.Update(ctx, projectID, tagID, convertProjectTagUpdateToDao(update))
	if err != nil {
		return nil, normalizeProjectTagError(err)
	}
	value := convertProjectTagToDomain(tag)
	return &value, nil
}

func (r *AssetRepositoryImpl) DeleteProjectTag(ctx context.Context, projectID, tagID uint) error {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return err
	}
	return normalizeProjectTagError(projectTagDao.Delete(ctx, projectID, tagID))
}

func convertProjectTagToDao(tag *domain.ProjectTag) *dao.ProjectTag {
	if tag == nil {
		return nil
	}
	return &dao.ProjectTag{
		ID:          tag.ID,
		ProjectID:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
	}
}

func convertProjectTagToDomain(tag *dao.ProjectTag) domain.ProjectTag {
	if tag == nil {
		return domain.ProjectTag{}
	}
	return domain.ProjectTag{
		ID:          tag.ID,
		ProjectID:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
	}
}

func convertProjectTagUpdateToDao(update *domain.ProjectTagUpdate) *dao.ProjectTagUpdate {
	if update == nil {
		return nil
	}
	return &dao.ProjectTagUpdate{
		Name:        update.Name,
		Description: update.Description,
		Color:       update.Color,
	}
}

func normalizeProjectTagError(err error) error {
	switch {
	case errors.Is(err, dao.ErrProjectTagNotFound):
		return domain.ErrProjectTagNotFound
	case errors.Is(err, dao.ErrProjectTagConflict):
		return domain.ErrProjectTagConflict
	case errors.Is(err, dao.ErrProjectNotFound):
		return domain.ErrProjectTagProjectNotFound
	default:
		return err
	}
}
