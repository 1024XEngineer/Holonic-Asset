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
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type projectPreviewUploadServiceStub struct {
	target  *service.ProjectPreviewUploadTarget
	err     error
	ctx     context.Context
	request *service.CreateProjectPreviewUploadRequest
	calls   int
}

func (s *projectPreviewUploadServiceStub) CreateProjectPreviewUploadTarget(
	ctx context.Context,
	request *service.CreateProjectPreviewUploadRequest,
) (*service.ProjectPreviewUploadTarget, error) {
	s.calls++
	s.ctx = ctx
	s.request = request
	return s.target, s.err
}

func TestCreateProjectPreviewUploadTargetForwardsRequestToService(t *testing.T) {
	wantTarget := &service.ProjectPreviewUploadTarget{
		ObjectKey: "users/7/project-previews/uuid",
		UploadURL: "https://r2.example/upload",
	}
	stub := &projectPreviewUploadServiceStub{target: wantTarget}
	mediaHandler := handler.NewMediaHandler(stub)
	request := dto.CreateProjectPreviewUploadRequest{ContentType: "image/png"}
	handlerContext := newHandlerContext()

	target, err := mediaHandler.CreateProjectPreviewUploadTarget(handlerContext, request)
	if err != nil {
		t.Fatalf("create project preview upload target: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("expected one service call, got %d", stub.calls)
	}
	if stub.ctx != handlerContext {
		t.Fatal("expected handler context to be forwarded to the service")
	}
	if stub.request == nil || stub.request.ContentType != request.ContentType {
		t.Fatalf("expected request %+v, got %+v", request, stub.request)
	}
	if target.ObjectKey != wantTarget.ObjectKey || target.UploadURL != wantTarget.UploadURL {
		t.Fatalf("expected target %+v, got %+v", wantTarget, target)
	}
}

func TestCreateProjectPreviewUploadTargetPropagatesServiceError(t *testing.T) {
	wantErr := errors.New("create project preview upload target failed")
	stub := &projectPreviewUploadServiceStub{err: wantErr}
	mediaHandler := handler.NewMediaHandler(stub)

	target, err := mediaHandler.CreateProjectPreviewUploadTarget(
		newHandlerContext(),
		dto.CreateProjectPreviewUploadRequest{},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if target != nil {
		t.Fatalf("expected a nil target, got %+v", target)
	}
}

func newHandlerContext() *echox.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	return &echox.Context{Context: e.NewContext(req, recorder)}
}
