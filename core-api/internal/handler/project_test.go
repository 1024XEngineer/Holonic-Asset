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
	updateErr          error
	updateContext      context.Context
	update             *domain.ProjectUpdate
	updateCalls        int
	generateErr        error
	generate           *domain.Project
	generatedReference string
	generateCtx        context.Context
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

func (s *projectManagerStub) GenerateReference(ctx context.Context, project *domain.Project) (string, error) {
	s.generateCtx = ctx
	s.generate = project
	return s.generatedReference, s.generateErr
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
	if stub.update.Name != nil || stub.update.GameType != nil || stub.update.Perspective != nil || stub.update.Style != nil {
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

func TestGenerateReferenceForwardsProjectDefinitionAndReturnsReference(t *testing.T) {
	const generatedReference = "data:image/png;base64,aGVsbG8="
	stub := &projectManagerStub{generatedReference: generatedReference}
	projectHandler := handler.NewProjectHandler(stub)
	ctx := context.Background()

	request := dto.GenerateProjectReferenceRequest{
		Name:           "Prototype",
		GameType:       domain.GameTypeRPG,
		Perspective:    domain.PerspectiveTopDown,
		TargetPlatform: domain.PlatformTypePC,
		Description:    "A small village adventure",
		Reference:      "https://media.example/current-reference.png",
		Style:          "warm pixel art",
	}
	response, err := projectHandler.GenerateReference(ctx, request)
	if err != nil {
		t.Fatalf("generate project reference: %v", err)
	}
	if stub.generateCtx != ctx {
		t.Fatal("expected handler context to be forwarded to the manager")
	}
	if stub.generate == nil || stub.generate.Name != request.Name || stub.generate.Description != request.Description {
		t.Fatalf("unexpected project reference request: %+v", stub.generate)
	}
	if stub.generate.UserID != 0 {
		t.Fatalf("expected reference generation not to require a user ID, got %d", stub.generate.UserID)
	}
	if stub.generate.Reference != request.Reference {
		t.Fatalf("expected reference %q to be forwarded, got %q", request.Reference, stub.generate.Reference)
	}
	if response.Code != dto.SuccessCode || response.Message != dto.SuccessMessage {
		t.Fatalf("unexpected response envelope: %+v", response)
	}
	if response.Data.Reference != generatedReference {
		t.Fatalf("expected generated reference %q, got %q", generatedReference, response.Data.Reference)
	}
}

func TestGenerateReferencePropagatesServiceError(t *testing.T) {
	wantErr := errors.New("generate project failed")
	projectHandler := handler.NewProjectHandler(&projectManagerStub{generateErr: wantErr})

	response, err := projectHandler.GenerateReference(context.Background(), dto.GenerateProjectReferenceRequest{
		Name:           "Prototype",
		GameType:       domain.GameType(""),
		Perspective:    domain.PerspectiveTopDown,
		TargetPlatform: domain.PlatformTypePC,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if response != (dto.SuccessResponse[dto.GenerateProjectReferenceResponse]{}) {
		t.Fatalf("expected an empty response on error, got %+v", response)
	}
}
