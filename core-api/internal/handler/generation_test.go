package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type requestServiceStub struct {
	createRequest *domain.GenerationRequest
	listQuery     *domain.RunListQuery
	listPage      *domain.RunListPage
	listErr       error
	run           *domain.GenerationRun
	cancelID      domain.RunID
	cancelErr     error
}

func (s *requestServiceStub) Create(
	_ context.Context,
	request *domain.GenerationRequest,
) (domain.RunID, error) {
	s.createRequest = request
	return 17, nil
}

func (s *requestServiceStub) List(
	_ context.Context,
	query *domain.RunListQuery,
) (*domain.RunListPage, error) {
	s.listQuery = query
	if s.listPage == nil {
		return &domain.RunListPage{}, s.listErr
	}
	return s.listPage, s.listErr
}

func (s *requestServiceStub) Get(context.Context, domain.RunID) (*domain.GenerationRun, error) {
	return s.run, nil
}

func (s *requestServiceStub) Cancel(_ context.Context, runID domain.RunID) error {
	s.cancelID = runID
	return s.cancelErr
}

func TestCreateMapsTransportRequest(t *testing.T) {
	assetID := uint(3)
	stub := &requestServiceStub{}
	generationHandler := handler.NewGenerationHandler(stub)
	parameters := json.RawMessage(`{"size":{"width":64,"height":64}}`)
	request := dto.CreateGenerationRequest{
		ProjectID:         2,
		AssetID:           &assetID,
		Kind:              domain.GenerateCharacterAnimation,
		Prompt:            "hero",
		ReferenceMediaIDs: []string{"media-1"},
		TargetAssetPaths:  []string{"animations.walk.frames"},
		Parameters:        parameters,
	}

	response, err := generationHandler.Create(newGenerationHandlerContext(), request)
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if response.GenerationRunID != 17 {
		t.Fatalf("expected run ID 17, got %d", response.GenerationRunID)
	}
	if stub.createRequest == nil || stub.createRequest.AssetID == nil ||
		stub.createRequest.ProjectID != request.ProjectID ||
		*stub.createRequest.AssetID != assetID || stub.createRequest.Kind != request.Kind ||
		stub.createRequest.Prompt != request.Prompt ||
		!reflect.DeepEqual(stub.createRequest.ReferenceMediaIDs, request.ReferenceMediaIDs) ||
		!reflect.DeepEqual(stub.createRequest.TargetAssetPaths, request.TargetAssetPaths) ||
		!reflect.DeepEqual(stub.createRequest.Parameters, request.Parameters) {
		t.Fatalf("unexpected generation request: %+v", stub.createRequest)
	}
}

func TestGetMapsTaskBackedGeneration(t *testing.T) {
	assetID := uint(3)
	stub := &requestServiceStub{run: &domain.GenerationRun{
		ID:        7,
		ProjectID: 2,
		AssetID:   &assetID,
		Kind:      domain.GenerateCharacterAnimation,
		Status:    taskdomain.StatusProcessing,
		Result:    json.RawMessage(`{"media_ids":["media-1"]}`),
	}}

	response, err := handler.NewGenerationHandler(stub).Get(
		newGenerationHandlerContext(),
		dto.GetGenerationRequest{GenerationRunID: 7},
	)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if response.ID != 7 || response.ProjectID != 2 || response.AssetID == nil ||
		*response.AssetID != assetID || response.Kind != domain.GenerateCharacterAnimation ||
		response.Status != taskdomain.StatusProcessing {
		t.Fatalf("unexpected run response: %+v", response)
	}
}

func TestListMapsTaskBackedRuns(t *testing.T) {
	assetID := uint(3)
	stub := &requestServiceStub{listPage: &domain.RunListPage{
		Runs: []domain.GenerationRun{
			{ID: 7, ProjectID: 2, AssetID: &assetID, Kind: domain.GenerateCharacterAnimation, Status: taskdomain.StatusProcessing},
			{ID: 8, ProjectID: 2, AssetID: &assetID, Kind: domain.RegenerateCharacterFrames, Status: taskdomain.StatusPending},
		},
		NextCursor: "next",
	}}

	query := dto.ListGenerationRunsRequest{
		ProjectID: 42,
		AssetID:   &assetID,
		Status:    domain.RunListStatusActive,
		Limit:     10,
		Cursor:    "cursor",
	}
	response, err := handler.NewGenerationHandler(stub).List(newGenerationHandlerContext(), query)
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if stub.listQuery == nil || stub.listQuery.AssetID == nil ||
		*stub.listQuery.AssetID != assetID || stub.listQuery.Status != query.Status {
		t.Fatalf("unexpected list query: %+v", stub.listQuery)
	}
	if len(response.Items) != 2 || response.Items[0].ID != 7 || response.Items[1].ID != 8 ||
		response.Items[0].Status != taskdomain.StatusProcessing ||
		response.Items[1].Status != taskdomain.StatusPending || response.NextCursor != "next" {
		t.Fatalf("unexpected list response: %+v", response)
	}
}

func TestListRejectsUnsupportedStatus(t *testing.T) {
	stub := &requestServiceStub{listErr: service.ErrInvalidRunListStatus}
	_, err := handler.NewGenerationHandler(stub).List(
		newGenerationHandlerContext(),
		dto.ListGenerationRunsRequest{Status: "completed"},
	)
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestCancelForwardsTaskBackedRunID(t *testing.T) {
	stub := &requestServiceStub{}
	response, err := handler.NewGenerationHandler(stub).Cancel(
		newGenerationHandlerContext(),
		dto.CancelGenerationRequest{GenerationRunID: 7},
	)
	if err != nil || !response.Cancelled || stub.cancelID != 7 {
		t.Fatalf("unexpected cancel response: %+v, id=%d, err=%v", response, stub.cancelID, err)
	}
}

func newGenerationHandlerContext() *echox.Context {
	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	return &echox.Context{Context: e.NewContext(request, httptest.NewRecorder())}
}
