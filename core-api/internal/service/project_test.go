package service_test

import (
	"context"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

type projectRepositoryStub struct {
	update *domain.ProjectUpdate
}

func (*projectRepositoryStub) Insert(context.Context, *domain.Project) error { return nil }

func (*projectRepositoryStub) FindByID(context.Context, uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

func (*projectRepositoryStub) FindByUserID(context.Context, uint) ([]*domain.Project, error) {
	return []*domain.Project{}, nil
}

func (s *projectRepositoryStub) Update(_ context.Context, update *domain.ProjectUpdate) error {
	s.update = update
	return nil
}

func (*projectRepositoryStub) Remove(context.Context, uint) error { return nil }

func TestUpdateForwardsPartialProjectUpdate(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	update := &domain.ProjectUpdate{ID: 42, Reference: &reference}
	repository := &projectRepositoryStub{}
	projectService := service.NewProjectService(repository)

	if err := projectService.Update(context.Background(), update); err != nil {
		t.Fatalf("update project: %v", err)
	}
	if repository.update != update {
		t.Fatalf("expected update pointer %p, got %p", update, repository.update)
	}
}
