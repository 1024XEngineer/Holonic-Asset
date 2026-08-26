package repository

import (
	"context"
	"errors"
	"fmt"

	tagdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type ProjectTagRepository struct {
	dao dao.ProjectTagDao
}

func NewProjectTagRepository(projectTagDao dao.ProjectTagDao) *ProjectTagRepository {
	return &ProjectTagRepository{dao: projectTagDao}
}

func (r *ProjectTagRepository) projectTagDao() (dao.ProjectTagDao, error) {
	if r != nil && r.dao != nil {
		return r.dao, nil
	}
	return nil, fmt.Errorf("repository: project tag storage is required")
}

func (r *ProjectTagRepository) CreateProjectTag(ctx context.Context, tag *tagdomain.Tag) error {
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

func (r *ProjectTagRepository) ListProjectTags(ctx context.Context, projectID uint) ([]tagdomain.Tag, error) {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return nil, err
	}
	tags, err := projectTagDao.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, normalizeProjectTagError(err)
	}
	result := make([]tagdomain.Tag, len(tags))
	for index := range tags {
		result[index] = convertProjectTagToDomain(&tags[index])
	}
	return result, nil
}

func (r *ProjectTagRepository) GetProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
) (*tagdomain.Tag, error) {
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

func (r *ProjectTagRepository) UpdateProjectTag(
	ctx context.Context,
	projectID uint,
	tagID uint,
	update *tagdomain.TagUpdate,
) (*tagdomain.Tag, error) {
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

func (r *ProjectTagRepository) DeleteProjectTag(ctx context.Context, projectID, tagID uint) error {
	projectTagDao, err := r.projectTagDao()
	if err != nil {
		return err
	}
	return normalizeProjectTagError(projectTagDao.Delete(ctx, projectID, tagID))
}

func convertProjectTagToDao(tag *tagdomain.Tag) *dao.ProjectTag {
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

func convertProjectTagToDomain(tag *dao.ProjectTag) tagdomain.Tag {
	if tag == nil {
		return tagdomain.Tag{}
	}
	return tagdomain.Tag{
		ID:          tag.ID,
		ProjectID:   tag.ProjectID,
		Name:        tag.Name,
		Description: tag.Description,
		Color:       tag.Color,
	}
}

func convertProjectTagUpdateToDao(update *tagdomain.TagUpdate) *dao.ProjectTagUpdate {
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
		return tagdomain.ErrTagNotFound
	case errors.Is(err, dao.ErrProjectTagConflict):
		return tagdomain.ErrTagConflict
	case errors.Is(err, dao.ErrProjectNotFound):
		return tagdomain.ErrTagProjectNotFound
	default:
		return err
	}
}

var _ tagdomain.Store = (*ProjectTagRepository)(nil)
