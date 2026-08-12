package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type taskManagerStub struct {
	createdTask   *taskdomain.Task
	createID      uint
	detail        *taskdomain.Task
	statusUpdates []taskStatusUpdate
	handlers      map[string]taskdomain.Handler
	listFilter    *taskdomain.ListFilter
	listedTasks   []*taskdomain.Task
	listErr       error
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

func (s *taskManagerStub) List(
	_ context.Context,
	filter *taskdomain.ListFilter,
) ([]*taskdomain.Task, error) {
	if filter != nil {
		copyFilter := *filter
		copyFilter.Statuses = append([]taskdomain.Status(nil), filter.Statuses...)
		copyFilter.Types = append([]string(nil), filter.Types...)
		s.listFilter = &copyFilter
	}
	return s.listedTasks, s.listErr
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

type projectReaderStub struct {
	project *projectdomain.Project
	err     error
	calls   int
}

type referenceStoreStub struct {
	persisted  []string
	persistErr error
}

func (s *referenceStoreStub) ResolveReference(_ context.Context, reference string) (string, error) {
	return reference, nil
}

func (s *referenceStoreStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persisted = append(s.persisted, reference)
	if s.persistErr != nil {
		return "", s.persistErr
	}
	return "uploads/generated-1.png", nil
}

func (s *referenceStoreStub) NewObjectKey(string) (string, error) {
	return "uploads/generated.png", nil
}

func (s *referenceStoreStub) PersistReferenceAt(context.Context, string, string) error {
	return nil
}

func (s *projectReaderStub) GetDetail(_ context.Context, _ uint) (*projectdomain.Project, error) {
	s.calls++
	return s.project, s.err
}

var _ generator.ProjectReader = (*projectReaderStub)(nil)

func TestCreateBuildsOneTaskFromRequest(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)
	request := &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.GenerateAnimation,
		CreativeBrief: "walk",
		Parameters:    json.RawMessage(`{"animation_name":"hero walk","direction":"back_left"}`),
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

	var payload generator.CreateAnimationPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.ProjectID != request.ProjectID || payload.AssetID != assetID ||
		payload.AnimationName != "hero walk" || payload.Direction != generator.AnimationDirectionBackLeft || payload.CreativeBrief != request.CreativeBrief {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
	if strings.Contains(string(tasks.createdTask.Payload), "parent_id") {
		t.Fatalf("animation task payload must not contain parent_id: %s", tasks.createdTask.Payload)
	}
}

func TestCreateBuildsCharacterPrototypePayload(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		Kind:          generator.GenerateCharacterProtoType,
		CreativeBrief: "hero",
		Parameters: json.RawMessage(
			`{"asset_name":"knight","creative_brief":"incorrect parameter brief","dimensions":{"width":64,"height":64},"perspective":"Top-Down"}`,
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
		payload.CreativeBrief != "hero" || payload.Reference != "" ||
		payload.Dimensions.Width != 64 || payload.Dimensions.Height != 64 || payload.Perspective != "Top-Down" {
		t.Fatalf("unexpected character prototype payload: %+v", payload)
	}
}

func TestCreateBuildsEditCharacterPrototypePayload(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditCharacterProtoType,
		CreativeBrief: "make the cape blue",
		Parameters: json.RawMessage(
			`{"asset_id":99,"project_id":99,"edit_instructions":"ignore me"}`,
		),
	})
	if err != nil {
		t.Fatalf("create character edit: %v", err)
	}

	var payload generator.EditCharacterPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.AssetID != assetID || payload.ProjectID != 42 ||
		payload.EditInstructions != "make the cape blue" {
		t.Fatalf("unexpected character edit payload: %+v", payload)
	}
}

func TestCreateEditCharacterPrototypeDoesNotPrepareProjectReference(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	projects := &projectReaderStub{project: &projectdomain.Project{Reference: "projects/42/reference.png"}}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{
		Projects: projects, References: references,
	})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditCharacterProtoType,
		CreativeBrief: "make the cape blue",
	})
	if err != nil {
		t.Fatalf("create character edit: %v", err)
	}
	if projects.calls != 0 || len(references.persisted) != 0 {
		t.Fatalf("character edit prepared an extra reference: project_calls=%d persisted=%v", projects.calls, references.persisted)
	}
}

func TestCreateEditCharacterPrototypeRequiresAssetID(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		Kind:          generator.EditCharacterProtoType,
		CreativeBrief: "make the cape blue",
	})
	if err == nil {
		t.Fatal("expected missing asset id error")
	}
	if tasks.createdTask != nil {
		t.Fatalf("task published without asset id: %+v", tasks.createdTask)
	}
}

func TestCreatePersistsExplicitReferenceBeforePublishing(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42,
		Kind:      generator.GenerateObjectProtoType,
		Parameters: json.RawMessage(`{
			"reference":"https://cdn.example.com/projects/42/reference.png"
		}`),
	})
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	var payload generator.CreateObjectPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.Reference != "uploads/generated-1.png" || len(references.persisted) != 1 ||
		references.persisted[0] != "https://cdn.example.com/projects/42/reference.png" {
		t.Fatalf("reference was not persisted before publish: payload=%+v persisted=%v", payload, references.persisted)
	}
}

func TestCreateUsesProjectReferenceWhenPayloadOmitsIt(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	projects := &projectReaderStub{project: &projectdomain.Project{Reference: "projects/42/reference.png"}}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{
		Projects: projects, References: references,
	})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42,
		Kind:      generator.GenerateCharacterProtoType,
	})
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	var payload generator.CreateCharacterPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if projects.calls != 1 || payload.Reference != "uploads/generated-1.png" ||
		len(references.persisted) != 1 || references.persisted[0] != "projects/42/reference.png" {
		t.Fatalf("project reference was not used: calls=%d payload=%+v persisted=%v", projects.calls, payload, references.persisted)
	}
}

func TestCreateDoesNotPublishWhenReferencePersistenceFails(t *testing.T) {
	wantErr := errors.New("storage unavailable")
	tasks := &taskManagerStub{createID: 17}
	references := &referenceStoreStub{persistErr: wantErr}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		Kind:       generator.GenerateObjectProtoType,
		Parameters: json.RawMessage(`{"reference":"https://cdn.example.com/reference.png"}`),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
	if tasks.createdTask != nil {
		t.Fatalf("task was published after persistence failure: %+v", tasks.createdTask)
	}
}

func TestGetProjectsTaskAsRun(t *testing.T) {
	assetID := uint(9)
	payload, err := json.Marshal(generator.CreateAnimationPayload{
		ProjectID: 42,
		AssetID:   assetID,
	})
	if err != nil {
		t.Fatalf("encode task payload: %v", err)
	}
	tasks := &taskManagerStub{detail: &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateAnimation),
		Status:  taskdomain.StatusProcessing,
		Payload: payload,
		Result:  json.RawMessage(`{"asset_id":9}`),
	}}
	engine := generator.NewEngine(tasks, nil)

	run, err := engine.Get(context.Background(), 17)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if run.ID != 17 || run.ProjectID != 42 || run.AssetID == nil || *run.AssetID != assetID ||
		run.Kind != generator.GenerateAnimation || run.Status != taskdomain.StatusProcessing {
		t.Fatalf("unexpected generation run: %+v", run)
	}
}

func TestListBuildsProjectScopeTaskFilter(t *testing.T) {
	tasks := &taskManagerStub{listedTasks: []*taskdomain.Task{
		{
			ID:      17,
			Type:    string(generator.GenerateCharacterProtoType),
			Status:  taskdomain.StatusPending,
			Payload: json.RawMessage(`{"project_id":42}`),
		},
		{
			ID:      16,
			Type:    string(generator.GenerateObjectProtoType),
			Status:  taskdomain.StatusPending,
			Payload: json.RawMessage(`{"project_id":99}`),
		},
	}}
	engine := generator.NewEngine(tasks, nil)

	page, err := engine.List(context.Background(), &generator.RunListQuery{
		ProjectID: 42,
		Status:    generator.RunListStatusActive,
		Limit:     10,
		Cursor:    "18",
	})
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if tasks.listFilter == nil || tasks.listFilter.BeforeID != 18 {
		t.Fatalf("unexpected task filter: %+v", tasks.listFilter)
	}
	if !reflect.DeepEqual(tasks.listFilter.Statuses, generator.ActiveTaskStatuses()) {
		t.Fatalf("unexpected statuses: %v", tasks.listFilter.Statuses)
	}
	wantTypes := make([]string, 0, len(generator.ProjectLevelTaskTypes()))
	for _, taskType := range generator.ProjectLevelTaskTypes() {
		wantTypes = append(wantTypes, string(taskType))
	}
	if !reflect.DeepEqual(tasks.listFilter.Types, wantTypes) {
		t.Fatalf("unexpected project task types: %v", tasks.listFilter.Types)
	}
	if len(page.Runs) != 1 || page.Runs[0].ID != 17 || page.Runs[0].ProjectID != 42 {
		t.Fatalf("unexpected project runs: %+v", page)
	}
}

func TestListBuildsAssetScopeTaskFilter(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{listedTasks: []*taskdomain.Task{
		{
			ID:      17,
			Type:    string(generator.GenerateAnimation),
			Status:  taskdomain.StatusProcessing,
			Payload: json.RawMessage(`{"project_id":42,"parent_id":9}`),
		},
		{
			ID:      16,
			Type:    string(generator.GenerateAnimation),
			Status:  taskdomain.StatusPending,
			Payload: json.RawMessage(`{"project_id":42,"parent_id":10}`),
		},
	}}
	engine := generator.NewEngine(tasks, nil)

	page, err := engine.List(context.Background(), &generator.RunListQuery{
		ProjectID: 42,
		AssetID:   &assetID,
	})
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if tasks.listFilter == nil {
		t.Fatal("expected task list filter")
	}
	for _, projectType := range generator.ProjectLevelTaskTypes() {
		for _, taskType := range tasks.listFilter.Types {
			if taskType == string(projectType) {
				t.Fatalf("project task type %q was not excluded", projectType)
			}
		}
	}
	if len(page.Runs) != 1 || page.Runs[0].AssetID == nil || *page.Runs[0].AssetID != assetID {
		t.Fatalf("unexpected asset runs: %+v", page)
	}
}

func TestListPaginatesTaskBackedRuns(t *testing.T) {
	tasks := &taskManagerStub{listedTasks: []*taskdomain.Task{
		{ID: 17, Type: string(generator.GenerateCharacterProtoType), Status: taskdomain.StatusPending, Payload: json.RawMessage(`{"project_id":42}`)},
		{ID: 16, Type: string(generator.GenerateObjectProtoType), Status: taskdomain.StatusPending, Payload: json.RawMessage(`{"project_id":42}`)},
	}}
	engine := generator.NewEngine(tasks, nil)

	page, err := engine.List(context.Background(), &generator.RunListQuery{ProjectID: 42, Limit: 1})
	if err != nil {
		t.Fatalf("list generation runs: %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].ID != 17 || page.NextCursor != "17" {
		t.Fatalf("unexpected generation page: %+v", page)
	}
}

func TestListRejectsUnsupportedStatus(t *testing.T) {
	engine := generator.NewEngine(&taskManagerStub{}, nil)
	_, err := engine.List(context.Background(), &generator.RunListQuery{Status: "completed"})
	if !errors.Is(err, generator.ErrInvalidRunListStatus) {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestListRejectsInvalidCursor(t *testing.T) {
	engine := generator.NewEngine(&taskManagerStub{}, nil)
	_, err := engine.List(context.Background(), &generator.RunListQuery{Cursor: "invalid"})
	if !errors.Is(err, generator.ErrInvalidRunListCursor) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestCancelUpdatesTaskStatus(t *testing.T) {
	tasks := &taskManagerStub{}
	engine := generator.NewEngine(tasks, nil)

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
	calls    int
}

func (s *executorStub) Generate(
	_ context.Context,
	taskType generator.TaskType,
	payload json.RawMessage,
) (json.RawMessage, error) {
	s.calls++
	s.taskType = taskType
	s.payload = append(json.RawMessage(nil), payload...)
	return s.result, s.err
}

func TestRegisteredGeneratorTaskHandlersDecodeTheirPayloads(t *testing.T) {
	tests := []struct {
		taskType generator.TaskType
		payload  json.RawMessage
	}{
		{
			taskType: generator.GenerateCharacterProtoType,
			payload:  json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","dimensions":{"width":64,"height":64},"perspective":"Top-Down","reference":"media-1","project_id":11}`),
		},
		{
			taskType: generator.EditCharacterProtoType,
			payload:  json.RawMessage(`{"asset_id":7,"project_id":11,"edit_instructions":"make the cape blue"}`),
		},
		{
			taskType: generator.GenerateAnimation,
			payload:  json.RawMessage(`{"animation_name":"walk","project_id":11,"asset_id":7,"creative_brief":"walking cycle"}`),
		},
		{
			taskType: generator.GenerateObjectProtoType,
			payload:  json.RawMessage(`{"asset_name":"chest","creative_brief":"wooden chest","dimensions":{"width":64,"height":64},"perspective":"Isometric","reference":"media-2","project_id":11}`),
		},
		{
			taskType: generator.GenerateAnimation,
			payload:  json.RawMessage(`{"animation_name":"open chest","project_id":11,"asset_id":8,"creative_brief":"opening animation"}`),
		},
		{
			taskType: generator.GenerateTileSet,
			payload:  json.RawMessage(`{"asset_name":"forest","project_id":11,"creative_brief":"forest ground","tile_num":2,"tile_descriptions":["grass","path"],"reference":"media-3"}`),
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			tasks := &taskManagerStub{}
			executor := &executorStub{result: json.RawMessage(`{"asset_id":7}`)}
			generator.NewEngine(tasks, executor)

			message := &taskdomain.Task{ID: 17, Type: string(tt.taskType), Payload: tt.payload}
			result, err := tasks.dispatch(context.Background(), message)
			if err != nil {
				t.Fatalf("dispatch generation task: %v", err)
			}
			shouldExecute := tt.taskType != generator.GenerateTileSet
			if shouldExecute {
				if executor.calls != 1 || executor.taskType != tt.taskType ||
					!reflect.DeepEqual(executor.payload, tt.payload) ||
					!reflect.DeepEqual(result, executor.result) {
					t.Fatalf("unexpected executor call: calls=%d type=%s payload=%s result=%s",
						executor.calls, executor.taskType, executor.payload, result)
				}
			} else if executor.calls != 0 || result != nil {
				t.Fatalf("tileset handler must remain deferred: calls=%d result=%v", executor.calls, result)
			}
			if len(tasks.statusUpdates) != 0 {
				t.Fatalf("task queue owns status updates, got %+v", tasks.statusUpdates)
			}
		})
	}
}

func TestRegisteredGeneratorTaskHandlerRejectsMismatchedPayload(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, executor)

	_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateCharacterProtoType),
		Payload: json.RawMessage(`{"project_id":"not-a-number"}`),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
	if executor.calls != 0 || len(tasks.statusUpdates) != 0 {
		t.Fatalf("malformed task must not be processed: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
	}
}

func TestRegisteredEditCharacterPrototypeHandlerRejectsMismatchedPayload(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, executor)

	_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      18,
		Type:    string(generator.EditCharacterProtoType),
		Payload: json.RawMessage(`{"asset_id":"not-a-number"}`),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
	if executor.calls != 0 || len(tasks.statusUpdates) != 0 {
		t.Fatalf("malformed edit task must not be processed: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
	}
}

func TestNewEngineRegistersAllTaskTypes(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, executor)

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
	if executor.calls != 4 || len(tasks.statusUpdates) != 0 {
		t.Fatalf("expected four implemented handler calls: calls=%d statuses=%+v",
			executor.calls, tasks.statusUpdates)
	}
}

func TestHandleCharacterPrototypeReturnsExecutorResult(t *testing.T) {
	payload := json.RawMessage(`{"asset_name":"hero","creative_brief":"pixel knight","dimensions":{"width":64,"height":64},"perspective":"Top-Down","reference":"media-1","project_id":42}`)
	tasks := &taskManagerStub{}
	executor := &executorStub{result: json.RawMessage(`{"asset_id":23}`)}
	generator.NewEngine(tasks, executor)

	got, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateCharacterProtoType),
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("handle generation task: %v", err)
	}
	if !reflect.DeepEqual(got, executor.result) {
		t.Fatalf("unexpected handler result: %s", got)
	}
	if executor.calls != 1 || executor.taskType != generator.GenerateCharacterProtoType ||
		!reflect.DeepEqual(executor.payload, payload) || len(tasks.statusUpdates) != 0 {
		t.Fatalf("unexpected handler execution: calls=%d type=%s payload=%s statuses=%+v",
			executor.calls, executor.taskType, executor.payload, tasks.statusUpdates)
	}
}

func TestImplementedHandlerRequiresExecutor(t *testing.T) {
	tasks := &taskManagerStub{}
	generator.NewEngine(tasks, nil)

	_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      17,
		Type:    string(generator.GenerateAnimation),
		Payload: json.RawMessage(`{"animation_name":"open","asset_id":8}`),
	})
	if !errors.Is(err, generator.ErrExecutorRequired) {
		t.Fatalf("expected executor required error, got %v", err)
	}
}
