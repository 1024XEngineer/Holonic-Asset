package asset_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type projectTagStoreStub struct {
	domain.Store
	created        *domain.ProjectTag
	createErr      error
	listed         []domain.ProjectTag
	listProject    uint
	listErr        error
	detail         *domain.ProjectTag
	detailProject  uint
	detailTag      uint
	detailErr      error
	updated        *domain.ProjectTag
	updateInput    *domain.ProjectTagUpdate
	updateProject  uint
	updateTag      uint
	updateErr      error
	deletedProject uint
	deletedTagID   uint
	deleteErr      error
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
	return s.listed, s.listErr
}

func (s *projectTagStoreStub) GetProjectTag(_ context.Context, projectID, tagID uint) (*domain.ProjectTag, error) {
	s.detailProject = projectID
	s.detailTag = tagID
	return s.detail, s.detailErr
}

func (s *projectTagStoreStub) UpdateProjectTag(
	_ context.Context,
	projectID, tagID uint,
	update *domain.ProjectTagUpdate,
) (*domain.ProjectTag, error) {
	s.updateProject = projectID
	s.updateTag = tagID
	s.updateInput = update
	return s.updated, s.updateErr
}

func (s *projectTagStoreStub) DeleteProjectTag(_ context.Context, projectID uint, tagID uint) error {
	s.deletedProject = projectID
	s.deletedTagID = tagID
	return s.deleteErr
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
	color := "  #16A34A  "
	store := &projectTagStoreStub{updated: &domain.ProjectTag{
		ID: 9, ProjectID: 42, Name: "Environment", Description: "World assets", Color: "#16A34A",
	}}

	got, err := domain.NewManager(store).UpdateProjectTag(context.Background(), 42, 9, &domain.ProjectTagUpdate{
		Name: &name, Description: &description, Color: &color,
	})
	if err != nil {
		t.Fatalf("update project tag: %v", err)
	}
	if got.ID != 9 || store.updateInput == nil || *store.updateInput.Name != "Environment" ||
		*store.updateInput.Description != "World assets" || *store.updateInput.Color != "#16A34A" {
		t.Fatalf("unexpected update: result=%+v input=%+v", got, store.updateInput)
	}
}

func TestProjectTagManagerCompletesScopedLifecycle(t *testing.T) {
	store := &projectTagStoreStub{
		listed: []domain.ProjectTag{{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"}},
		detail: &domain.ProjectTag{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"},
	}
	manager := domain.NewManager(store)

	tags, err := manager.ListProjectTags(context.Background(), 42)
	if err != nil || len(tags) != 1 || store.listProject != 42 {
		t.Fatalf("list project tags: tags=%+v project=%d err=%v", tags, store.listProject, err)
	}
	tag, err := manager.GetProjectTag(context.Background(), 42, 7)
	if err != nil || tag.ID != 7 || store.detailProject != 42 || store.detailTag != 7 {
		t.Fatalf("get project tag: tag=%+v project=%d id=%d err=%v", tag, store.detailProject, store.detailTag, err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 42, 7); err != nil {
		t.Fatalf("delete project tag: %v", err)
	}
	if store.deletedProject != 42 || store.deletedTagID != 7 {
		t.Fatalf("unexpected delete scope: project=%d tag=%d", store.deletedProject, store.deletedTagID)
	}
}

func TestProjectTagManagerPropagatesStoreErrorsAndMissingResults(t *testing.T) {
	storeErr := errors.New("store failed")
	name := "player"
	tests := []struct {
		name  string
		run   func(domain.Manager) error
		store *projectTagStoreStub
		want  error
	}{
		{
			name:  "create error",
			store: &projectTagStoreStub{createErr: storeErr},
			run: func(manager domain.Manager) error {
				_, err := manager.CreateProjectTag(context.Background(), domain.ProjectTag{ProjectID: 42, Name: name})
				return err
			},
			want: storeErr,
		},
		{
			name:  "list error",
			store: &projectTagStoreStub{listErr: storeErr},
			run: func(manager domain.Manager) error {
				_, err := manager.ListProjectTags(context.Background(), 42)
				return err
			},
			want: storeErr,
		},
		{
			name:  "get error",
			store: &projectTagStoreStub{detailErr: storeErr},
			run: func(manager domain.Manager) error {
				_, err := manager.GetProjectTag(context.Background(), 42, 7)
				return err
			},
			want: storeErr,
		},
		{
			name:  "get missing result",
			store: &projectTagStoreStub{},
			run: func(manager domain.Manager) error {
				_, err := manager.GetProjectTag(context.Background(), 42, 7)
				return err
			},
			want: domain.ErrProjectTagNotFound,
		},
		{
			name:  "update error",
			store: &projectTagStoreStub{updateErr: storeErr},
			run: func(manager domain.Manager) error {
				_, err := manager.UpdateProjectTag(context.Background(), 42, 7, &domain.ProjectTagUpdate{Name: &name})
				return err
			},
			want: storeErr,
		},
		{
			name:  "update missing result",
			store: &projectTagStoreStub{},
			run: func(manager domain.Manager) error {
				_, err := manager.UpdateProjectTag(context.Background(), 42, 7, &domain.ProjectTagUpdate{Name: &name})
				return err
			},
			want: domain.ErrProjectTagNotFound,
		},
		{
			name:  "delete error",
			store: &projectTagStoreStub{deleteErr: storeErr},
			run: func(manager domain.Manager) error {
				return manager.DeleteProjectTag(context.Background(), 42, 7)
			},
			want: storeErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(domain.NewManager(test.store)); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestProjectTagManagerRejectsInvalidTextAndUpdates(t *testing.T) {
	longName := strings.Repeat("n", 101)
	longDescription := strings.Repeat("d", 256)
	invalidColor := "blue"
	invalidNames := []string{longName, "player\nadmin"}
	for _, name := range invalidNames {
		_, err := domain.NewManager(&projectTagStoreStub{}).CreateProjectTag(context.Background(), domain.ProjectTag{
			ProjectID: 42, Name: name,
		})
		if !errors.Is(err, domain.ErrInvalidProjectTag) {
			t.Fatalf("expected invalid name error for %q, got %v", name, err)
		}
	}
	_, err := domain.NewManager(&projectTagStoreStub{}).CreateProjectTag(context.Background(), domain.ProjectTag{
		ProjectID: 42, Name: "player", Description: longDescription,
	})
	if !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid description error, got %v", err)
	}

	updates := []*domain.ProjectTagUpdate{
		nil,
		{Name: &longName},
		{Description: &longDescription},
		{Color: &invalidColor},
	}
	for _, update := range updates {
		_, err := domain.NewManager(&projectTagStoreStub{}).UpdateProjectTag(context.Background(), 42, 7, update)
		if !errors.Is(err, domain.ErrInvalidProjectTag) {
			t.Fatalf("expected invalid update error for %+v, got %v", update, err)
		}
	}
}

func TestProjectTagManagerRequiresScopedIDsAndChanges(t *testing.T) {
	manager := domain.NewManager(&projectTagStoreStub{})
	name := "player"
	if _, err := manager.ListProjectTags(context.Background(), 0); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid project ID, got %v", err)
	}
	if _, err := manager.GetProjectTag(context.Background(), 0, 9); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid get project ID, got %v", err)
	}
	if _, err := manager.GetProjectTag(context.Background(), 42, 0); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid tag ID, got %v", err)
	}
	if _, err := manager.UpdateProjectTag(context.Background(), 0, 9, &domain.ProjectTagUpdate{Name: &name}); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid update project ID, got %v", err)
	}
	if _, err := manager.UpdateProjectTag(context.Background(), 42, 9, &domain.ProjectTagUpdate{}); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected empty update to fail, got %v", err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 0, 9); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid delete project ID, got %v", err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 42, 0); !errors.Is(err, domain.ErrInvalidProjectTag) {
		t.Fatalf("expected invalid delete tag ID, got %v", err)
	}
}
