package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func TestCreateBuildsSelfContainedUISetPayload(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	projects := &projectReaderStub{project: &projectdomain.Project{
		Name: "Moon Forge", GameType: "RPG", TargetPlatform: projectdomain.PlatformTypePC,
		Description: "inventory-driven adventure",
	}}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{Projects: projects, References: references})

	runID, err := engine.Create(context.Background(), validUISetRequest())
	if err != nil {
		t.Fatalf("create UI Set generation: %v", err)
	}
	if runID != 17 || tasks.createdTask == nil || tasks.createdTask.Type != string(generator.GenerateUISet) {
		t.Fatalf("unexpected task creation: run=%d task=%+v", runID, tasks.createdTask)
	}
	var payload generator.CreateUISetPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode UI Set payload: %v", err)
	}
	if payload.ProjectID != 42 || payload.AssetName != "Fantasy Inventory" ||
		payload.CreativeBrief != "A compact fantasy inventory interface" || payload.Style != "ornate brass" ||
		payload.Dimensions.Width != 1024 || payload.Dimensions.Height != 768 || len(payload.Components) != 2 ||
		payload.Components[0].Name != "Inventory Panel" || payload.Components[1].Description != "icon-only close control" ||
		payload.ProjectContext.Name != "Moon Forge" || payload.ProjectContext.TargetPlatform != "PC" ||
		payload.Reference != "uploads/generated-1.png" {
		t.Fatalf("unexpected UI Set payload: %+v", payload)
	}
	if !reflect.DeepEqual(references.persisted, []string{"https://cdn.example/ui-reference.png"}) {
		t.Fatalf("unexpected persisted references: %v", references.persisted)
	}
	for _, forbidden := range []string{"tileSize", "tileAmount", "shape", "tiles"} {
		if strings.Contains(string(tasks.createdTask.Payload), forbidden) {
			t.Fatalf("UI Set payload contains Tileset field %q: %s", forbidden, tasks.createdTask.Payload)
		}
	}
}

func TestCreateBuildsUISetComponentEditPayload(t *testing.T) {
	assetID := uint(100)
	tasks := &taskManagerStub{createID: 18}
	projects := &projectReaderStub{}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{Projects: projects, References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42, AssetID: &assetID, Kind: generator.EditUISetComponents,
		CreativeBrief:    "Make the selected controls brighter",
		TargetAssetPaths: []string{"components.1", "components.3"},
		Parameters:       json.RawMessage(`{"reference":"https://cdn.example/button.png"}`),
	})
	if err != nil {
		t.Fatalf("create UI Set edit: %v", err)
	}
	var payload generator.EditUISetComponentsPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode UI Set edit payload: %v", err)
	}
	if payload.AssetID != assetID || payload.ProjectID != 42 ||
		payload.CreativeBrief != "Make the selected controls brighter" ||
		!reflect.DeepEqual(payload.TargetAssetPaths, []string{"components.1", "components.3"}) ||
		payload.Reference != "uploads/generated-1.png" {
		t.Fatalf("unexpected UI Set edit payload: %+v", payload)
	}
	if projects.calls != 0 {
		t.Fatalf("UI Set edit preparation unexpectedly loaded the Project: %d", projects.calls)
	}
}

func TestCreateRejectsInvalidUISetRequestsBeforePublishing(t *testing.T) {
	assetID := uint(100)
	tests := []struct {
		name    string
		request *generator.Request
	}{
		{"generation asset", uiSetRequestWith(func(request *generator.Request) { request.AssetID = &assetID })},
		{"generation targets", uiSetRequestWith(func(request *generator.Request) { request.TargetAssetPaths = []string{"components.0"} })},
		{"missing project", uiSetRequestWith(func(request *generator.Request) { request.ProjectID = 0 })},
		{"unknown parameter", uiSetRequestWithParameters(`{"unexpected":true}`)},
		{"frontend-only component field", uiSetRequestWithParameters(`{"asset_name":"UI","dimensions":{"width":100,"height":100},"style":"pixel","components":[{"name":"button","description":"start","isCustom":true}]}`)},
		{"blank asset name", uiSetRequestWithParameters(validUISetParametersWith("asset_name", " "))},
		{"blank style", uiSetRequestWithParameters(validUISetParametersWith("style", " "))},
		{"zero dimensions", uiSetRequestWithParameters(validUISetParametersWith("dimensions", json.RawMessage(`{"width":0,"height":768}`)))},
		{"oversized dimensions", uiSetRequestWithParameters(validUISetParametersWith("dimensions", json.RawMessage(`{"width":4097,"height":768}`)))},
		{"missing components", uiSetRequestWithParameters(validUISetParametersWith("components", json.RawMessage(`[]`)))},
		{"blank component name", uiSetRequestWithParameters(validUISetParametersWith("components", json.RawMessage(`[{"name":" ","description":"button"}]`)))},
		{"blank component description", uiSetRequestWithParameters(validUISetParametersWith("components", json.RawMessage(`[{"name":"button","description":" "}]`)))},
		{"blank reference", uiSetRequestWithParameters(validUISetParametersWith("reference", " "))},
		{"trailing JSON", uiSetRequestWithParameters(validUISetParameters() + `{}`)},
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

func TestCreateRejectsInvalidUISetEditPathsBeforePublishing(t *testing.T) {
	assetID := uint(100)
	tests := []struct {
		name       string
		assetID    *uint
		paths      []string
		parameters json.RawMessage
	}{
		{name: "missing asset", paths: []string{"components.0"}},
		{name: "missing paths", assetID: &assetID},
		{name: "wrong root", assetID: &assetID, paths: []string{"items.0"}},
		{name: "negative index", assetID: &assetID, paths: []string{"components.-1"}},
		{name: "fractional index", assetID: &assetID, paths: []string{"components.1.5"}},
		{name: "trailing segment", assetID: &assetID, paths: []string{"components.1.texture"}},
		{name: "duplicate index", assetID: &assetID, paths: []string{"components.1", "components.01"}},
		{name: "unknown parameter", assetID: &assetID, paths: []string{"components.0"}, parameters: json.RawMessage(`{"style":"new"}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tasks := &taskManagerStub{}
			_, err := generator.NewEngine(tasks, nil).Create(context.Background(), &generator.Request{
				ProjectID: 42, AssetID: test.assetID, Kind: generator.EditUISetComponents,
				CreativeBrief: "edit", TargetAssetPaths: test.paths, Parameters: test.parameters,
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

func TestUISetTaskHandlersRejectInvalidQueuedPayloads(t *testing.T) {
	tests := []struct {
		kind    generator.TaskType
		payload json.RawMessage
	}{
		{generator.GenerateUISet, json.RawMessage(`{"asset_name":"legacy","project_id":42,"style":"pixel","dimensions":{"width":100,"height":100},"components":[],"project_context":{}}`)},
		{generator.EditUISetComponents, json.RawMessage(`{"asset_id":100,"project_id":42,"creative_brief":"edit","target_asset_paths":["components.1.texture"]}`)},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			tasks := &taskManagerStub{}
			executor := &executorStub{}
			generator.NewEngine(tasks, executor)
			_, err := tasks.dispatch(context.Background(), &taskdomain.Task{ID: 17, Type: string(test.kind), Payload: test.payload})
			if !errors.Is(err, generator.ErrInvalidTaskPayload) || executor.calls != 0 {
				t.Fatalf("invalid queued payload reached executor: err=%v calls=%d", err, executor.calls)
			}
		})
	}
}

func validUISetRequest() *generator.Request {
	return &generator.Request{
		ProjectID: 42, Kind: generator.GenerateUISet,
		CreativeBrief: "A compact fantasy inventory interface",
		Parameters:    json.RawMessage(validUISetParameters()),
	}
}

func uiSetRequestWith(change func(*generator.Request)) *generator.Request {
	request := validUISetRequest()
	change(request)
	return request
}

func uiSetRequestWithParameters(parameters string) *generator.Request {
	request := validUISetRequest()
	request.Parameters = json.RawMessage(parameters)
	return request
}

func validUISetParameters() string {
	return `{
		"asset_name":"Fantasy Inventory",
		"dimensions":{"width":1024,"height":768},
		"style":"ornate brass",
		"components":[
			{"name":"Inventory Panel","description":"main item grid container"},
			{"name":"Close Button","description":"icon-only close control"}
		],
		"reference":"https://cdn.example/ui-reference.png"
	}`
}

func validUISetParametersWith(field string, value any) string {
	var parameters map[string]any
	if err := json.Unmarshal([]byte(validUISetParameters()), &parameters); err != nil {
		panic(err)
	}
	parameters[field] = value
	encoded, err := json.Marshal(parameters)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
