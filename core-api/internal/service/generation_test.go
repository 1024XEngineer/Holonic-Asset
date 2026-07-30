package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

type generationReaderStub struct {
	filter *domain.RunListFilter
	page   *domain.RunListPage
}

func (s *generationReaderStub) ListRuns(
	_ context.Context,
	filter *domain.RunListFilter,
) (*domain.RunListPage, error) {
	s.filter = filter
	return s.page, nil
}

var _ repository.GenerationTaskReader = (*generationReaderStub)(nil)

type taskServiceStub struct {
	createdTask   *taskdomain.Task
	createID      uint
	detail        *taskdomain.Task
	statusUpdates []taskStatusUpdate
	resultTaskID  uint
	result        json.RawMessage
}

type taskStatusUpdate struct {
	taskID uint
	status taskdomain.Status
}

func (s *taskServiceStub) Create(_ context.Context, message *taskdomain.Task) (uint, error) {
	s.createdTask = message
	return s.createID, nil
}

func (s *taskServiceStub) GetDetail(context.Context, uint) (*taskdomain.Task, error) {
	return s.detail, nil
}

func (s *taskServiceStub) UpdateStatus(_ context.Context, taskID uint, status taskdomain.Status) error {
	s.statusUpdates = append(s.statusUpdates, taskStatusUpdate{taskID: taskID, status: status})
	return nil
}

func (s *taskServiceStub) UpdateResult(_ context.Context, taskID uint, result json.RawMessage) error {
	s.resultTaskID = taskID
	s.result = result
	return nil
}

func TestCreateBuildsOneTaskFromGenerationRequest(t *testing.T) {
	assetID := uint(9)
	tasks := &taskServiceStub{createID: 17}
	generationService := service.NewGenerationService(nil, tasks, nil)
	request := &domain.GenerationRequest{
		ProjectID:         42,
		AssetID:           &assetID,
		Kind:              domain.GenerateCharacterAnimation,
		Prompt:            "walk",
		ReferenceMediaIDs: []string{"media-1"},
		Parameters:        json.RawMessage(`{"asset_name":"hero walk"}`),
	}

	runID, err := generationService.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if runID != 17 || tasks.createdTask == nil {
		t.Fatalf("unexpected task creation: run=%d task=%+v", runID, tasks.createdTask)
	}
	if tasks.createdTask.Type != string(request.Kind) ||
		tasks.createdTask.Status != taskdomain.StatusPending {
		t.Fatalf("unexpected task envelope: %+v", tasks.createdTask)
	}

	var payload domain.CreateCharacterAnimationPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.ProjectID != request.ProjectID || payload.ParentID != assetID ||
		payload.AssetName != "hero walk" || payload.CreativeBrief != request.Prompt {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
}

func TestCreateBuildsCharacterPrototypePayload(t *testing.T) {
	tasks := &taskServiceStub{createID: 17}
	generationService := service.NewGenerationService(nil, tasks, nil)

	_, err := generationService.Create(context.Background(), &domain.GenerationRequest{
		ProjectID:         42,
		Kind:              domain.GenerateCharacterProtoType,
		Prompt:            "hero",
		ReferenceMediaIDs: []string{"media-1"},
		Parameters: json.RawMessage(
			`{"asset_name":"knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4"}`,
		),
	})
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}

	var payload domain.CreateCharacterPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.ProjectID != 42 || payload.AssetName != "knight" ||
		payload.CreativeBrief != "hero" || payload.Reference != "media-1" ||
		payload.CanvasSize != "64x64" || payload.Perspective != "top-down" ||
		payload.DirectionCount != "4" {
		t.Fatalf("unexpected character prototype payload: %+v", payload)
	}
}

func TestGetProjectsTaskAsGenerationRun(t *testing.T) {
	assetID := uint(9)
	payload, err := json.Marshal(domain.CreateCharacterAnimationPayload{
		ProjectID: 42,
		ParentID:  assetID,
	})
	if err != nil {
		t.Fatalf("encode task payload: %v", err)
	}
	tasks := &taskServiceStub{detail: &taskdomain.Task{
		ID:      17,
		Type:    string(domain.GenerateCharacterAnimation),
		Status:  taskdomain.StatusProcessing,
		Payload: payload,
		Result:  json.RawMessage(`{"asset_id":9}`),
	}}
	generationService := service.NewGenerationService(nil, tasks, nil)

	run, err := generationService.Get(context.Background(), 17)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if run.ID != 17 || run.ProjectID != 42 || run.AssetID == nil || *run.AssetID != assetID ||
		run.Kind != domain.GenerateCharacterAnimation || run.Status != taskdomain.StatusProcessing {
		t.Fatalf("unexpected generation run: %+v", run)
	}
}

func TestListBuildsProjectScopeTaskFilter(t *testing.T) {
	reader := &generationReaderStub{page: &domain.RunListPage{}}
	generationService := service.NewGenerationService(reader, &taskServiceStub{}, nil)

	_, err := generationService.List(context.Background(), &domain.RunListQuery{
		ProjectID: 42,
		Status:    domain.RunListStatusActive,
		Limit:     10,
		Cursor:    "cursor",
	})
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if reader.filter == nil || reader.filter.ProjectID != 42 || reader.filter.AssetID != nil ||
		reader.filter.Limit != 10 || reader.filter.Cursor != "cursor" {
		t.Fatalf("unexpected filter: %+v", reader.filter)
	}
	if !reflect.DeepEqual(reader.filter.Statuses, domain.ActiveTaskStatuses()) {
		t.Fatalf("unexpected statuses: %v", reader.filter.Statuses)
	}
	if !reflect.DeepEqual(reader.filter.IncludeTaskTypes, domain.ProjectLevelTaskTypes()) {
		t.Fatalf("unexpected project task types: %v", reader.filter.IncludeTaskTypes)
	}
}

func TestListBuildsAssetScopeTaskFilter(t *testing.T) {
	assetID := uint(9)
	reader := &generationReaderStub{page: &domain.RunListPage{}}
	generationService := service.NewGenerationService(reader, &taskServiceStub{}, nil)

	_, err := generationService.List(context.Background(), &domain.RunListQuery{
		ProjectID: 42,
		AssetID:   &assetID,
	})
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if reader.filter == nil || reader.filter.AssetID == nil || *reader.filter.AssetID != assetID {
		t.Fatalf("unexpected asset filter: %+v", reader.filter)
	}
	if len(reader.filter.IncludeTaskTypes) != 0 ||
		!reflect.DeepEqual(reader.filter.ExcludeTaskTypes, domain.ProjectLevelTaskTypes()) {
		t.Fatalf("unexpected task type filter: %+v", reader.filter)
	}
	if reader.filter.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", reader.filter.Limit)
	}
}

func TestListRejectsUnsupportedStatus(t *testing.T) {
	generationService := service.NewGenerationService(&generationReaderStub{}, &taskServiceStub{}, nil)
	_, err := generationService.List(context.Background(), &domain.RunListQuery{Status: "completed"})
	if !errors.Is(err, service.ErrInvalidRunListStatus) {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestCancelUpdatesTaskStatus(t *testing.T) {
	tasks := &taskServiceStub{}
	generationService := service.NewGenerationService(nil, tasks, nil)

	if err := generationService.Cancel(context.Background(), 17); err != nil {
		t.Fatalf("cancel generation: %v", err)
	}
	if len(tasks.statusUpdates) != 1 || tasks.statusUpdates[0].taskID != 17 ||
		tasks.statusUpdates[0].status != taskdomain.StatusCancelled {
		t.Fatalf("unexpected status updates: %+v", tasks.statusUpdates)
	}
}

type generationAIServiceStub struct {
	taskType domain.TaskType
	payload  json.RawMessage
	result   json.RawMessage
	err      error
}

func (s *generationAIServiceStub) Generate(
	_ context.Context,
	taskType domain.TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	s.taskType = taskType
	s.payload = payload
	return s.result, s.err
}

func TestRegisteredGenerationTaskHandlersDecodeTheirPayloads(t *testing.T) {
	tests := []struct {
		taskType domain.TaskType
		payload  json.RawMessage
	}{
		{
			taskType: domain.GenerateCharacterProtoType,
			payload:  json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4","reference":"media-1","project_id":11}`),
		},
		{
			taskType: domain.GenerateCharacterAnimation,
			payload:  json.RawMessage(`{"asset_name":"walk","project_id":11,"parent_id":7,"creative_brief":"walking cycle"}`),
		},
		{
			taskType: domain.GenerateObjectProtoType,
			payload:  json.RawMessage(`{"asset_name":"chest","creative_brief":"wooden chest","canvas_size":"64x64","perspective":"isometric","reference":"media-2","project_id":11}`),
		},
		{
			taskType: domain.GenerateObjectAnimation,
			payload:  json.RawMessage(`{"asset_name":"open chest","project_id":11,"parent_id":8,"creative_brief":"opening animation"}`),
		},
		{
			taskType: domain.GenerateTileSet,
			payload:  json.RawMessage(`{"asset_name":"forest","project_id":11,"creative_brief":"forest ground","tile_num":2,"tile_descriptions":["grass","path"],"reference":"media-3"}`),
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			tasks := &taskServiceStub{}
			ai := &generationAIServiceStub{}
			generationService := service.NewGenerationService(nil, tasks, ai)
			registry := taskdomain.NewRegistry()
			service.RegisterGenerationTaskHandlers(registry, generationService)

			message := &taskdomain.Task{ID: 17, Type: string(tt.taskType), Payload: tt.payload}
			if err := registry.Dispatch(context.Background(), message); err != nil {
				t.Fatalf("dispatch generation task: %v", err)
			}
			if ai.payload != nil || len(tasks.statusUpdates) != 0 {
				t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v",
					ai.payload, tasks.statusUpdates)
			}
		})
	}
}

func TestRegisteredGenerationTaskHandlerRejectsMismatchedPayload(t *testing.T) {
	tasks := &taskServiceStub{}
	ai := &generationAIServiceStub{}
	generationService := service.NewGenerationService(nil, tasks, ai)
	registry := taskdomain.NewRegistry()
	service.RegisterGenerationTaskHandlers(registry, generationService)

	err := registry.Dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(domain.GenerateCharacterProtoType),
		Payload: json.RawMessage(`{"project_id":"not-a-number"}`),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
	if ai.payload != nil || len(tasks.statusUpdates) != 0 {
		t.Fatalf("malformed task must not be processed: payload=%s statuses=%+v",
			ai.payload, tasks.statusUpdates)
	}
}

func TestRegisterGenerationTaskHandlersIncludesEmptyPayloadTypes(t *testing.T) {
	tasks := &taskServiceStub{}
	ai := &generationAIServiceStub{}
	generationService := service.NewGenerationService(nil, tasks, ai)
	registry := taskdomain.NewRegistry()
	service.RegisterGenerationTaskHandlers(registry, generationService)

	for _, taskType := range domain.TaskTypes() {
		message := &taskdomain.Task{
			ID:      uint(len(tasks.statusUpdates) + 1),
			Type:    string(taskType),
			Payload: json.RawMessage(`{}`),
		}
		if err := registry.Dispatch(context.Background(), message); err != nil {
			t.Fatalf("dispatch task type %q: %v", taskType, err)
		}
	}
	if ai.payload != nil || len(tasks.statusUpdates) != 0 {
		t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v",
			ai.payload, tasks.statusUpdates)
	}
}

func TestHandleCharacterPrototypeOnlyDecodesPayload(t *testing.T) {
	payload := json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4","reference":"media-1","project_id":42}`)
	tasks := &taskServiceStub{}
	ai := &generationAIServiceStub{}
	generationService := service.NewGenerationService(nil, tasks, ai)

	got, err := generationService.HandleCharacterPrototype(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(domain.GenerateCharacterProtoType),
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("handle generation task: %v", err)
	}
	if got != nil {
		t.Fatalf("handler skeleton must return a nil result, got %v", got)
	}
	if ai.payload != nil || len(tasks.statusUpdates) != 0 || tasks.result != nil {
		t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v result=%s",
			ai.payload, tasks.statusUpdates, tasks.result)
	}
}
