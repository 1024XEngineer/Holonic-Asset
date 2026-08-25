package asset_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type projectTagStoreStub struct {
	domain.Store
	created      *domain.ProjectTag
	createErr    error
	listed       []domain.ProjectTag
	listProject  uint
	detail       *domain.ProjectTag
	updated      *domain.ProjectTag
	updateInput  *domain.ProjectTagUpdate
	deletedTagID uint
}

func (s *projectTagStoreStub) CreateProjectTag(_ context.Context, tag *domain.ProjectTag) error {
	s.created = tag
	if s.createErr == nil {
		tag.ID = 7
	}
	return s.createErr
}

func (s *projectTagStoreStub) ListProjectTags(_ context.Context, projectID uint) ([]domain.ProjectTag, error) {
	s.listProject = projectID
	return s.listed, nil
}

func (s *projectTagStoreStub) GetProjectTag(_ context.Context, _, _ uint) (*domain.ProjectTag, error) {
	return s.detail, nil
}

func (s *projectTagStoreStub) UpdateProjectTag(
	_ context.Context,
	_, _ uint,
	update *domain.ProjectTagUpdate,
) (*domain.ProjectTag, error) {
	s.updateInput = update
	return s.updated, nil
}

func (s *projectTagStoreStub) DeleteProjectTag(_ context.Context, _ uint, tagID uint) error {
	s.deletedTagID = tagID
	return nil
}

func TestProjectTagManagerCreatesNormalizedTagWithDefaultColor(t *testing.T) {
	store := &projectTagStoreStub{}
	manager := domain.NewManager(store)

	got, err := manager.CreateProjectTag(context.Background(), domain.ProjectTag{
		ProjectID:   42,
		Name:        "  Player  ",
		Description: "  Controllable hero  ",
	})
	if err != nil {
		t.Fatalf("create project tag: %v", err)
	}
	if got.ID != 7 || got.Name != "Player" || got.Description != "Controllable hero" || got.Color != domain.DefaultTagColor {
		t.Fatalf("unexpected project tag: %+v", got)
	}
	if store.created == nil || !reflect.DeepEqual(*store.created, got) {
		t.Fatalf("unexpected persisted project tag: %+v", store.created)
	}
}

func TestProjectTagManagerRejectsInvalidTagsBeforePersistence(t *testing.T) {
	tests := []domain.ProjectTag{
		{ProjectID: 0, Name: "player", Color: "#123456"},
		{ProjectID: 42, Name: "   ", Color: "#123456"},
		{ProjectID: 42, Name: "player", Color: "123456"},
	}
	for _, input := range tests {
		store := &projectTagStoreStub{}
		_, err := domain.NewManager(store).CreateProjectTag(context.Background(), input)
		if !errors.Is(err, domain.ErrInvalidProjectTag) {
			t.Fatalf("expected invalid tag error for %+v, got %v", input, err)
		}
		if store.created != nil {
			t.Fatalf("invalid tag reached persistence: %+v", input)
		}
	}
}

func TestProjectTagManagerNormalizesPartialUpdate(t *testing.T) {
	name := "  Environment  "
	description := "  World assets  "
	store := &projectTagStoreStub{updated: &domain.ProjectTag{
		ID: 9, ProjectID: 42, Name: "Environment", Description: "World assets", Color: "#16A34A",
	}}

	got, err := domain.NewManager(store).UpdateProjectTag(context.Background(), 42, 9, &domain.ProjectTagUpdate{
		Name: &name, Description: &description,
	})
	if err != nil {
		t.Fatalf("update project tag: %v", err)
	}
	if got.ID != 9 || store.updateInput == nil || *store.updateInput.Name != "Environment" || *store.updateInput.Description != "World assets" {
		t.Fatalf("unexpected update: result=%+v input=%+v", got, store.updateInput)
	}
}

func TestProjectTagManagerRequiresScopedIDsAndChanges(t *testing.T) {
	manager := domain.NewManager(&projectTagStoreStub{})
	if _, err := manager.ListProjectTags(context.Background(), 0); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid project ID, got %v", err)
	}
	if _, err := manager.GetProjectTag(context.Background(), 42, 0); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid tag ID, got %v", err)
	}
	if _, err := manager.UpdateProjectTag(context.Background(), 42, 9, &domain.ProjectTagUpdate{}); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected empty update to fail, got %v", err)
	}
}
