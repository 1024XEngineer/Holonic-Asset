package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectTagDaoStub struct {
	dao.ProjectTagDao
	created *dao.ProjectTag
	tags    []dao.ProjectTag
}

func (s *projectTagDaoStub) Create(_ context.Context, tag *dao.ProjectTag) error {
	s.created = tag
	tag.ID = 7
	return nil
}

func (s *projectTagDaoStub) ListByProjectID(context.Context, uint) ([]dao.ProjectTag, error) {
	return s.tags, nil
}

func TestAssetRepositoryConvertsProjectTagsAcrossDomainBoundary(t *testing.T) {
	projectTagDao := &projectTagDaoStub{tags: []dao.ProjectTag{{
		ID: 7, ProjectID: 42, Name: "player", Description: "hero", Color: "#123456",
	}}}
	repository := &AssetRepositoryImpl{ProjectTagDao: projectTagDao}
	tag := &domain.ProjectTag{ProjectID: 42, Name: "player", Description: "hero", Color: "#123456"}

	if err := repository.CreateProjectTag(context.Background(), tag); err != nil {
		t.Fatalf("create project tag: %v", err)
	}
	if tag.ID != 7 || projectTagDao.created == nil || projectTagDao.created.ProjectID != 42 {
		t.Fatalf("unexpected created tag: domain=%+v dao=%+v", tag, projectTagDao.created)
	}

	got, err := repository.ListProjectTags(context.Background(), 42)
	if err != nil {
		t.Fatalf("list project tags: %v", err)
	}
	want := []domain.ProjectTag{{ID: 7, ProjectID: 42, Name: "player", Description: "hero", Color: "#123456"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected project tags: got=%+v want=%+v", got, want)
	}
}

func TestProjectTagRepositoryNormalizesPersistenceErrors(t *testing.T) {
	tests := []struct {
		input error
		want  error
	}{
		{input: dao.ErrProjectTagNotFound, want: domain.ErrProjectTagNotFound},
		{input: dao.ErrProjectTagConflict, want: domain.ErrProjectTagConflict},
		{input: dao.ErrProjectNotFound, want: domain.ErrProjectTagProjectNotFound},
	}
	for _, test := range tests {
		if got := normalizeProjectTagError(test.input); !errors.Is(got, test.want) {
			t.Fatalf("expected %v for %v, got %v", test.want, test.input, got)
		}
	}
}
