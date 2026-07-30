package project_test

import (
	"context"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type projectStoreStub struct {
	update *domain.ProjectUpdate
}

func (*projectStoreStub) Insert(context.Context, *domain.Project) error { return nil }

func (*projectStoreStub) FindByID(context.Context, uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

func (*projectStoreStub) FindByUserID(context.Context, uint) ([]*domain.Project, error) {
	return []*domain.Project{}, nil
}

func (s *projectStoreStub) Update(_ context.Context, update *domain.ProjectUpdate) error {
	s.update = update
	return nil
}

func (*projectStoreStub) Remove(context.Context, uint) error { return nil }

func TestUpdateForwardsPartialProjectUpdate(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	update := &domain.ProjectUpdate{ID: 42, Reference: &reference}
	store := &projectStoreStub{}
	manager := domain.NewManager(store)

	if err := manager.Update(context.Background(), update); err != nil {
		t.Fatalf("update project: %v", err)
	}
	if store.update != update {
		t.Fatalf("expected update pointer %p, got %p", update, store.update)
	}
}
