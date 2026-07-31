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
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

type uploadManagerStub struct {
	target  *upload.UploadTarget
	err     error
	ctx     context.Context
	request *upload.CreateUploadTargetRequest
	calls   int
}

func (s *uploadManagerStub) CreateUploadTarget(
	ctx context.Context,
	request *upload.CreateUploadTargetRequest,
) (*upload.UploadTarget, error) {
	s.calls++
	s.ctx = ctx
	s.request = request
	return s.target, s.err
}

func TestCreateUploadTargetForwardsRequestToManager(t *testing.T) {
	wantTarget := &upload.UploadTarget{
		ObjectKey:   "users/7/uploads/uuid",
		ObjectURL:   "https://files.example.com/users/7/uploads/uuid",
		UploadURL:   "https://upload.qiniup.com",
		UploadToken: "access-key:signature:policy",
	}
	stub := &uploadManagerStub{target: wantTarget}
	uploadHandler := handler.NewUploadHandler(stub)
	request := dto.CreateUploadTargetRequest{
		ContentType:   "image/png",
		ContentLength: 8,
	}
	handlerContext := newHandlerContext()

	target, err := uploadHandler.CreateUploadTarget(handlerContext, request)
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("expected one service call, got %d", stub.calls)
	}
	if stub.ctx != handlerContext {
		t.Fatal("expected handler context to be forwarded to the service")
	}
	if stub.request == nil || stub.request.ContentType != request.ContentType || stub.request.ContentLength != request.ContentLength {
		t.Fatalf("expected request %+v, got %+v", request, stub.request)
	}
	if target.ObjectKey != wantTarget.ObjectKey || target.ObjectURL != wantTarget.ObjectURL || target.UploadURL != wantTarget.UploadURL || target.UploadToken != wantTarget.UploadToken {
		t.Fatalf("expected target %+v, got %+v", wantTarget, target)
	}
}

func TestCreateUploadTargetPropagatesManagerError(t *testing.T) {
	wantErr := errors.New("create upload target failed")
	stub := &uploadManagerStub{err: wantErr}
	uploadHandler := handler.NewUploadHandler(stub)

	target, err := uploadHandler.CreateUploadTarget(
		newHandlerContext(),
		dto.CreateUploadTargetRequest{},
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
