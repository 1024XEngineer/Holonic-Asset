package generator

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type uiSetTestLLM struct {
	result *llmclient.CompletionResult
	err    error
}

func (s *uiSetTestLLM) Complete(
	context.Context,
	*llmclient.CompletionRequest,
) (*llmclient.CompletionResult, error) {
	return s.result, s.err
}

type uiSetTestProjectReader struct {
	project *projectdomain.Project
	err     error
}

func (s *uiSetTestProjectReader) GetDetail(context.Context, uint) (*projectdomain.Project, error) {
	return s.project, s.err
}

type uiSetTestReferenceStore struct{ err error }

func (*uiSetTestReferenceStore) ResolveReference(context.Context, string) (string, error) {
	return "", nil
}

func (s *uiSetTestReferenceStore) PersistReference(context.Context, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "uploads/reference.png", nil
}

func (*uiSetTestReferenceStore) NewObjectKey(string) (string, error) { return "", nil }

func (*uiSetTestReferenceStore) PersistReferenceAt(context.Context, string, string) error {
	return nil
}

func (*uiSetTestReferenceStore) DeleteObjects(context.Context, []string) error { return nil }

func TestUISetExecutorErrorBranches(t *testing.T) {
	payload := validUISetPayload()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&executor{}).Generate(context.Background(), GenerateUISet, encoded); !errors.Is(err, ErrLLMServiceRequired) {
		t.Fatalf("expected missing LLM error, got %v", err)
	}
	executor := &executor{llm: &uiSetTestLLM{}}
	for _, test := range []struct {
		name    string
		kind    TaskType
		payload json.RawMessage
	}{
		{name: "generation JSON", kind: GenerateUISet, payload: json.RawMessage(`{`)},
		{name: "generation validation", kind: GenerateUISet, payload: json.RawMessage(`{}`)},
		{name: "edit JSON", kind: EditUISetComponents, payload: json.RawMessage(`{`)},
		{name: "edit validation", kind: EditUISetComponents, payload: json.RawMessage(`{}`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := executor.Generate(context.Background(), test.kind, test.payload); err == nil {
				t.Fatal("expected dispatch error")
			}
		})
	}
	if _, err := executor.Generate(context.Background(), GenerateUISet, encoded); !errors.Is(err, ErrInvalidUISetPlan) {
		t.Fatalf("expected nil-completion plan error, got %v", err)
	}
	wantErr := errors.New("planning failed")
	executor.llm = &uiSetTestLLM{err: wantErr}
	if _, err := executor.generateUISet(context.Background(), payload); !errors.Is(err, wantErr) {
		t.Fatalf("expected generation to preserve planning failure, got %v", err)
	}
}

func TestUISetPreparationErrorBranches(t *testing.T) {
	payload := validUISetPayload()
	if _, err := (&Engine{}).prepareTaskPayload(context.Background(), payload.ProjectID, payload); !errors.Is(err, ErrProjectReaderRequired) {
		t.Fatalf("expected missing Project reader error, got %v", err)
	}
	wantErr := errors.New("project unavailable")
	engine := &Engine{projects: &uiSetTestProjectReader{err: wantErr}}
	if _, err := engine.prepareTaskPayload(context.Background(), payload.ProjectID, payload); !errors.Is(err, wantErr) {
		t.Fatalf("expected Project error, got %v", err)
	}
	engine.projects = &uiSetTestProjectReader{}
	if _, err := engine.prepareTaskPayload(context.Background(), payload.ProjectID, payload); err == nil || !strings.Contains(err.Error(), "empty result") {
		t.Fatalf("expected empty Project error, got %v", err)
	}
	engine.projects = &uiSetTestProjectReader{project: &projectdomain.Project{Name: "Moon Forge"}}
	engine.references = &uiSetTestReferenceStore{err: wantErr}
	payload.Reference = "https://cdn.example/ui.png"
	if _, err := engine.prepareTaskPayload(context.Background(), payload.ProjectID, payload); !errors.Is(err, wantErr) {
		t.Fatalf("expected generation reference error, got %v", err)
	}
	edit := EditUISetComponentsPayload{
		AssetID: 7, ProjectID: payload.ProjectID, CreativeBrief: "brighter",
		TargetAssetPaths: []string{"components.0"}, Reference: "https://cdn.example/edit.png",
	}
	if _, err := engine.prepareTaskPayload(context.Background(), payload.ProjectID, edit); !errors.Is(err, wantErr) {
		t.Fatalf("expected edit reference error, got %v", err)
	}
}

func TestUISetValidationErrorBranches(t *testing.T) {
	if err := validateCreateUISetPayload(nil); !errors.Is(err, ErrInvalidTaskPayload) {
		t.Fatalf("expected nil generation payload error, got %v", err)
	}
	if err := validateEditUISetComponentsPayload(nil); !errors.Is(err, ErrInvalidTaskPayload) {
		t.Fatalf("expected nil edit payload error, got %v", err)
	}
	payload := validUISetPayload()
	payload.Components = make([]UISetComponentDefinition, maxUISetComponents+1)
	if err := validateCreateUISetPayload(&payload); !errors.Is(err, ErrInvalidTaskPayload) {
		t.Fatalf("expected Component limit error, got %v", err)
	}
	for _, edit := range []EditUISetComponentsPayload{
		{AssetID: 7, CreativeBrief: "edit", TargetAssetPaths: []string{"components.0"}},
		{ProjectID: 42, CreativeBrief: "edit", TargetAssetPaths: []string{"components.0"}},
		{ProjectID: 42, AssetID: 7, TargetAssetPaths: []string{"components.0"}},
		{ProjectID: 42, AssetID: 7, CreativeBrief: "edit", TargetAssetPaths: []string{"components.0"}, Reference: " "},
	} {
		value := edit
		if err := validateEditUISetComponentsPayload(&value); !errors.Is(err, ErrInvalidTaskPayload) {
			t.Fatalf("expected edit validation error for %+v, got %v", edit, err)
		}
	}
	definitions := validUISetPayload().Components
	canvas := assetdomain.Size{Width: 1024, Height: 768}
	for _, raw := range []string{
		`{"components":[{"size":{"width":100,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`,
		`{"components":[{"index":0,"size":{"width":100}},{"index":1,"size":{"width":50,"height":50}}]}`,
	} {
		if _, err := decodeUISetComponentPlan([]byte(raw), definitions, canvas); !errors.Is(err, ErrInvalidUISetPlan) {
			t.Fatalf("expected invalid plan for %s, got %v", raw, err)
		}
	}
}

func TestUISetHandlersRejectMalformedJSON(t *testing.T) {
	engine := &Engine{}
	for _, test := range []struct {
		name    string
		handle  func(context.Context, *taskdomain.Task) (any, error)
		message *taskdomain.Task
	}{
		{name: "generate", handle: engine.handleUISet, message: &taskdomain.Task{Payload: json.RawMessage(`{`)}},
		{name: "edit", handle: engine.handleEditUISetComponents, message: &taskdomain.Task{Payload: json.RawMessage(`{`)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.handle(context.Background(), test.message); err == nil {
				t.Fatal("expected queued payload decode error")
			}
		})
	}
}

func validUISetPayload() CreateUISetPayload {
	return CreateUISetPayload{
		AssetName: "Fantasy Inventory", ProjectID: 42, CreativeBrief: "compact fantasy inventory",
		Style: "ornate brass", Dimensions: assetdomain.Size{Width: 1024, Height: 768},
		Components: []UISetComponentDefinition{
			{Name: "Inventory Panel", Description: "main item grid"},
			{Name: "Close Button", Description: "icon-only control"},
		},
		ProjectContext: UISetProjectContext{Name: "Moon Forge", GameType: "RPG"},
	}
}
