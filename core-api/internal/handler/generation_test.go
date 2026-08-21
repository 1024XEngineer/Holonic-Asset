package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	createErr     error
	listQuery     *generator.RunListQuery
	listPage      *generator.RunListPage
	listErr       error
	run           *generator.Run
	cancelID      generator.RunID
	cancelErr     error
	resolveID     generator.RunID
	resolved      bool
	resolveErr    error
}

func (s *runManagerStub) Create(
	_ context.Context,
	request *generator.Request,
) (generator.RunID, error) {
	s.createRequest = request
	return 17, s.createErr
}

type failedTaskManagerStub struct {
	retryID     uint
	retryStatus taskdomain.Status
	retryErr    error
	deleteID    uint
	deleteErr   error
}

func (s *failedTaskManagerStub) RetryFailed(
	_ context.Context,
	taskID uint,
	completionStatus taskdomain.Status,
) error {
	s.retryID = taskID
	s.retryStatus = completionStatus
	return s.retryErr
}

func (s *failedTaskManagerStub) DeleteFailed(_ context.Context, taskID uint) error {
	s.deleteID = taskID
	return s.deleteErr
}

func TestCreateMapsInvalidTaskPayloadToBadRequest(t *testing.T) {
	wantErr := fmt.Errorf("%w: invalid target", generator.ErrInvalidTaskPayload)
	stub := &runManagerStub{createErr: wantErr}
	_, err := handler.NewGenerationHandler(stub, nil).Create(
		context.Background(),
		dto.CreateGenerationRequest{Kind: generator.EditTiles, CreativeBrief: "edit"},
	)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Code != http.StatusBadRequest || !errors.Is(err, wantErr) {
		t.Fatalf("expected invalid payload bad request, got %v", err)
	}
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

func (s *runManagerStub) ResolveApplication(_ context.Context, runID generator.RunID, applied bool) error {
	s.resolveID = runID
	s.resolved = applied
	return s.resolveErr
}

func TestCreateMapsTransportRequest(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{}
	generationHandler := handler.NewGenerationHandler(stub, nil)
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
		Kind:      generator.EditCharacterProtoType,
		Status:    taskdomain.StatusCompleted,
		Result:    json.RawMessage(`{"asset_id":3,"version":2}`),
	}}

	response, err := handler.NewGenerationHandler(stub, nil).Get(
		context.Background(),
		dto.GetGenerationRequest{GenerationRunID: 7},
	)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if response.Data.ID != 7 || response.Data.ProjectID != 2 || response.Data.AssetID == nil ||
		*response.Data.AssetID != assetID || response.Data.Kind != generator.EditCharacterProtoType ||
		response.Data.Status != "completed" || response.Data.Result == nil ||
		response.Data.Result.AssetID != assetID || response.Data.Result.Version != 2 {
		t.Fatalf("unexpected run response: %+v", response)
	}
}

func TestGetResolvesGenerationResultContentReferences(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{run: &generator.Run{
		ID:        7,
		ProjectID: 2,
		AssetID:   &assetID,
		Kind:      generator.GenerateAnimation,
		Status:    taskdomain.StatusAwaitingApplication,
		Result: json.RawMessage(`{
			"asset_id":3,
			"version":2,
			"content":{"animations":[{"name":"walk","frames":[{"id":1,"url":"uploads/frame.png"}],"generation":{"direction":"front","frameCount":1,"columns":1,"frameWidth":32,"frameHeight":32,"fps":10,"resolution":"720p","duration":1,"aspectRatio":"1:1"}}]}
		}`),
	}}
	resolver := &referenceResolverStub{}

	response, err := handler.NewGenerationHandler(stub, nil, resolver).Get(
		context.Background(),
		dto.GetGenerationRequest{GenerationRunID: 7},
	)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	var content struct {
		Animations []struct {
			ID     *uint `json:"id"`
			Frames []struct {
				URL string `json:"url"`
			} `json:"frames"`
			Generation json.RawMessage `json:"generation"`
		} `json:"animations"`
	}
	if response.Data.Result == nil || json.Unmarshal(response.Data.Result.Content, &content) != nil {
		t.Fatalf("decode generation result content: %+v", response.Data.Result)
	}
	if len(content.Animations) != 1 || content.Animations[0].ID != nil ||
		len(content.Animations[0].Frames) != 1 ||
		content.Animations[0].Frames[0].URL != "signed:uploads/frame.png" ||
		len(content.Animations[0].Generation) == 0 {
		t.Fatalf("unexpected resolved generation content: %+v", content)
	}
	if !reflect.DeepEqual(resolver.calls, []string{"uploads/frame.png"}) {
		t.Fatalf("unexpected resolver calls: %v", resolver.calls)
	}
}

func TestListMapsTaskBackedRuns(t *testing.T) {
	assetID := uint(3)
	stub := &runManagerStub{listPage: &generator.RunListPage{
		Runs: []generator.Run{
			{ID: 7, ProjectID: 2, AssetID: &assetID, Kind: generator.GenerateAnimation, Status: taskdomain.StatusProcessing},
			{ID: 8, ProjectID: 2, AssetID: &assetID, Kind: generator.EditFrames, Status: taskdomain.StatusPending},
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
	response, err := handler.NewGenerationHandler(stub, nil).List(context.Background(), query)
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
	_, err := handler.NewGenerationHandler(stub, nil).List(
		context.Background(),
		dto.ListGenerationRunsRequest{Status: "completed"},
	)
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestListRejectsInvalidCursor(t *testing.T) {
	stub := &runManagerStub{listErr: generator.ErrInvalidRunListCursor}
	_, err := handler.NewGenerationHandler(stub, nil).List(
		context.Background(),
		dto.ListGenerationRunsRequest{Cursor: "invalid"},
	)
	if !errors.Is(err, echo.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestCancelForwardsTaskBackedRunID(t *testing.T) {
	stub := &runManagerStub{}
	response, err := handler.NewGenerationHandler(stub, nil).Cancel(
		context.Background(),
		dto.CancelGenerationRequest{GenerationRunID: 7},
	)
	if err != nil || !response.Data.Cancelled || stub.cancelID != 7 {
		t.Fatalf("unexpected cancel response: %+v, id=%d, err=%v", response, stub.cancelID, err)
	}
}

func TestRetryForwardsTaskBackedRunID(t *testing.T) {
	runs := &runManagerStub{run: &generator.Run{Kind: generator.GenerateAnimation}}
	tasks := &failedTaskManagerStub{}
	response, err := handler.NewGenerationHandler(runs, tasks).Retry(
		context.Background(),
		dto.RetryGenerationRequest{GenerationRunID: 7},
	)
	if err != nil || response.Data.GenerationRunID != 7 || tasks.retryID != 7 ||
		tasks.retryStatus != taskdomain.StatusAwaitingApplication {
		t.Fatalf("unexpected retry response: %+v, id=%d, status=%s, err=%v", response, tasks.retryID, tasks.retryStatus, err)
	}
}

func TestDeleteForwardsTaskBackedRunID(t *testing.T) {
	runs := &runManagerStub{run: &generator.Run{Kind: generator.GenerateScenery}}
	tasks := &failedTaskManagerStub{}
	response, err := handler.NewGenerationHandler(runs, tasks).Delete(
		context.Background(),
		dto.DeleteGenerationRequest{GenerationRunID: 7},
	)
	if err != nil || !response.Data.Deleted || tasks.deleteID != 7 {
		t.Fatalf("unexpected delete response: %+v, id=%d, err=%v", response, tasks.deleteID, err)
	}
}

func TestRetryAndDeleteRejectNonFailedRuns(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(*handler.GenerationHandler) error
	}{
		{name: "retry", run: func(generationHandler *handler.GenerationHandler) error {
			_, err := generationHandler.Retry(context.Background(), dto.RetryGenerationRequest{GenerationRunID: 7})
			return err
		}},
		{name: "delete", run: func(generationHandler *handler.GenerationHandler) error {
			_, err := generationHandler.Delete(context.Background(), dto.DeleteGenerationRequest{GenerationRunID: 7})
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			runs := &runManagerStub{run: &generator.Run{Kind: generator.GenerateScenery}}
			tasks := &failedTaskManagerStub{retryErr: taskdomain.ErrTaskNotFailed, deleteErr: taskdomain.ErrTaskNotFailed}
			err := operation.run(handler.NewGenerationHandler(runs, tasks))
			var httpErr *echo.HTTPError
			if !errors.As(err, &httpErr) || httpErr.Code != http.StatusConflict ||
				!errors.Is(err, taskdomain.ErrTaskNotFailed) {
				t.Fatalf("expected conflict, got %v", err)
			}
		})
	}
}

func TestRetryReturnsUnexpectedManagerError(t *testing.T) {
	wantErr := errors.New("retry unavailable")
	runs := &runManagerStub{run: &generator.Run{Kind: generator.GenerateScenery}}
	tasks := &failedTaskManagerStub{retryErr: wantErr}
	_, err := handler.NewGenerationHandler(runs, tasks).Retry(
		context.Background(),
		dto.RetryGenerationRequest{GenerationRunID: 7},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected manager error, got %v", err)
	}
}

func TestResolveApplicationForwardsTaskBackedRun(t *testing.T) {
	stub := &runManagerStub{}
	err := handler.NewGenerationHandler(stub, nil).ResolveApplication(
		context.Background(),
		dto.ResolveGenerationApplicationRequest{GenerationRunID: 7, Applied: true},
	)
	if err != nil || stub.resolveID != 7 || !stub.resolved {
		t.Fatalf("unexpected application result: stub=%+v, err=%v", stub, err)
	}
}

func TestResolveApplicationReturnsManagerError(t *testing.T) {
	wantErr := errors.New("transition failed")
	stub := &runManagerStub{resolveErr: wantErr}
	err := handler.NewGenerationHandler(stub, nil).ResolveApplication(
		context.Background(),
		dto.ResolveGenerationApplicationRequest{GenerationRunID: 7},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected manager error, got %v", err)
	}
}
