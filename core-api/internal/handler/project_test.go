package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/project"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type projectServiceStub struct {
	updateErr     error
	updateContext context.Context
	update        *domain.ProjectUpdate
	updateCalls   int
}

func (*projectServiceStub) Create(context.Context, *domain.Project) error { return nil }

func (*projectServiceStub) ListByUID(context.Context, uint) ([]*domain.Project, error) {
	return []*domain.Project{}, nil
}

func (*projectServiceStub) GetDetail(context.Context, uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

func (s *projectServiceStub) Update(ctx context.Context, update *domain.ProjectUpdate) error {
	s.updateCalls++
	s.updateContext = ctx
	s.update = update
	return s.updateErr
}

func (*projectServiceStub) Delete(context.Context, uint) error { return nil }

func TestUpdateForwardsOnlyProvidedFields(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	description := "updated description"
	request := dto.UpdateProjectRequest{
		ProjectID:   42,
		Description: &description,
		Reference:   &reference,
	}
	stub := &projectServiceStub{}
	projectHandler := handler.NewProjectHandler(stub)
	handlerContext := newProjectHandlerContext()

	response, err := projectHandler.Update(handlerContext, request)
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if stub.updateCalls != 1 {
		t.Fatalf("expected one service call, got %d", stub.updateCalls)
	}
	if stub.updateContext != handlerContext {
		t.Fatal("expected handler context to be forwarded to the service")
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
	if !response.Success {
		t.Fatal("expected successful update response")
	}
}

func TestUpdatePropagatesServiceError(t *testing.T) {
	wantErr := errors.New("update project failed")
	projectHandler := handler.NewProjectHandler(&projectServiceStub{updateErr: wantErr})

	response, err := projectHandler.Update(newProjectHandlerContext(), dto.UpdateProjectRequest{ProjectID: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if response.Success {
		t.Fatal("expected unsuccessful update response")
	}
}

func newProjectHandlerContext() *echox.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	return &echox.Context{Context: e.NewContext(req, httptest.NewRecorder())}
}
