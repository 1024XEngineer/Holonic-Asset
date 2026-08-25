package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type projectTagManagerStub struct {
	domain.Manager
	created      domain.ProjectTag
	createResult domain.ProjectTag
	createErr    error
	listed       []domain.ProjectTag
	updated      *domain.ProjectTagUpdate
	deletedID    uint
}

func (s *projectTagManagerStub) CreateProjectTag(_ context.Context, tag domain.ProjectTag) (domain.ProjectTag, error) {
	s.created = tag
	return s.createResult, s.createErr
}

func (s *projectTagManagerStub) ListProjectTags(context.Context, uint) ([]domain.ProjectTag, error) {
	return s.listed, nil
}

func (s *projectTagManagerStub) GetProjectTag(_ context.Context, projectID, tagID uint) (domain.ProjectTag, error) {
	return domain.ProjectTag{ID: tagID, ProjectID: projectID, Name: "player", Color: "#123456"}, nil
}

func (s *projectTagManagerStub) UpdateProjectTag(
	_ context.Context,
	projectID uint,
	tagID uint,
	update *domain.ProjectTagUpdate,
) (domain.ProjectTag, error) {
	s.updated = update
	return domain.ProjectTag{ID: tagID, ProjectID: projectID, Name: "hero", Color: "#654321"}, nil
}

func (s *projectTagManagerStub) DeleteProjectTag(_ context.Context, _ uint, tagID uint) error {
	s.deletedID = tagID
	return nil
}

func TestProjectTagHandlerMapsCRUDResponses(t *testing.T) {
	manager := &projectTagManagerStub{
		createResult: domain.ProjectTag{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"},
		listed:       []domain.ProjectTag{{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"}},
	}
	h := handler.NewHandler(manager)

	created, err := h.CreateProjectTag(context.Background(), dto.CreateProjectTagRequest{
		ProjectID: 42, Name: "player", Color: "#123456",
	})
	if err != nil || created.Data.Tag.TagID != 7 || manager.created.ProjectID != 42 {
		t.Fatalf("create project tag: response=%+v input=%+v err=%v", created, manager.created, err)
	}

	listed, err := h.ListProjectTags(context.Background(), dto.ListProjectTagsRequest{ProjectID: 42})
	if err != nil || len(listed.Data.Tags) != 1 || listed.Data.Tags[0].TagID != 7 {
		t.Fatalf("list project tags: response=%+v err=%v", listed, err)
	}

	name := "hero"
	updated, err := h.UpdateProjectTag(context.Background(), dto.UpdateProjectTagRequest{
		ProjectID: 42, TagID: 7, Name: &name,
	})
	if err != nil || updated.Data.Tag.Name != "hero" || manager.updated == nil || *manager.updated.Name != name {
		t.Fatalf("update project tag: response=%+v input=%+v err=%v", updated, manager.updated, err)
	}

	deleted, err := h.DeleteProjectTag(context.Background(), dto.DeleteProjectTagRequest{ProjectID: 42, TagID: 7})
	if err != nil || !deleted.Data.Success || manager.deletedID != 7 {
		t.Fatalf("delete project tag: response=%+v deleted=%d err=%v", deleted, manager.deletedID, err)
	}
}

func TestProjectTagHandlerMapsDomainErrors(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: domain.ErrInvalidProjectTag, status: http.StatusBadRequest},
		{err: domain.ErrProjectTagNotFound, status: http.StatusNotFound},
		{err: domain.ErrProjectTagProjectNotFound, status: http.StatusNotFound},
		{err: domain.ErrProjectTagConflict, status: http.StatusConflict},
	}
	for _, test := range tests {
		h := handler.NewHandler(&projectTagManagerStub{createErr: test.err})
		_, err := h.CreateProjectTag(context.Background(), dto.CreateProjectTagRequest{ProjectID: 42, Name: "player"})
		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != test.status {
			t.Fatalf("expected HTTP %d for %v, got %v", test.status, test.err, err)
		}
	}
}
