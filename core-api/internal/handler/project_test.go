package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type projectManagerStub struct {
	updateErr     error
	updateContext context.Context
	update        *domain.ProjectUpdate
	updateCalls   int
	generateErr   error
	generate      *domain.Project
	generated     *domain.Project
	generateCtx   context.Context
}

func (*projectManagerStub) Create(context.Context, *domain.Project) error { return nil }

func (*projectManagerStub) ListByUID(context.Context, uint) ([]*domain.Project, error) {
	return []*domain.Project{}, nil
}

func (*projectManagerStub) GetDetail(context.Context, uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

func (s *projectManagerStub) Update(ctx context.Context, update *domain.ProjectUpdate) error {
	s.updateCalls++
	s.updateContext = ctx
	s.update = update
	return s.updateErr
}

func (*projectManagerStub) Delete(context.Context, uint) error { return nil }

func (s *projectManagerStub) GenerateVisualImage(ctx context.Context, project *domain.Project) (*domain.Project, error) {
	s.generateCtx = ctx
	s.generate = project
	if s.generated != nil {
		return s.generated, s.generateErr
	}
	return project, s.generateErr
}

func TestUpdateForwardsOnlyProvidedFields(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	description := "updated description"
	request := dto.UpdateProjectRequest{
		ProjectID:   42,
		Description: &description,
		Reference:   &reference,
	}
	stub := &projectManagerStub{}
	projectHandler := handler.NewProjectHandler(stub)
	handlerContext := context.Background()

	response, err := projectHandler.Update(handlerContext, request)
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if stub.updateCalls != 1 {
		t.Fatalf("expected one manager call, got %d", stub.updateCalls)
	}
	if stub.updateContext != handlerContext {
		t.Fatal("expected handler context to be forwarded to the manager")
	}
	if stub.update == nil || stub.update.ID != request.ProjectID {
		t.Fatalf("expected project ID %d, got %+v", request.ProjectID, stub.update)
	}
	if stub.update.Description == nil || *stub.update.Description != description {
		t.Fatalf("expected description %q, got %+v", description, stub.update.Description)
	}
	if stub.update.Reference == nil || *stub.update.Reference != reference {
		t.Fatalf("expected reference %q, got %+v", reference, stub.update.Reference)
	}
	if stub.update.Name != nil || stub.update.GameType != nil || stub.update.ViewType != nil || stub.update.Style != nil {
		t.Fatalf("expected omitted fields to remain nil, got %+v", stub.update)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response envelope: %+v", response)
	}
	if !response.Data.Success {
		t.Fatal("expected successful update response")
	}
}

func TestUpdatePropagatesServiceError(t *testing.T) {
	wantErr := errors.New("update project failed")
	projectHandler := handler.NewProjectHandler(&projectManagerStub{updateErr: wantErr})

	response, err := projectHandler.Update(context.Background(), dto.UpdateProjectRequest{ProjectID: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if response != (dto.SuccessResponse[dto.UpdateProjectResponse]{}) {
		t.Fatalf("expected an empty response on error, got %+v", response)
	}
}

func TestGenerateForwardsProjectAndReturnsGeneratedProject(t *testing.T) {
	generated := &domain.Project{
		UserID:         7,
		ID:             42,
		Name:           "Prototype",
		GameType:       domain.GameTypeRPG,
		ViewType:       domain.ViewTypeTopDown,
		TargetPlatform: domain.PlatformTypePC,
		Description:    "A small village adventure",
		Reference:      "aGVsbG8=",
		Style:          "warm pixel art",
	}
	stub := &projectManagerStub{generated: generated}
	projectHandler := handler.NewProjectHandler(stub)
	ctx := context.Background()

	response, err := projectHandler.Generate(ctx, dto.CreateProjectRequest{
		UserID:         generated.UserID,
		Name:           generated.Name,
		GameType:       generated.GameType,
		ViewType:       generated.ViewType,
		TargetPlatform: generated.TargetPlatform,
		Description:    generated.Description,
		Reference:      "optional-reference",
		Style:          generated.Style,
	})
	if err != nil {
		t.Fatalf("generate project: %v", err)
	}
	if stub.generateCtx != ctx {
		t.Fatal("expected handler context to be forwarded to the manager")
	}
	if stub.generate == nil || stub.generate.UserID != generated.UserID || stub.generate.Name != generated.Name {
		t.Fatalf("unexpected generated project request: %+v", stub.generate)
	}
	if stub.generate.Reference != "optional-reference" {
		t.Fatalf("expected reference to be forwarded, got %q", stub.generate.Reference)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response envelope: %+v", response)
	}
	if response.Data.Reference != generated.Reference {
		t.Fatalf("expected generated base64 %q, got %q", generated.Reference, response.Data.Reference)
	}
	if response.Data.ID != generated.ID || response.Data.Name != generated.Name {
		t.Fatalf("unexpected generated project response: %+v", response.Data)
	}
}

func TestGeneratePropagatesServiceError(t *testing.T) {
	wantErr := errors.New("generate project failed")
	projectHandler := handler.NewProjectHandler(&projectManagerStub{generateErr: wantErr})

	response, err := projectHandler.Generate(context.Background(), dto.CreateProjectRequest{
		UserID:         7,
		Name:           "Prototype",
		GameType:       domain.GameTypeOther,
		ViewType:       domain.ViewTypeTopDown,
		TargetPlatform: domain.PlatformTypePC,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if response != (dto.SuccessResponse[dto.ProjectResponse]{}) {
		t.Fatalf("expected an empty response on error, got %+v", response)
	}
}
