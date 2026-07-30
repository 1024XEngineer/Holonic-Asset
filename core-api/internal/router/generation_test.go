package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/router"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type generationRouterStub struct {
	listRequest *dto.ListGenerationRunsRequest
}

func (*generationRouterStub) Create(
	*echox.Context,
	dto.CreateGenerationRequest,
) (dto.CreateGenerationResponse, error) {
	return dto.CreateGenerationResponse{}, nil
}

func (s *generationRouterStub) List(
	_ *echox.Context,
	request dto.ListGenerationRunsRequest,
) (dto.ListGenerationRunsResponse, error) {
	s.listRequest = &request
	return dto.ListGenerationRunsResponse{}, nil
}

func (*generationRouterStub) Get(
	*echox.Context,
	dto.GetGenerationRequest,
) (dto.GetGenerationResponse, error) {
	return dto.GetGenerationResponse{}, nil
}

func (*generationRouterStub) Cancel(
	*echox.Context,
	dto.CancelGenerationRequest,
) (dto.CancelGenerationResponse, error) {
	return dto.CancelGenerationResponse{}, nil
}

func TestGenerationRoutesAreRegistered(t *testing.T) {
	stub := &generationRouterStub{}
	e := router.Register(nil, nil, stub, nil)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{
			method: http.MethodPost,
			path:   "/api/v1/projects/42/generation-runs",
			body:   `{"kind":"generate_character","prompt":"hero"}`,
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/generation-runs/7",
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/projects/42/generation-runs?assetId=9&status=active&limit=10&cursor=next",
		},
		{
			method: http.MethodPost,
			path:   "/api/v1/generation-runs/7/cancel",
			body:   `{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
			}
		})
	}

	if stub.listRequest == nil || stub.listRequest.ProjectID != 42 ||
		stub.listRequest.AssetID == nil || *stub.listRequest.AssetID != 9 ||
		stub.listRequest.Status != "active" || stub.listRequest.Limit != 10 ||
		stub.listRequest.Cursor != "next" {
		t.Fatalf("unexpected list request: %+v", stub.listRequest)
	}
}

func TestAIRoutesAreNotExposed(t *testing.T) {
	e := router.Register(nil, nil, &generationRouterStub{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tile-set/item/edit", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestCandidateConfirmationRouteIsNotExposed(t *testing.T) {
	e := router.Register(nil, nil, &generationRouterStub{}, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/generation-runs/7/candidates/9/confirm",
		strings.NewReader(`{}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}
