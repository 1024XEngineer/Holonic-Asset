package tag_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
)

type tagStoreStub struct {
	tag.Store
	created        *tag.Tag
	createErr      error
	listed         []tag.Tag
	listProject    uint
	listErr        error
	detail         *tag.Tag
	detailProject  uint
	detailTag      uint
	detailErr      error
	updated        *tag.Tag
	updateInput    *tag.TagUpdate
	updateProject  uint
	updateTag      uint
	updateErr      error
	deletedProject uint
	deletedTagID   uint
	deleteErr      error
}

func (s *tagStoreStub) CreateProjectTag(_ context.Context, t *tag.Tag) error {
	s.created = t
	if s.createErr == nil {
		t.ID = 7
	}
	return s.createErr
}

func (s *tagStoreStub) ListProjectTags(_ context.Context, projectID uint) ([]tag.Tag, error) {
	s.listProject = projectID
	return s.listed, s.listErr
}

func (s *tagStoreStub) GetProjectTag(_ context.Context, projectID, tagID uint) (*tag.Tag, error) {
	s.detailProject = projectID
	s.detailTag = tagID
	return s.detail, s.detailErr
}

func (s *tagStoreStub) UpdateProjectTag(
	_ context.Context,
	projectID, tagID uint,
	update *tag.TagUpdate,
) (*tag.Tag, error) {
	s.updateProject = projectID
	s.updateTag = tagID
	s.updateInput = update
	return s.updated, s.updateErr
}

func (s *tagStoreStub) DeleteProjectTag(_ context.Context, projectID uint, tagID uint) error {
	s.deletedProject = projectID
	s.deletedTagID = tagID
	return s.deleteErr
}

func TestTagManagerCreatesNormalizedTagWithDefaultColor(t *testing.T) {
	store := &tagStoreStub{}
	manager := tag.NewManager(store)

	got, err := manager.CreateProjectTag(context.Background(), tag.Tag{
		ProjectID:   42,
		Name:        "  Player  ",
		Description: "  Controllable hero  ",
	})
	if err != nil {
		t.Fatalf("create project tag: %v", err)
	}
	if got.ID != 7 || got.Name != "Player" || got.Description != "Controllable hero" || got.Color != tag.DefaultTagColor {
		t.Fatalf("unexpected project tag: %+v", got)
	}
	if store.created == nil || !reflect.DeepEqual(*store.created, got) {
		t.Fatalf("unexpected persisted project tag: %+v", store.created)
	}
}

func TestTagManagerRejectsInvalidTagsBeforePersistence(t *testing.T) {
	tests := []tag.Tag{
		{ProjectID: 0, Name: "player", Color: "#123456"},
		{ProjectID: 42, Name: "   ", Color: "#123456"},
		{ProjectID: 42, Name: "player", Color: "123456"},
	}
	for _, input := range tests {
		store := &tagStoreStub{}
		_, err := tag.NewManager(store).CreateProjectTag(context.Background(), input)
		if !errors.Is(err, tag.ErrInvalidTag) {
			t.Fatalf("expected invalid tag error for %+v, got %v", input, err)
		}
		if store.created != nil {
			t.Fatalf("invalid tag reached persistence: %+v", input)
		}
	}
}

func TestTagManagerNormalizesPartialUpdate(t *testing.T) {
	name := "  Environment  "
	description := "  World assets  "
	color := "  #16A34A  "
	store := &tagStoreStub{updated: &tag.Tag{
		ID: 9, ProjectID: 42, Name: "Environment", Description: "World assets", Color: "#16A34A",
	}}

	got, err := tag.NewManager(store).UpdateProjectTag(context.Background(), 42, 9, &tag.TagUpdate{
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

func TestTagManagerCompletesScopedLifecycle(t *testing.T) {
	store := &tagStoreStub{
		listed: []tag.Tag{{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"}},
		detail: &tag.Tag{ID: 7, ProjectID: 42, Name: "player", Color: "#123456"},
	}
	manager := tag.NewManager(store)

	tags, err := manager.ListProjectTags(context.Background(), 42)
	if err != nil || len(tags) != 1 || store.listProject != 42 {
		t.Fatalf("list project tags: tags=%+v project=%d err=%v", tags, store.listProject, err)
	}
	tDetail, err := manager.GetProjectTag(context.Background(), 42, 7)
	if err != nil || tDetail.ID != 7 || store.detailProject != 42 || store.detailTag != 7 {
		t.Fatalf("get project tag: tag=%+v project=%d id=%d err=%v", tDetail, store.detailProject, store.detailTag, err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 42, 7); err != nil {
		t.Fatalf("delete project tag: %v", err)
	}
	if store.deletedProject != 42 || store.deletedTagID != 7 {
		t.Fatalf("unexpected delete scope: project=%d tag=%d", store.deletedProject, store.deletedTagID)
	}
}

func TestTagManagerPropagatesStoreErrorsAndMissingResults(t *testing.T) {
	storeErr := errors.New("store failed")
	name := "player"
	tests := []struct {
		name  string
		run   func(tag.Manager) error
		store *tagStoreStub
		want  error
	}{
		{
			name:  "create error",
			store: &tagStoreStub{createErr: storeErr},
			run: func(manager tag.Manager) error {
				_, err := manager.CreateProjectTag(context.Background(), tag.Tag{ProjectID: 42, Name: name})
				return err
			},
			want: storeErr,
		},
		{
			name:  "list error",
			store: &tagStoreStub{listErr: storeErr},
			run: func(manager tag.Manager) error {
				_, err := manager.ListProjectTags(context.Background(), 42)
				return err
			},
			want: storeErr,
		},
		{
			name:  "get error",
			store: &tagStoreStub{detailErr: storeErr},
			run: func(manager tag.Manager) error {
				_, err := manager.GetProjectTag(context.Background(), 42, 7)
				return err
			},
			want: storeErr,
		},
		{
			name:  "get missing result",
			store: &tagStoreStub{},
			run: func(manager tag.Manager) error {
				_, err := manager.GetProjectTag(context.Background(), 42, 7)
				return err
			},
			want: tag.ErrTagNotFound,
		},
		{
			name:  "update error",
			store: &tagStoreStub{updateErr: storeErr},
			run: func(manager tag.Manager) error {
				_, err := manager.UpdateProjectTag(context.Background(), 42, 7, &tag.TagUpdate{Name: &name})
				return err
			},
			want: storeErr,
		},
		{
			name:  "update missing result",
			store: &tagStoreStub{},
			run: func(manager tag.Manager) error {
				_, err := manager.UpdateProjectTag(context.Background(), 42, 7, &tag.TagUpdate{Name: &name})
				return err
			},
			want: tag.ErrTagNotFound,
		},
		{
			name:  "delete error",
			store: &tagStoreStub{deleteErr: storeErr},
			run: func(manager tag.Manager) error {
				return manager.DeleteProjectTag(context.Background(), 42, 7)
			},
			want: storeErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(tag.NewManager(test.store)); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestTagManagerRejectsInvalidTextAndUpdates(t *testing.T) {
	longName := strings.Repeat("n", 101)
	longDescription := strings.Repeat("d", 256)
	invalidColor := "blue"
	invalidNames := []string{longName, "player\nadmin"}
	for _, name := range invalidNames {
		_, err := tag.NewManager(&tagStoreStub{}).CreateProjectTag(context.Background(), tag.Tag{
			ProjectID: 42, Name: name,
		})
		if !errors.Is(err, tag.ErrInvalidTag) {
			t.Fatalf("expected invalid name error for %q, got %v", name, err)
		}
	}
	_, err := tag.NewManager(&tagStoreStub{}).CreateProjectTag(context.Background(), tag.Tag{
		ProjectID: 42, Name: "player", Description: longDescription,
	})
	if !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid description error, got %v", err)
	}

	updates := []*tag.TagUpdate{
		nil,
		{Name: &longName},
		{Description: &longDescription},
		{Color: &invalidColor},
	}
	for _, update := range updates {
		_, err := tag.NewManager(&tagStoreStub{}).UpdateProjectTag(context.Background(), 42, 7, update)
		if !errors.Is(err, tag.ErrInvalidTag) {
			t.Fatalf("expected invalid update error for %+v, got %v", update, err)
		}
	}
}

func TestTagManagerRequiresScopedIDsAndChanges(t *testing.T) {
	manager := tag.NewManager(&tagStoreStub{})
	name := "player"
	if _, err := manager.ListProjectTags(context.Background(), 0); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid project ID, got %v", err)
	}
	if _, err := manager.GetProjectTag(context.Background(), 0, 9); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid get project ID, got %v", err)
	}
	if _, err := manager.GetProjectTag(context.Background(), 42, 0); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid tag ID, got %v", err)
	}
	if _, err := manager.UpdateProjectTag(context.Background(), 0, 9, &tag.TagUpdate{Name: &name}); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid update project ID, got %v", err)
	}
	if _, err := manager.UpdateProjectTag(context.Background(), 42, 9, &tag.TagUpdate{}); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected empty update to fail, got %v", err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 0, 9); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid delete project ID, got %v", err)
	}
	if err := manager.DeleteProjectTag(context.Background(), 42, 0); !errors.Is(err, tag.ErrInvalidTag) {
		t.Fatalf("expected invalid delete tag ID, got %v", err)
	}
}
