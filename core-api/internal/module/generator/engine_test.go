package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

type generationReaderStub struct {
	filter *generator.RunListFilter
	page   *generator.RunListPage
}

func (s *generationReaderStub) ListRuns(
	_ context.Context,
	filter *generator.RunListFilter,
) (*generator.RunListPage, error) {
	s.filter = filter
	return s.page, nil
}

var _ generator.RunReader = (*generationReaderStub)(nil)

type taskManagerStub struct {
	createdTask   *taskdomain.Task
	createID      uint
	detail        *taskdomain.Task
	statusUpdates []taskStatusUpdate
	handlers      map[string]taskdomain.Handler
}

type taskStatusUpdate struct {
	taskID uint
	status taskdomain.Status
}

func (s *taskManagerStub) Register(taskType string, handler taskdomain.Handler) {
	if s.handlers == nil {
		s.handlers = make(map[string]taskdomain.Handler)
	}
	s.handlers[taskType] = handler
}

func (s *taskManagerStub) Start(context.Context) error { return nil }

func (s *taskManagerStub) Stop() error { return nil }

func (s *taskManagerStub) Publish(_ context.Context, message *taskdomain.Task) (uint, error) {
	s.createdTask = message
	return s.createID, nil
}

func (s *taskManagerStub) GetDetail(context.Context, uint) (*taskdomain.Task, error) {
	return s.detail, nil
}

func (*taskManagerStub) ListByStatus(
	context.Context,
	taskdomain.Status,
) ([]*taskdomain.Task, error) {
	return nil, nil
}

func (s *taskManagerStub) Cancel(_ context.Context, taskID uint) error {
	s.statusUpdates = append(s.statusUpdates, taskStatusUpdate{taskID: taskID, status: taskdomain.StatusCancelled})
	return nil
}

func (s *taskManagerStub) dispatch(
	ctx context.Context,
	message *taskdomain.Task,
) (any, error) {
	if s.handlers == nil {
		return nil, errors.New("task handler is not registered")
	}
	handler, ok := s.handlers[message.Type]
	if !ok {
		return nil, errors.New("task handler is not registered")
	}
	return handler.Handle(ctx, message)
}

var _ taskdomain.Manager = (*taskManagerStub)(nil)

func TestCreateBuildsOneTaskFromRequest(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil, nil)
	request := &generator.Request{
		ProjectID:         42,
		AssetID:           &assetID,
		Kind:              generator.GenerateCharacterAnimation,
		Prompt:            "walk",
		ReferenceMediaIDs: []string{"media-1"},
		Parameters:        json.RawMessage(`{"asset_name":"hero walk"}`),
	}

	runID, err := engine.Create(context.Background(), request)
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

	var payload generator.CreateCharacterAnimationPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.ProjectID != request.ProjectID || payload.ParentID != assetID ||
		payload.AssetName != "hero walk" || payload.CreativeBrief != request.Prompt {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
}

func TestCreateBuildsCharacterPrototypePayload(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:         42,
		Kind:              generator.GenerateCharacterProtoType,
		Prompt:            "hero",
		ReferenceMediaIDs: []string{"media-1"},
		Parameters: json.RawMessage(
			`{"asset_name":"knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4"}`,
		),
	})
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}

	var payload generator.CreateCharacterPrototypePayload
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

func TestGetProjectsTaskAsRun(t *testing.T) {
	assetID := uint(9)
	payload, err := json.Marshal(generator.CreateCharacterAnimationPayload{
		ProjectID: 42,
		ParentID:  assetID,
	})
	if err != nil {
		t.Fatalf("encode task payload: %v", err)
	}
	tasks := &taskManagerStub{detail: &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateCharacterAnimation),
		Status:  taskdomain.StatusProcessing,
		Payload: payload,
		Result:  json.RawMessage(`{"asset_id":9}`),
	}}
	engine := generator.NewEngine(tasks, nil, nil)

	run, err := engine.Get(context.Background(), 17)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if run.ID != 17 || run.ProjectID != 42 || run.AssetID == nil || *run.AssetID != assetID ||
		run.Kind != generator.GenerateCharacterAnimation || run.Status != taskdomain.StatusProcessing {
		t.Fatalf("unexpected generation run: %+v", run)
	}
}

func TestListBuildsProjectScopeTaskFilter(t *testing.T) {
	reader := &generationReaderStub{page: &generator.RunListPage{}}
	engine := generator.NewEngine(&taskManagerStub{}, reader, nil)

	_, err := engine.List(context.Background(), &generator.RunListQuery{
		ProjectID: 42,
		Status:    generator.RunListStatusActive,
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
	if !reflect.DeepEqual(reader.filter.Statuses, generator.ActiveTaskStatuses()) {
		t.Fatalf("unexpected statuses: %v", reader.filter.Statuses)
	}
	if !reflect.DeepEqual(reader.filter.IncludeTaskTypes, generator.ProjectLevelTaskTypes()) {
		t.Fatalf("unexpected project task types: %v", reader.filter.IncludeTaskTypes)
	}
}

func TestListBuildsAssetScopeTaskFilter(t *testing.T) {
	assetID := uint(9)
	reader := &generationReaderStub{page: &generator.RunListPage{}}
	engine := generator.NewEngine(&taskManagerStub{}, reader, nil)

	_, err := engine.List(context.Background(), &generator.RunListQuery{
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
		!reflect.DeepEqual(reader.filter.ExcludeTaskTypes, generator.ProjectLevelTaskTypes()) {
		t.Fatalf("unexpected task type filter: %+v", reader.filter)
	}
	if reader.filter.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", reader.filter.Limit)
	}
}

func TestListRejectsUnsupportedStatus(t *testing.T) {
	engine := generator.NewEngine(&taskManagerStub{}, &generationReaderStub{}, nil)
	_, err := engine.List(context.Background(), &generator.RunListQuery{Status: "completed"})
	if !errors.Is(err, generator.ErrInvalidRunListStatus) {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestCancelUpdatesTaskStatus(t *testing.T) {
	tasks := &taskManagerStub{}
	engine := generator.NewEngine(tasks, nil, nil)

	if err := engine.Cancel(context.Background(), 17); err != nil {
		t.Fatalf("cancel generation: %v", err)
	}
	if len(tasks.statusUpdates) != 1 || tasks.statusUpdates[0].taskID != 17 ||
		tasks.statusUpdates[0].status != taskdomain.StatusCancelled {
		t.Fatalf("unexpected status updates: %+v", tasks.statusUpdates)
	}
}

type executorStub struct {
	taskType generator.TaskType
	payload  json.RawMessage
	result   json.RawMessage
	err      error
}

func (s *executorStub) Generate(
	_ context.Context,
	taskType generator.TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	s.taskType = taskType
	s.payload = payload
	return s.result, s.err
}

func TestRegisteredGeneratorTaskHandlersDecodeTheirPayloads(t *testing.T) {
	tests := []struct {
		taskType generator.TaskType
		payload  json.RawMessage
	}{
		{
			taskType: generator.GenerateCharacterProtoType,
			payload:  json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4","reference":"media-1","project_id":11}`),
		},
		{
			taskType: generator.GenerateCharacterAnimation,
			payload:  json.RawMessage(`{"asset_name":"walk","project_id":11,"parent_id":7,"creative_brief":"walking cycle"}`),
		},
		{
			taskType: generator.GenerateObjectProtoType,
			payload:  json.RawMessage(`{"asset_name":"chest","creative_brief":"wooden chest","canvas_size":"64x64","perspective":"isometric","reference":"media-2","project_id":11}`),
		},
		{
			taskType: generator.GenerateObjectAnimation,
			payload:  json.RawMessage(`{"asset_name":"open chest","project_id":11,"parent_id":8,"creative_brief":"opening animation"}`),
		},
		{
			taskType: generator.GenerateTileSet,
			payload:  json.RawMessage(`{"asset_name":"forest","project_id":11,"creative_brief":"forest ground","tile_num":2,"tile_descriptions":["grass","path"],"reference":"media-3"}`),
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			tasks := &taskManagerStub{}
			executor := &executorStub{}
			generator.NewEngine(tasks, nil, executor)

			message := &taskdomain.Task{ID: 17, Type: string(tt.taskType), Payload: tt.payload}
			if _, err := tasks.dispatch(context.Background(), message); err != nil {
				t.Fatalf("dispatch generation task: %v", err)
			}
			if executor.payload != nil || len(tasks.statusUpdates) != 0 {
				t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v",
					executor.payload, tasks.statusUpdates)
			}
		})
	}
}

func TestRegisteredGeneratorTaskHandlerRejectsMismatchedPayload(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, nil, executor)

	_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateCharacterProtoType),
		Payload: json.RawMessage(`{"project_id":"not-a-number"}`),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
	if executor.payload != nil || len(tasks.statusUpdates) != 0 {
		t.Fatalf("malformed task must not be processed: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
	}
}

func TestNewEngineRegistersAllTaskTypes(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, nil, executor)

	for _, taskType := range generator.TaskTypes() {
		message := &taskdomain.Task{
			ID:      uint(len(tasks.statusUpdates) + 1),
			Type:    string(taskType),
			Payload: json.RawMessage(`{}`),
		}
		if _, err := tasks.dispatch(context.Background(), message); err != nil {
			t.Fatalf("dispatch task type %q: %v", taskType, err)
		}
	}
	if executor.payload != nil || len(tasks.statusUpdates) != 0 {
		t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
	}
}

func TestHandleCharacterPrototypeOnlyDecodesPayload(t *testing.T) {
	payload := json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","canvas_size":"64x64","perspective":"top-down","direction_count":"4","reference":"media-1","project_id":42}`)
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, nil, executor)

	got, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateCharacterProtoType),
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("handle generation task: %v", err)
	}
	if got != nil {
		t.Fatalf("handler skeleton must return a nil result, got %v", got)
	}
	if executor.payload != nil || len(tasks.statusUpdates) != 0 {
		t.Fatalf("handler skeleton must not execute business logic: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
	}
}
