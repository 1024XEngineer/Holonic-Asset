package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	tagdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

type projectTagDaoStub struct {
	dao.ProjectTagDao
	created       *dao.ProjectTag
	createErr     error
	tags          []dao.ProjectTag
	listErr       error
	found         *dao.ProjectTag
	findProjectID uint
	findTagID     uint
	findErr       error
	updated       *dao.ProjectTag
	updateInput   *dao.ProjectTagUpdate
	updateProject uint
	updateTag     uint
	updateErr     error
	deleteProject uint
	deleteTag     uint
	deleteErr     error
}

func (s *projectTagDaoStub) Create(_ context.Context, tag *dao.ProjectTag) error {
	s.created = tag
	if s.createErr == nil {
		tag.ID = 7
	}
	return s.createErr
}

func (s *projectTagDaoStub) ListByProjectID(context.Context, uint) ([]dao.ProjectTag, error) {
	return s.tags, s.listErr
}

func (s *projectTagDaoStub) FindByID(_ context.Context, projectID, tagID uint) (*dao.ProjectTag, error) {
	s.findProjectID = projectID
	s.findTagID = tagID
	return s.found, s.findErr
}

func (s *projectTagDaoStub) Update(
	_ context.Context,
	projectID, tagID uint,
	update *dao.ProjectTagUpdate,
) (*dao.ProjectTag, error) {
	s.updateProject = projectID
	s.updateTag = tagID
	s.updateInput = update
	return s.updated, s.updateErr
}

func (s *projectTagDaoStub) Delete(_ context.Context, projectID, tagID uint) error {
	s.deleteProject = projectID
	s.deleteTag = tagID
	return s.deleteErr
}

func TestProjectTagRepositoryConvertsTagsAcrossDomainBoundary(t *testing.T) {
	projectTagDao := &projectTagDaoStub{tags: []dao.ProjectTag{{
		ID: 7, ProjectID: 42, Name: "player", Description: "hero", Color: "#123456",
	}}, found: &dao.ProjectTag{
		ID: 7, ProjectID: 42, Name: "player", Description: "hero", Color: "#123456",
	}, updated: &dao.ProjectTag{
		ID: 7, ProjectID: 42, Name: "hero", Description: "main character", Color: "#654321",
	}}
	repository := NewProjectTagRepository(projectTagDao)
	tag := &tagdomain.Tag{ProjectID: 42, Name: "player", Description: "hero", Color: "#123456"}

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
	want := []tagdomain.Tag{{ID: 7, ProjectID: 42, Name: "player", Description: "hero", Color: "#123456"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected project tags: got=%+v want=%+v", got, want)
	}

	detail, err := repository.GetProjectTag(context.Background(), 42, 7)
	if err != nil || !reflect.DeepEqual(*detail, want[0]) || projectTagDao.findProjectID != 42 || projectTagDao.findTagID != 7 {
		t.Fatalf("unexpected project tag detail: tag=%+v project=%d id=%d err=%v", detail, projectTagDao.findProjectID, projectTagDao.findTagID, err)
	}

	name := "hero"
	description := "main character"
	color := "#654321"
	updated, err := repository.UpdateProjectTag(context.Background(), 42, 7, &tagdomain.TagUpdate{
		Name: &name, Description: &description, Color: &color,
	})
	if err != nil || updated.Name != name || projectTagDao.updateInput == nil ||
		*projectTagDao.updateInput.Description != description || *projectTagDao.updateInput.Color != color {
		t.Fatalf("unexpected updated project tag: tag=%+v input=%+v err=%v", updated, projectTagDao.updateInput, err)
	}
	if projectTagDao.updateProject != 42 || projectTagDao.updateTag != 7 {
		t.Fatalf("unexpected update scope: project=%d tag=%d", projectTagDao.updateProject, projectTagDao.updateTag)
	}

	if err := repository.DeleteProjectTag(context.Background(), 42, 7); err != nil {
		t.Fatalf("delete project tag: %v", err)
	}
	if projectTagDao.deleteProject != 42 || projectTagDao.deleteTag != 7 {
		t.Fatalf("unexpected delete scope: project=%d tag=%d", projectTagDao.deleteProject, projectTagDao.deleteTag)
	}
}

func TestProjectTagRepositoryNormalizesPersistenceErrors(t *testing.T) {
	tests := []struct {
		input error
		want  error
	}{
		{input: dao.ErrProjectTagNotFound, want: tagdomain.ErrTagNotFound},
		{input: dao.ErrProjectTagConflict, want: tagdomain.ErrTagConflict},
		{input: dao.ErrProjectNotFound, want: tagdomain.ErrTagProjectNotFound},
	}
	for _, test := range tests {
		if got := normalizeProjectTagError(test.input); !errors.Is(got, test.want) {
			t.Fatalf("expected %v for %v, got %v", test.want, test.input, got)
		}
	}
}

func TestProjectTagRepositoryMapsErrorsFromEveryOperation(t *testing.T) {
	name := "hero"
	tests := []struct {
		name string
		run  func(*ProjectTagRepository) error
		stub *projectTagDaoStub
		want error
	}{
		{
			name: "create conflict",
			stub: &projectTagDaoStub{createErr: dao.ErrProjectTagConflict},
			run: func(repository *ProjectTagRepository) error {
				return repository.CreateProjectTag(context.Background(), &tagdomain.Tag{ProjectID: 42, Name: "hero"})
			},
			want: tagdomain.ErrTagConflict,
		},
		{
			name: "list missing project",
			stub: &projectTagDaoStub{listErr: dao.ErrProjectNotFound},
			run: func(repository *ProjectTagRepository) error {
				_, err := repository.ListProjectTags(context.Background(), 42)
				return err
			},
			want: tagdomain.ErrTagProjectNotFound,
		},
		{
			name: "get missing tag",
			stub: &projectTagDaoStub{findErr: dao.ErrProjectTagNotFound},
			run: func(repository *ProjectTagRepository) error {
				_, err := repository.GetProjectTag(context.Background(), 42, 7)
				return err
			},
			want: tagdomain.ErrTagNotFound,
		},
		{
			name: "update conflict",
			stub: &projectTagDaoStub{updateErr: dao.ErrProjectTagConflict},
			run: func(repository *ProjectTagRepository) error {
				_, err := repository.UpdateProjectTag(context.Background(), 42, 7, &tagdomain.TagUpdate{Name: &name})
				return err
			},
			want: tagdomain.ErrTagConflict,
		},
		{
			name: "delete missing tag",
			stub: &projectTagDaoStub{deleteErr: dao.ErrProjectTagNotFound},
			run: func(repository *ProjectTagRepository) error {
				return repository.DeleteProjectTag(context.Background(), 42, 7)
			},
			want: tagdomain.ErrTagNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := NewProjectTagRepository(test.stub)
			if err := test.run(repository); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestProjectTagRepositoryRequiresStorage(t *testing.T) {
	repository := &ProjectTagRepository{}
	tag := &tagdomain.Tag{ProjectID: 42, Name: "hero"}
	name := "player"
	operations := []func() error{
		func() error { return repository.CreateProjectTag(context.Background(), tag) },
		func() error {
			_, err := repository.ListProjectTags(context.Background(), 42)
			return err
		},
		func() error {
			_, err := repository.GetProjectTag(context.Background(), 42, 7)
			return err
		},
		func() error {
			_, err := repository.UpdateProjectTag(context.Background(), 42, 7, &tagdomain.TagUpdate{Name: &name})
			return err
		},
		func() error { return repository.DeleteProjectTag(context.Background(), 42, 7) },
	}
	for _, operation := range operations {
		if err := operation(); err == nil {
			t.Fatal("expected missing storage error")
		}
	}
}

func TestProjectTagRepositoryHandlesNilConversions(t *testing.T) {
	if got := convertProjectTagToDao(nil); got != nil {
		t.Fatalf("expected nil DAO tag, got %+v", got)
	}
	if got := convertProjectTagToDomain(nil); !reflect.DeepEqual(got, tagdomain.Tag{}) {
		t.Fatalf("expected empty domain tag, got %+v", got)
	}
	if got := convertProjectTagUpdateToDao(nil); got != nil {
		t.Fatalf("expected nil DAO update, got %+v", got)
	}
	unknown := errors.New("unknown")
	if got := normalizeProjectTagError(unknown); !errors.Is(got, unknown) {
		t.Fatalf("expected unknown error passthrough, got %v", got)
	}
}
