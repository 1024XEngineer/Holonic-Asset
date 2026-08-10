package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type runManagerStub struct {
	createRequest *generator.Request
	listQuery     *generator.RunListQuery
	listPage      *generator.RunListPage
	listErr       error
	run           *generator.Run
	cancelID      generator.RunID
	cancelErr     error
}

func (s *runManagerStub) Create(
	_ context.Context,
	request *generator.Request,
) (generator.RunID, error) {
	s.createRequest = request
	return 17, nil
}

func (s *runManagerStub) List(
	_ context.Context,
	query *generator.RunListQuery,
) (*generator.RunListPage, error) {
	s.listQuery = query
	if s.listPage == nil {
		return &generator.RunListPage{}, s.listErr
	}
	return s.listPage, s.listErr
}

func (s *runManagerStub) Get(context.Context, generator.RunID) (*generator.Run, error) {
	return s.run, nil
}

func (s *runManagerStub) Cancel(_ context.Context, runID generator.RunID) error {
	s.cancelID = runID
	return s.cancelErr
}

func TestCreateMapsTransportRequest(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{}
	generationHandler := handler.NewGenerationHandler(stub)
	parameters := json.RawMessage(`{"size":{"width":64,"height":64}}`)
	request := dto.CreateGenerationRequest{
		ProjectID:        2,
		AssetID:          &assetID,
		Kind:             generator.GenerateAnimation,
		CreativeBrief:    "hero",
		TargetAssetPaths: []string{"animations.walk.frames"},
		Parameters:       parameters,
	}

	response, err := generationHandler.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if response.Data.GenerationRunID != 17 {
		t.Fatalf("expected run ID 17, got %d", response.Data.GenerationRunID)
	}
	if stub.createRequest == nil || stub.createRequest.AssetID == nil ||
		stub.createRequest.ProjectID != request.ProjectID ||
		*stub.createRequest.AssetID != assetID || stub.createRequest.Kind != request.Kind ||
		stub.createRequest.CreativeBrief != request.CreativeBrief ||
		!reflect.DeepEqual(stub.createRequest.TargetAssetPaths, request.TargetAssetPaths) ||
		!reflect.DeepEqual(stub.createRequest.Parameters, request.Parameters) {
		t.Fatalf("unexpected generation request: %+v", stub.createRequest)
	}
}

func TestGetMapsTaskBackedGeneration(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{run: &generator.Run{
		ID:        7,
		ProjectID: 2,
		AssetID:   &assetID,
		Kind:      generator.GenerateAnimation,
		Status:    taskdomain.StatusProcessing,
		Result:    json.RawMessage(`{"media_ids":["media-1"]}`),
	}}

	response, err := handler.NewGenerationHandler(stub).Get(
		context.Background(),
		dto.GetGenerationRequest{GenerationRunID: 7},
	)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if response.Data.ID != 7 || response.Data.ProjectID != 2 || response.Data.AssetID == nil ||
		*response.Data.AssetID != assetID || response.Data.Kind != generator.GenerateAnimation ||
		response.Data.Status != "processing" {
		t.Fatalf("unexpected run response: %+v", response)
	}
}

func TestListMapsTaskBackedRuns(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{listPage: &generator.RunListPage{
		Runs: []generator.Run{
			{ID: 7, ProjectID: 2, AssetID: &assetID, Kind: generator.GenerateAnimation, Status: taskdomain.StatusProcessing},
			{ID: 8, ProjectID: 2, AssetID: &assetID, Kind: generator.EditCharacterFrames, Status: taskdomain.StatusPending},
		},
		NextCursor: "next",
	}}

	query := dto.ListGenerationRunsRequest{
		ProjectID: 42,
		AssetID:   &assetID,
		Status:    generator.RunListStatusActive,
		Limit:     10,
		Cursor:    "cursor",
	}
	response, err := handler.NewGenerationHandler(stub).List(context.Background(), query)
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if stub.listQuery == nil || stub.listQuery.AssetID == nil ||
		*stub.listQuery.AssetID != assetID || stub.listQuery.Status != query.Status {
		t.Fatalf("unexpected list query: %+v", stub.listQuery)
	}
	if len(response.Data.Items) != 2 || response.Data.Items[0].ID != 7 || response.Data.Items[1].ID != 8 ||
		response.Data.Items[0].Status != "processing" ||
		response.Data.Items[1].Status != "pending" || response.Data.NextCursor != "next" {
		t.Fatalf("unexpected list response: %+v", response)
	}
}

func TestListRejectsUnsupportedStatus(t *testing.T) {
	stub := &runManagerStub{listErr: generator.ErrInvalidRunListStatus}
	_, err := handler.NewGenerationHandler(stub).List(
		context.Background(),
		dto.ListGenerationRunsRequest{Status: "completed"},
	)
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestListRejectsInvalidCursor(t *testing.T) {
	stub := &runManagerStub{listErr: generator.ErrInvalidRunListCursor}
	_, err := handler.NewGenerationHandler(stub).List(
		context.Background(),
		dto.ListGenerationRunsRequest{Cursor: "invalid"},
	)
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestCancelForwardsTaskBackedRunID(t *testing.T) {
	stub := &runManagerStub{}
	response, err := handler.NewGenerationHandler(stub).Cancel(
		context.Background(),
		dto.CancelGenerationRequest{GenerationRunID: 7},
	)
	if err != nil || !response.Data.Cancelled || stub.cancelID != 7 {
		t.Fatalf("unexpected cancel response: %+v, id=%d, err=%v", response, stub.cancelID, err)
	}
}
