package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type generationRouterStub struct {
	listRequest *dto.ListGenerationRunsRequest
	createCalls int
}

func (s *generationRouterStub) Create(
	context.Context,
	dto.CreateGenerationRequest,
) (dto.SuccessResponse[dto.CreateGenerationResponse], error) {
	s.createCalls++
	return dto.NewTypedSuccessResponse(dto.CreateGenerationResponse{}), nil
}

func (s *generationRouterStub) List(
	_ context.Context,
	request dto.ListGenerationRunsRequest,
) (dto.SuccessResponse[dto.ListGenerationRunsResponse], error) {
	s.listRequest = &request
	return dto.NewTypedSuccessResponse(dto.ListGenerationRunsResponse{}), nil
}

func (*generationRouterStub) Get(
	context.Context,
	dto.GetGenerationRequest,
) (dto.SuccessResponse[dto.GetGenerationResponse], error) {
	return dto.NewTypedSuccessResponse(dto.GetGenerationResponse{}), nil
}

func (*generationRouterStub) Cancel(
	context.Context,
	dto.CancelGenerationRequest,
) (dto.SuccessResponse[dto.CancelGenerationResponse], error) {
	return dto.NewTypedSuccessResponse(dto.CancelGenerationResponse{}), nil
}

func (*generationRouterStub) Retry(
	context.Context,
	dto.RetryGenerationRequest,
) (dto.SuccessResponse[dto.RetryGenerationResponse], error) {
	return dto.NewTypedSuccessResponse(dto.RetryGenerationResponse{GenerationRunID: 7}), nil
}

func (*generationRouterStub) Delete(
	context.Context,
	dto.DeleteGenerationRequest,
) (dto.SuccessResponse[dto.DeleteGenerationResponse], error) {
	return dto.NewTypedSuccessResponse(dto.DeleteGenerationResponse{Deleted: true}), nil
}

func (*generationRouterStub) ResolveApplication(
	context.Context,
	dto.ResolveGenerationApplicationRequest,
) error {
	return nil
}

func TestGenerationRoutesAreRegistered(t *testing.T) {
	stub := &generationRouterStub{}
	e := router.Register(nil, nil, stub, nil)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			method:         http.MethodPost,
			path:           "/api/v1/projects/42/generation-runs",
			body:           `{"kind":"generate_character_prototype","creative_brief":"hero"}`,
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodGet,
			path:           "/api/v1/generation-runs/7",
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodDelete,
			path:           "/api/v1/generation-runs/7",
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodPost,
			path:           "/api/v1/generation-runs/7/retry",
			body:           `{}`,
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodGet,
			path:           "/api/v1/projects/42/generation-runs?assetId=9&status=active&limit=10&cursor=next",
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodPost,
			path:           "/api/v1/generation-runs/7/cancel",
			body:           `{}`,
			expectedStatus: http.StatusOK,
		},
		{
			method:         http.MethodPost,
			path:           "/api/v1/generation-runs/7/application",
			body:           `{"applied":true}`,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()

			e.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", test.expectedStatus, recorder.Code, recorder.Body.String())
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

func TestGenerationListRejectsZeroAssetID(t *testing.T) {
	stub := &generationRouterStub{}
	e := router.Register(nil, nil, stub, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/42/generation-runs?assetId=0", nil)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	if stub.listRequest != nil {
		t.Fatalf("expected invalid filter not to reach the handler, got %+v", stub.listRequest)
	}
}

func TestGenerationCreateRejectsInvalidKind(t *testing.T) {
	stub := &generationRouterStub{}
	e := router.Register(nil, nil, stub, nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/42/generation-runs",
		strings.NewReader(`{"kind":"unsupported","creative_brief":"hero"}`),
	)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	if stub.createCalls != 0 {
		t.Fatalf("expected invalid body not to reach the handler, got %d calls", stub.createCalls)
	}
}

func TestGenerationCreateAcceptsEditFramesParameters(t *testing.T) {
	stub := &generationRouterStub{}
	e := router.Register(nil, nil, stub, nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/42/generation-runs",
		strings.NewReader(`{"kind":"edit_frames","creative_brief":"make the stride longer","assetId":7,"parameters":{"animationId":3,"frameIds":[1]}}`),
	)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if stub.createCalls != 1 {
		t.Fatalf("expected edit frames request to reach the handler, got %d calls", stub.createCalls)
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
