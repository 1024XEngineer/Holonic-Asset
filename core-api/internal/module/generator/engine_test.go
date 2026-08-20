package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
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
	return fmt.Sprintf("uploads/generated-%d.png", len(s.persisted)), nil
}

func (s *referenceStoreStub) NewObjectKey(string) (string, error) {
	return "uploads/generated.png", nil
}

func (s *referenceStoreStub) PersistReferenceAt(context.Context, string, string) error {
	return nil
}

func (s *referenceStoreStub) DeleteObjects(context.Context, []string) error {
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

func TestCreateBuildsUnifiedEditFramesPayload(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditFrames,
		CreativeBrief: "make the stride longer",
		Parameters:    json.RawMessage(`{"animationId":3,"frameIds":[5,7]}`),
	})
	if err != nil {
		t.Fatalf("create edit frames: %v", err)
	}
	var payload generator.EditFramesPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode edit frames payload: %v", err)
	}
	want := generator.EditFramesPayload{
		AssetID: 9, ProjectID: 42, AnimationID: 3,
		FrameIDs: []uint{5, 7}, Prompt: "make the stride longer",
	}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("unexpected edit frames payload: got %+v want %+v", payload, want)
	}
}

func TestCreateEditFramesValidatesParameters(t *testing.T) {
	assetID := uint(9)
	tests := []struct {
		name       string
		parameters json.RawMessage
		want       string
		noBrief    bool
	}{
		{name: "missing animation", parameters: json.RawMessage(`{"frameIds":[1]}`), want: "animation id is required"},
		{name: "missing frames", parameters: json.RawMessage(`{"animationId":3}`), want: "frame ids are required"},
		{name: "invalid parameters", parameters: json.RawMessage(`{"animationId":`), want: "decode edit_frames parameters"},
		{name: "missing creative brief", parameters: json.RawMessage(`{"animationId":3,"frameIds":[1]}`), want: "creative brief is required", noBrief: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tasks := &taskManagerStub{createID: 17}
			engine := generator.NewEngine(tasks, nil)
			brief := "change pose"
			if test.noBrief {
				brief = "   "
			}
			_, err := engine.Create(context.Background(), &generator.Request{
				ProjectID: 42, AssetID: &assetID, Kind: generator.EditFrames,
				CreativeBrief: brief, Parameters: test.parameters,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if tasks.createdTask != nil {
				t.Fatalf("invalid edit parameters published task: %+v", tasks.createdTask)
			}
		})
	}
}

func TestCreateEditFramesUsesCreativeBriefAsPrompt(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)
	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42, AssetID: &assetID, Kind: generator.EditFrames,
		CreativeBrief: "make the stride longer",
		Parameters:    json.RawMessage(`{"animationId":3,"frameIds":[1]}`),
	})
	if err != nil {
		t.Fatalf("create edit frames: %v", err)
	}
	var payload generator.EditFramesPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Prompt != "make the stride longer" {
		t.Fatalf("creative brief not mapped: %+v", payload)
	}
}

func TestCreateDerivesAnimationStyleAndRejectsRemovedParameters(t *testing.T) {
	assetID := uint(9)

	t.Run("inherits project style", func(t *testing.T) {
		tasks := &taskManagerStub{createID: 17}
		projects := &projectReaderStub{project: &projectdomain.Project{Style: " clean pixel art "}}
		engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{Projects: projects})

		_, err := engine.Create(context.Background(), &generator.Request{
			ProjectID:     42,
			AssetID:       &assetID,
			Kind:          generator.GenerateAnimation,
			CreativeBrief: "walk",
			Parameters:    json.RawMessage(`{"animation_name":"hero walk","direction":"front","frame_count":10}`),
		})
		if err != nil {
			t.Fatalf("create generation: %v", err)
		}
		var payload generator.CreateAnimationPayload
		if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Style != " clean pixel art " || projects.calls != 1 {
			t.Fatalf("animation style was not inherited: payload=%+v calls=%d", payload, projects.calls)
		}
	})

	for _, field := range []string{"style", "columns", "frame_width", "frame_height", "aspect_ratio"} {
		t.Run("rejects "+field, func(t *testing.T) {
			tasks := &taskManagerStub{createID: 17}
			engine := generator.NewEngine(tasks, nil)
			parameters := fmt.Sprintf(`{"animation_name":"hero walk","direction":"front","%s":1}`, field)
			_, err := engine.Create(context.Background(), &generator.Request{
				ProjectID:     42,
				AssetID:       &assetID,
				Kind:          generator.GenerateAnimation,
				CreativeBrief: "walk",
				Parameters:    json.RawMessage(parameters),
			})
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("expected removed parameter %q to be rejected, got %v", field, err)
			}
			if tasks.createdTask != nil {
				t.Fatalf("task published with removed parameter %q", field)
			}
		})
	}
}

func TestCreateBuildsEditAnimationPayload(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	runID, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditAnimation,
		CreativeBrief: "attack with sword",
		Parameters: json.RawMessage(
			`{"asset_id":99,"animation_id":3,"project_id":99,"creative_brief":"ignore me"}`,
		),
	})
	if err != nil {
		t.Fatalf("create animation edit: %v", err)
	}
	if runID != 17 || tasks.createdTask == nil || tasks.createdTask.Type != string(generator.EditAnimation) {
		t.Fatalf("unexpected edit animation task: run=%d task=%+v", runID, tasks.createdTask)
	}

	var payload generator.EditAnimationPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode edit animation task payload: %v", err)
	}
	if payload.AssetID != assetID || payload.AnimationID != 3 || payload.ProjectID != 42 ||
		payload.CreativeBrief != "attack with sword" {
		t.Fatalf("unexpected edit animation payload: %+v", payload)
	}
}

func TestCreateEditAnimationRequiresAssetAndAnimationIDs(t *testing.T) {
	tests := []struct {
		name       string
		assetID    *uint
		parameters json.RawMessage
		want       string
	}{
		{
			name:       "asset id",
			parameters: json.RawMessage(`{"animation_id":3}`),
			want:       "asset id is required",
		},
		{
			name:       "animation id",
			assetID:    func() *uint { value := uint(9); return &value }(),
			parameters: json.RawMessage(`{}`),
			want:       "animation id is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := &taskManagerStub{createID: 17}
			engine := generator.NewEngine(tasks, nil)
			_, err := engine.Create(context.Background(), &generator.Request{
				ProjectID:     42,
				AssetID:       tt.assetID,
				Kind:          generator.EditAnimation,
				CreativeBrief: "attack with sword",
				Parameters:    tt.parameters,
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
			if tasks.createdTask != nil {
				t.Fatalf("task published with invalid edit animation identity: %+v", tasks.createdTask)
			}
		})
	}
}

func TestCreateEditAnimationDoesNotPrepareProjectReference(t *testing.T) {
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
		Kind:          generator.EditAnimation,
		CreativeBrief: "attack with sword",
		Parameters:    json.RawMessage(`{"animation_id":3}`),
	})
	if err != nil {
		t.Fatalf("create animation edit: %v", err)
	}
	if projects.calls != 0 || len(references.persisted) != 0 {
		t.Fatalf("animation edit prepared an extra reference: project_calls=%d persisted=%v", projects.calls, references.persisted)
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
			`{"asset_name":"knight","creative_brief":"incorrect parameter brief","dimensions":{"width":64,"height":64},"perspective":"Top-Down","project_reference":"client-controlled.png"}`,
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
		payload.CreativeBrief != "hero" || payload.Reference != "" || payload.ProjectReference != "" ||
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

func TestCreateBuildsEditObjectPrototypePayload(t *testing.T) {
	assetID := uint(9)
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditObjectProtoType,
		CreativeBrief: "change only the lock to gold",
		Parameters: json.RawMessage(
			`{"asset_id":99,"project_id":99,"edit_instructions":"ignore me"}`,
		),
	})
	if err != nil {
		t.Fatalf("create object edit: %v", err)
	}

	var payload generator.EditObjectPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.AssetID != assetID || payload.ProjectID != 42 ||
		payload.EditInstructions != "change only the lock to gold" {
		t.Fatalf("unexpected object edit payload: %+v", payload)
	}
}

func TestCreateEditObjectPrototypeDoesNotPrepareProjectReference(t *testing.T) {
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
		Kind:          generator.EditObjectProtoType,
		CreativeBrief: "change only the lock to gold",
	})
	if err != nil {
		t.Fatalf("create object edit: %v", err)
	}
	if projects.calls != 0 || len(references.persisted) != 0 {
		t.Fatalf("object edit prepared an extra reference: project_calls=%d persisted=%v", projects.calls, references.persisted)
	}
}

func TestCreateEditObjectPrototypeRequiresAssetID(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		Kind:          generator.EditObjectProtoType,
		CreativeBrief: "change only the lock to gold",
	})
	if err == nil {
		t.Fatal("expected missing asset id error")
	}
	if tasks.createdTask != nil {
		t.Fatalf("task published without asset id: %+v", tasks.createdTask)
	}
}

func TestCreatePersistsProjectAndUserReferencesBeforePublishing(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	projects := &projectReaderStub{project: &projectdomain.Project{Reference: "projects/42/style.png"}}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{
		Projects: projects, References: references,
	})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42,
		Kind:      generator.GenerateObjectProtoType,
		Parameters: json.RawMessage(`{
			"reference":"https://cdn.example.com/user/object.png"
		}`),
	})
	if err != nil {
		t.Fatalf("create generation: %v", err)
	}
	var payload generator.CreateObjectPrototypePayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if projects.calls != 1 || payload.ProjectReference != "uploads/generated-1.png" ||
		payload.Reference != "uploads/generated-2.png" ||
		!reflect.DeepEqual(references.persisted, []string{
			"projects/42/style.png",
			"https://cdn.example.com/user/object.png",
		}) {
		t.Fatalf(
			"references were not persisted independently before publish: calls=%d payload=%+v persisted=%v",
			projects.calls,
			payload,
			references.persisted,
		)
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
	if projects.calls != 1 || payload.Reference != "" || payload.ProjectReference != "uploads/generated-1.png" ||
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

func TestCreateRejectsExternalReferenceBeforePublishing(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	references := &referenceStoreStub{persistErr: upload.ErrUntrustedReference}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		Kind: generator.GenerateObjectProtoType,
		Parameters: json.RawMessage(
			`{"reference":"https://attacker.example/reference.png"}`,
		),
	})
	if !errors.Is(err, generator.ErrInvalidTaskPayload) ||
		!strings.Contains(err.Error(), "configured object-storage URL") {
		t.Fatalf("create error = %v, want invalid managed reference", err)
	}
	if tasks.createdTask != nil {
		t.Fatalf("task was published with an external reference: %+v", tasks.createdTask)
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
		{
			ID:      15,
			Type:    string(generator.GenerateCharacterProtoType),
			Status:  taskdomain.StatusFailed,
			Payload: json.RawMessage(`{"project_id":42}`),
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
	if len(page.Runs) != 2 || page.Runs[0].ID != 17 || page.Runs[0].ProjectID != 42 ||
		page.Runs[1].ID != 15 || page.Runs[1].Status != taskdomain.StatusFailed {
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
			taskType: generator.EditObjectProtoType,
			payload:  json.RawMessage(`{"asset_id":8,"project_id":11,"edit_instructions":"change only the lock"}`),
		},
		{
			taskType: generator.EditAnimation,
			payload:  json.RawMessage(`{"asset_id":8,"animation_id":3,"project_id":11,"creative_brief":"opening animation"}`),
		},
		{
			taskType: generator.GenerateTileSet,
			payload: json.RawMessage(`{
				"asset_name":"forest",
				"project_id":11,
				"creative_brief":"forest ground",
				"dimensions":{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":8,"rows":8}},
				"items":[{"name":"grass","description":"grass edge","shape":[[0,0],[1,0]]}]
			}`),
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
			shouldExecute := true
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

func TestRegisteredEditObjectPrototypeHandlerRejectsMismatchedPayload(t *testing.T) {
	tasks := &taskManagerStub{}
	executor := &executorStub{}
	generator.NewEngine(tasks, executor)

	_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
		ID:      19,
		Type:    string(generator.EditObjectProtoType),
		Payload: json.RawMessage(`{"asset_id":"not-a-number"}`),
	})
	if err == nil {
		t.Fatal("expected payload decode error")
	}
	if executor.calls != 0 || len(tasks.statusUpdates) != 0 {
		t.Fatalf("malformed object edit task must not be processed: payload=%s statuses=%+v",
			executor.payload, tasks.statusUpdates)
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
		payload := json.RawMessage(`{}`)
		switch taskType {
		case generator.GenerateTileSet:
			payload = json.RawMessage(`{
				"asset_name":"forest","project_id":11,"creative_brief":"forest ground",
				"dimensions":{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":8,"rows":8}},
				"items":[{"name":"grass","description":"grass edge","shape":[[0,0]]}]
			}`)
		case generator.EditTilesetItem:
			payload = json.RawMessage(`{
				"asset_id":7,"project_id":11,"creative_brief":"brighter",
				"target":{"position":{"x":2,"y":3}}
			}`)
		case generator.EditTiles:
			payload = json.RawMessage(`{
				"asset_id":7,"project_id":11,"creative_brief":"add moss",
				"targets":[{"position":{"x":2,"y":3}}]
			}`)
		}
		message := &taskdomain.Task{
			ID:      uint(len(tasks.statusUpdates) + 1),
			Type:    string(taskType),
			Payload: payload,
		}
		if _, err := tasks.dispatch(context.Background(), message); err != nil {
			t.Fatalf("dispatch task type %q: %v", taskType, err)
		}
	}
	if executor.calls != 11 || len(tasks.statusUpdates) != 0 {
		t.Fatalf("expected eleven implemented handler calls: calls=%d statuses=%+v",
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

func TestCreateBuildsCompleteTileSetPayload(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{References: references})

	runID, err := engine.Create(context.Background(), validTileSetRequest())
	if err != nil {
		t.Fatalf("create Tileset generation: %v", err)
	}
	if runID != 17 || tasks.createdTask == nil {
		t.Fatalf("unexpected task creation: run=%d task=%+v", runID, tasks.createdTask)
	}

	var payload generator.CreateTileSetPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.ProjectID != 42 || payload.AssetName != "Forest Terrain" ||
		payload.CreativeBrief != "A compact forest terrain set" ||
		payload.Dimensions.TileSize.Width != 16 || payload.Dimensions.TileAmount.Columns != 16 ||
		len(payload.Items) != 2 || payload.Items[0].Name != "Grass edge" ||
		!reflect.DeepEqual(payload.Items[0].Shape, []generator.TileSetCoordinate{{0, 0}, {1, 0}}) {
		t.Fatalf("unexpected Tileset payload: %+v", payload)
	}
	if len(references.persisted) != 0 {
		t.Fatalf("Tileset creation must resolve the Project reference during execution: %v", references.persisted)
	}
}

func TestCreateBuildsCompleteTilesetEditingPayloads(t *testing.T) {
	tests := []struct {
		name        string
		kind        generator.TaskType
		parameters  json.RawMessage
		wantX       int
		wantY       int
		wantRef     string
		decodeAsset func(*testing.T, json.RawMessage) (uint, uint, string, int, int, string)
	}{
		{
			name:       "complete Item with edit reference",
			kind:       generator.EditTilesetItem,
			parameters: json.RawMessage(`{"target":{"position":{"x":2,"y":3}},"reference":"https://cdn.example/edit.png"}`),
			wantX:      2,
			wantY:      3,
			wantRef:    "uploads/generated-1.png",
			decodeAsset: func(t *testing.T, raw json.RawMessage) (uint, uint, string, int, int, string) {
				t.Helper()
				var payload generator.EditTilesetItemPayload
				if err := json.Unmarshal(raw, &payload); err != nil {
					t.Fatalf("decode Item edit payload: %v", err)
				}
				return payload.ProjectID, payload.AssetID, payload.CreativeBrief,
					*payload.Target.Position.X, *payload.Target.Position.Y, payload.Reference
			},
		},
		{
			name:       "Tile batch without edit reference",
			kind:       generator.EditTiles,
			parameters: json.RawMessage(`{"targets":[{"position":{"x":4,"y":5}}]}`),
			wantX:      4,
			wantY:      5,
			decodeAsset: func(t *testing.T, raw json.RawMessage) (uint, uint, string, int, int, string) {
				t.Helper()
				var payload generator.EditTilesPayload
				if err := json.Unmarshal(raw, &payload); err != nil {
					t.Fatalf("decode Tile edit payload: %v", err)
				}
				return payload.ProjectID, payload.AssetID, payload.CreativeBrief,
					*payload.Targets[0].Position.X, *payload.Targets[0].Position.Y, payload.Reference
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assetID := uint(100)
			tasks := &taskManagerStub{createID: 17}
			projects := &projectReaderStub{project: &projectdomain.Project{Reference: "projects/42/reference.png"}}
			references := &referenceStoreStub{}
			engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{
				Projects: projects, References: references,
			})
			_, err := engine.Create(context.Background(), &generator.Request{
				ProjectID:     42,
				AssetID:       &assetID,
				Kind:          test.kind,
				CreativeBrief: "Make the target brighter",
				Parameters:    test.parameters,
			})
			if err != nil {
				t.Fatalf("create edit task: %v", err)
			}

			projectID, gotAssetID, brief, x, y, reference := test.decodeAsset(t, tasks.createdTask.Payload)
			if projectID != 42 || gotAssetID != assetID || brief != "Make the target brighter" ||
				x != test.wantX || y != test.wantY || reference != test.wantRef {
				t.Fatalf("editing request was not preserved: project=%d asset=%d brief=%q position=(%d,%d) reference=%q",
					projectID, gotAssetID, brief, x, y, reference)
			}
			if projects.calls != 0 {
				t.Fatalf("edit reference preparation loaded the Project: calls=%d", projects.calls)
			}
			wantPersisted := []string(nil)
			if test.wantRef != "" {
				wantPersisted = []string{"https://cdn.example/edit.png"}
			}
			if !reflect.DeepEqual(references.persisted, wantPersisted) {
				t.Fatalf("unexpected persisted edit references: got=%v want=%v", references.persisted, wantPersisted)
			}
		})
	}
}

func TestCreateDoesNotPublishTilesetEditWhenReferencePersistenceFails(t *testing.T) {
	assetID := uint(100)
	tasks := &taskManagerStub{}
	wantErr := errors.New("storage unavailable")
	references := &referenceStoreStub{persistErr: wantErr}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID:     42,
		AssetID:       &assetID,
		Kind:          generator.EditTiles,
		CreativeBrief: "Make the target brighter",
		Parameters:    json.RawMessage(`{"targets":[{"position":{"x":2,"y":3}}],"reference":"https://cdn.example/edit.png"}`),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected persistence failure, got %v", err)
	}
	if tasks.createdTask != nil {
		t.Fatalf("task was published after edit reference persistence failure: %+v", tasks.createdTask)
	}
}

func TestCreateRejectsInvalidTileSetRequestsBeforePublishing(t *testing.T) {
	assetID := uint(100)
	tests := []struct {
		name    string
		request *generator.Request
	}{
		{"legacy fields", tileSetRequestWithParameters(`{"asset_name":"forest","tile_num":2,"tile_descriptions":["grass"]}`)},
		{"unknown field", tileSetRequestWithParameters(`{"unexpected":true}`)},
		{"request asset", func() *generator.Request {
			request := validTileSetRequest()
			request.AssetID = &assetID
			return request
		}()},
		{"request targets", func() *generator.Request {
			request := validTileSetRequest()
			request.TargetAssetPaths = []string{"items.0"}
			return request
		}()},
		{"missing project", func() *generator.Request { request := validTileSetRequest(); request.ProjectID = 0; return request }()},
		{"blank asset name", tileSetRequestWithParameters(validTileSetParametersWith("asset_name", " "))},
		{"zero dimensions", tileSetRequestWithParameters(validTileSetParametersWith("dimensions", json.RawMessage(`{"tileSize":{"width":0,"height":16},"tileAmount":{"columns":16,"rows":16}}`)))},
		{"oversized tile", tileSetRequestWithParameters(validTileSetParametersWith("dimensions", json.RawMessage(`{"tileSize":{"width":1025,"height":16},"tileAmount":{"columns":4,"rows":4}}`)))},
		{"oversized grid", tileSetRequestWithParameters(validTileSetParametersWith("dimensions", json.RawMessage(`{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":65,"rows":64}}`)))},
		{"missing Item name", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":" ","description":"edge","shape":[[0,0]]}]`)))},
		{"missing Item description", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":" ","shape":[[0,0]]}]`)))},
		{"empty shape", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":"edge","shape":[]}]`)))},
		{"negative shape coordinate", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":"edge","shape":[[-1,0]]}]`)))},
		{"duplicate shape coordinate", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":"edge","shape":[[0,0],[0,0]]}]`)))},
		{"shape outside grid", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":"edge","shape":[[16,0]]}]`)))},
		{"oversized Item image", tileSetRequestWithParameters(`{
			"asset_name":"forest",
			"dimensions":{"tileSize":{"width":1024,"height":16},"tileAmount":{"columns":5,"rows":1}},
			"items":[{"name":"edge","description":"edge","shape":[[0,0],[4,0]]}]
		}`)},
		{"project reference", tileSetRequestWithParameters(validTileSetParametersWith("reference", "https://cdn.example/project.png"))},
		{"Item reference", tileSetRequestWithParameters(validTileSetParametersWith("items", json.RawMessage(`[{"name":"edge","description":"edge","shape":[[0,0]],"reference":"https://cdn.example/item.png"}]`)))},
		{"trailing JSON", tileSetRequestWithParameters(validTileSetParameters() + `{}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tasks := &taskManagerStub{}
			_, err := generator.NewEngine(tasks, nil).Create(context.Background(), test.request)
			if !errors.Is(err, generator.ErrInvalidTaskPayload) {
				t.Fatalf("expected invalid payload error, got %v", err)
			}
			if tasks.createdTask != nil {
				t.Fatalf("invalid request was published: %+v", tasks.createdTask)
			}
		})
	}
}

func TestCreateRejectsInvalidTilesetEditingTargetsBeforePublishing(t *testing.T) {
	assetID := uint(100)
	tests := []struct {
		name       string
		kind       generator.TaskType
		assetID    *uint
		parameters json.RawMessage
		paths      []string
	}{
		{name: "missing asset", kind: generator.EditTilesetItem, parameters: json.RawMessage(`{"target":{"position":{"x":1,"y":2}}}`)},
		{name: "missing Item target", kind: generator.EditTilesetItem, assetID: &assetID, parameters: json.RawMessage(`{}`)},
		{name: "null Item position", kind: generator.EditTilesetItem, assetID: &assetID, parameters: json.RawMessage(`{"target":{"position":null}}`)},
		{name: "incomplete Item position", kind: generator.EditTilesetItem, assetID: &assetID, parameters: json.RawMessage(`{"target":{"position":{"x":1}}}`)},
		{name: "negative Item position", kind: generator.EditTilesetItem, assetID: &assetID, parameters: json.RawMessage(`{"target":{"position":{"x":-1,"y":0}}}`)},
		{name: "Item edit with targets", kind: generator.EditTilesetItem, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1,"y":2}}]}`)},
		{name: "Item edit with legacy path", kind: generator.EditTilesetItem, assetID: &assetID, paths: []string{"items.0"}},
		{name: "missing Tile targets", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{}`)},
		{name: "null Tile position", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":null}]}`)},
		{name: "negative Tile position", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":0,"y":-1}}]}`)},
		{name: "fractional Tile position", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1.5,"y":2}}]}`)},
		{name: "duplicate Tile position", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1,"y":2}},{"position":{"x":1,"y":2}}]}`)},
		{name: "Tile edit with target", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"target":{"position":{"x":1,"y":2}}}`)},
		{name: "Tile edit with legacy path", kind: generator.EditTiles, assetID: &assetID, paths: []string{"items.0.tiles.0"}},
		{name: "blank edit reference", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1,"y":2}}],"reference":" "}`)},
		{name: "control character in edit reference", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1,"y":2}}],"reference":"bad\u0000reference"}`)},
		{name: "oversized edit reference", kind: generator.EditTiles, assetID: &assetID, parameters: json.RawMessage(`{"targets":[{"position":{"x":1,"y":2}}],"reference":"` + strings.Repeat("x", (8<<20)+1) + `"}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tasks := &taskManagerStub{}
			_, err := generator.NewEngine(tasks, nil).Create(context.Background(), &generator.Request{
				ProjectID:        42,
				AssetID:          test.assetID,
				Kind:             test.kind,
				CreativeBrief:    "edit target",
				TargetAssetPaths: test.paths,
				Parameters:       test.parameters,
			})
			if !errors.Is(err, generator.ErrInvalidTaskPayload) {
				t.Fatalf("expected invalid payload error, got %v", err)
			}
			if tasks.createdTask != nil {
				t.Fatalf("invalid edit was published: %+v", tasks.createdTask)
			}
		})
	}
}

func TestTilesetTaskHandlersRejectInvalidQueuedPayloads(t *testing.T) {
	tests := []struct {
		kind    generator.TaskType
		payload json.RawMessage
	}{
		{
			kind:    generator.GenerateTileSet,
			payload: json.RawMessage(`{"asset_name":"legacy","tile_num":1}`),
		},
		{
			kind: generator.EditTilesetItem,
			payload: json.RawMessage(`{
				"asset_id":100,"project_id":42,"creative_brief":"edit",
				"target":{"position":{"x":-1,"y":0}}
			}`),
		},
		{
			kind: generator.EditTiles,
			payload: json.RawMessage(`{
				"asset_id":100,"project_id":42,"creative_brief":"edit",
				"targets":[{"position":{"x":1,"y":2}},{"position":{"x":1,"y":2}}]
			}`),
		},
	}

	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			tasks := &taskManagerStub{}
			executor := &executorStub{}
			generator.NewEngine(tasks, executor)
			_, err := tasks.dispatch(context.Background(), &taskdomain.Task{
				ID: 17, Type: string(test.kind), Payload: test.payload,
			})
			if !errors.Is(err, generator.ErrInvalidTaskPayload) {
				t.Fatalf("expected invalid queued payload error, got %v", err)
			}
			if executor.calls != 0 {
				t.Fatalf("invalid queued payload reached executor: %d calls", executor.calls)
			}
		})
	}
}

func TestTilesetEditingTaskProjectsAssetScope(t *testing.T) {
	tasks := &taskManagerStub{detail: &taskdomain.Task{
		ID:      17,
		Type:    string(generator.EditTiles),
		Status:  taskdomain.StatusPending,
		Payload: json.RawMessage(`{"project_id":42,"asset_id":100}`),
	}}
	run, err := generator.NewEngine(tasks, nil).Get(context.Background(), 17)
	if err != nil {
		t.Fatalf("get editing run: %v", err)
	}
	if run.ProjectID != 42 || run.AssetID == nil || *run.AssetID != 100 {
		t.Fatalf("unexpected editing run scope: %+v", run)
	}
}

func validTileSetRequest() *generator.Request {
	return &generator.Request{
		ProjectID:     42,
		Kind:          generator.GenerateTileSet,
		CreativeBrief: "A compact forest terrain set",
		Parameters:    json.RawMessage(validTileSetParameters()),
	}
}

func tileSetRequestWithParameters(parameters string) *generator.Request {
	request := validTileSetRequest()
	request.Parameters = json.RawMessage(parameters)
	return request
}

func validTileSetParameters() string {
	return `{
		"asset_name":"Forest Terrain",
		"dimensions":{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":16,"rows":16}},
		"items":[
			{"name":"Grass edge","description":"A seamless grass edge","shape":[[0,0],[1,0]]},
			{"name":"Dirt","description":"A dirt Tile","shape":[[0,0]]}
		]
	}`
}

func validTileSetParametersWith(field string, value any) string {
	var parameters map[string]any
	if err := json.Unmarshal([]byte(validTileSetParameters()), &parameters); err != nil {
		panic(err)
	}
	parameters[field] = value
	encoded, err := json.Marshal(parameters)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func TestCreateEditFramesRequiresAssetID(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	engine := generator.NewEngine(tasks, nil)
	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42, Kind: generator.EditFrames, CreativeBrief: "change pose",
		Parameters: json.RawMessage(`{"animationId":3,"frameIds":[1]}`),
	})
	if err == nil || !strings.Contains(err.Error(), "asset id is required for edit_frames") {
		t.Fatalf("expected missing asset error, got %v", err)
	}
	if tasks.createdTask != nil {
		t.Fatalf("missing asset published task: %+v", tasks.createdTask)
	}
}
