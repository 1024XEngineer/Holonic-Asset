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
	created        domain.ProjectTag
	createResult   domain.ProjectTag
	createErr      error
	listed         []domain.ProjectTag
	listedProject  uint
	listErr        error
	detailProject  uint
	detailID       uint
	detailErr      error
	updated        *domain.ProjectTagUpdate
	updatedProject uint
	updatedID      uint
	updateErr      error
	deletedProject uint
	deletedID      uint
	deleteErr      error
}

func (s *projectTagManagerStub) CreateProjectTag(_ context.Context, tag domain.ProjectTag) (domain.ProjectTag, error) {
	s.created = tag
	return s.createResult, s.createErr
}

func (s *projectTagManagerStub) ListProjectTags(_ context.Context, projectID uint) ([]domain.ProjectTag, error) {
	s.listedProject = projectID
	return s.listed, s.listErr
}

func (s *projectTagManagerStub) GetProjectTag(_ context.Context, projectID, tagID uint) (domain.ProjectTag, error) {
	s.detailProject = projectID
	s.detailID = tagID
	return domain.ProjectTag{ID: tagID, ProjectID: projectID, Name: "player", Color: "#123456"}, s.detailErr
}

func (s *projectTagManagerStub) UpdateProjectTag(
	_ context.Context,
	projectID uint,
	tagID uint,
	update *domain.ProjectTagUpdate,
) (domain.ProjectTag, error) {
	s.updatedProject = projectID
	s.updatedID = tagID
	s.updated = update
	return domain.ProjectTag{ID: tagID, ProjectID: projectID, Name: "hero", Color: "#654321"}, s.updateErr
}

func (s *projectTagManagerStub) DeleteProjectTag(_ context.Context, projectID uint, tagID uint) error {
	s.deletedProject = projectID
	s.deletedID = tagID
	return s.deleteErr
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
	if manager.listedProject != 42 {
		t.Fatalf("unexpected list project: %d", manager.listedProject)
	}

	detail, err := h.GetProjectTag(context.Background(), dto.ProjectTagDetailRequest{ProjectID: 42, TagID: 7})
	if err != nil || detail.Data.Tag.TagID != 7 || manager.detailProject != 42 || manager.detailID != 7 {
		t.Fatalf("get project tag: response=%+v project=%d id=%d err=%v", detail, manager.detailProject, manager.detailID, err)
	}

	name := "hero"
	updated, err := h.UpdateProjectTag(context.Background(), dto.UpdateProjectTagRequest{
		ProjectID: 42, TagID: 7, Name: &name,
	})
	if err != nil || updated.Data.Tag.Name != "hero" || manager.updated == nil || *manager.updated.Name != name {
		t.Fatalf("update project tag: response=%+v input=%+v err=%v", updated, manager.updated, err)
	}
	if manager.updatedProject != 42 || manager.updatedID != 7 {
		t.Fatalf("unexpected update scope: project=%d tag=%d", manager.updatedProject, manager.updatedID)
	}

	deleted, err := h.DeleteProjectTag(context.Background(), dto.DeleteProjectTagRequest{ProjectID: 42, TagID: 7})
	if err != nil || !deleted.Data.Success || manager.deletedProject != 42 || manager.deletedID != 7 {
		t.Fatalf("delete project tag: response=%+v project=%d deleted=%d err=%v", deleted, manager.deletedProject, manager.deletedID, err)
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

func TestProjectTagHandlerPropagatesManagerErrorsAcrossCRUD(t *testing.T) {
	managerErr := errors.New("manager failed")
	name := "hero"
	tests := []struct {
		name    string
		manager *projectTagManagerStub
		run     func(*handler.Handler) error
	}{
		{
			name:    "list",
			manager: &projectTagManagerStub{listErr: managerErr},
			run: func(h *handler.Handler) error {
				_, err := h.ListProjectTags(context.Background(), dto.ListProjectTagsRequest{ProjectID: 42})
				return err
			},
		},
		{
			name:    "detail",
			manager: &projectTagManagerStub{detailErr: managerErr},
			run: func(h *handler.Handler) error {
				_, err := h.GetProjectTag(context.Background(), dto.ProjectTagDetailRequest{ProjectID: 42, TagID: 7})
				return err
			},
		},
		{
			name:    "update",
			manager: &projectTagManagerStub{updateErr: managerErr},
			run: func(h *handler.Handler) error {
				_, err := h.UpdateProjectTag(context.Background(), dto.UpdateProjectTagRequest{ProjectID: 42, TagID: 7, Name: &name})
				return err
			},
		},
		{
			name:    "delete",
			manager: &projectTagManagerStub{deleteErr: managerErr},
			run: func(h *handler.Handler) error {
				_, err := h.DeleteProjectTag(context.Background(), dto.DeleteProjectTagRequest{ProjectID: 42, TagID: 7})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(handler.NewHandler(test.manager)); !errors.Is(err, managerErr) {
				t.Fatalf("expected manager error, got %v", err)
			}
		})
	}
}
