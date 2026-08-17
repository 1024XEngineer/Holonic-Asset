package generator

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type uiSetLLMStub struct {
	request *llmclient.CompletionRequest
	result  *llmclient.CompletionResult
	err     error
}

func (s *uiSetLLMStub) Complete(
	_ context.Context,
	request *llmclient.CompletionRequest,
) (*llmclient.CompletionResult, error) {
	s.request = request
	return s.result, s.err
}

func TestPlanUISetComponentsUsesContextAndReturnsRequestOrder(t *testing.T) {
	llm := &uiSetLLMStub{result: &llmclient.CompletionResult{JSON: json.RawMessage(`{
		"components":[
			{"index":1,"size":{"width":64,"height":64}},
			{"index":0,"size":{"width":640,"height":480}}
		]
	}`)}}
	executor := &executor{llm: llm}
	payload := validUISetPlanningPayload()

	plans, err := executor.planUISetComponents(context.Background(), payload)
	if err != nil {
		t.Fatalf("plan UI Set components: %v", err)
	}
	if len(plans) != 2 || plans[0].Index != 0 || plans[0].Name != "Inventory Panel" ||
		plans[0].Description != "main item grid container" || plans[0].Size.Width != 640 ||
		plans[1].Index != 1 || plans[1].Name != "Close Button" || plans[1].Size.Height != 64 {
		t.Fatalf("unexpected UI Set plan: %+v", plans)
	}
	if llm.request == nil || len(llm.request.Images) != 0 ||
		llm.request.ResponseSchema.Name != uiSetComponentPlanSchemaName ||
		!reflect.DeepEqual(llm.request.ResponseSchema.Schema, uiSetComponentPlanJSONSchema) {
		t.Fatalf("unexpected planning request: %+v", llm.request)
	}
	for _, required := range []string{
		"Fantasy Inventory", "compact fantasy inventory", "ornate brass", "Moon Forge", "RPG", "PC",
		"inventory-driven adventure", `<canvas width="1024" height="768" />`,
		`<component index="0"><name>Inventory Panel</name><description>main item grid container</description></component>`,
		"one complete independently generated Component image", "no Tiles", "Do not choose positions",
	} {
		if !strings.Contains(llm.request.Prompt, required) {
			t.Fatalf("planning prompt omitted %q: %s", required, llm.request.Prompt)
		}
	}
}

func TestDecodeUISetComponentPlanRejectsInvalidPlans(t *testing.T) {
	definitions := validUISetPlanningPayload().Components
	canvas := assetdomain.Size{Width: 1024, Height: 768}
	tests := []struct {
		name string
		raw  string
	}{
		{"missing components", `{}`},
		{"missing component", `{"components":[{"index":0,"size":{"width":100,"height":100}}]}`},
		{"duplicate component", `{"components":[{"index":0,"size":{"width":100,"height":100}},{"index":0,"size":{"width":50,"height":50}}]}`},
		{"unknown component", `{"components":[{"index":0,"size":{"width":100,"height":100}},{"index":2,"size":{"width":50,"height":50}}]}`},
		{"negative component", `{"components":[{"index":-1,"size":{"width":100,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"fractional component", `{"components":[{"index":0.5,"size":{"width":100,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"missing size", `{"components":[{"index":0},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"missing width", `{"components":[{"index":0,"size":{"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"zero size", `{"components":[{"index":0,"size":{"width":0,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"negative size", `{"components":[{"index":0,"size":{"width":-1,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"fractional size", `{"components":[{"index":0,"size":{"width":10.5,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"outside canvas", `{"components":[{"index":0,"size":{"width":1025,"height":100}},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"unknown field", `{"components":[{"index":0,"size":{"width":100,"height":100},"tiles":[]},{"index":1,"size":{"width":50,"height":50}}]}`},
		{"trailing JSON", `{"components":[{"index":0,"size":{"width":100,"height":100}},{"index":1,"size":{"width":50,"height":50}}]} {}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeUISetComponentPlan([]byte(test.raw), definitions, canvas)
			if !errors.Is(err, ErrInvalidUISetPlan) {
				t.Fatalf("expected invalid UI Set plan, got %v", err)
			}
		})
	}
}

func TestExecutorDispatchesUISetPhaseOneAndTwo(t *testing.T) {
	llm := &uiSetLLMStub{result: &llmclient.CompletionResult{JSON: json.RawMessage(`{
		"components":[
			{"index":0,"size":{"width":640,"height":480}},
			{"index":1,"size":{"width":64,"height":64}}
		]
	}`)}}
	executor := &executor{llm: llm}
	payload, err := json.Marshal(validUISetPlanningPayload())
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Generate(context.Background(), GenerateUISet, payload)
	if err != nil || result != nil || llm.request == nil {
		t.Fatalf("unexpected UI Set generation dispatch: result=%s err=%v request=%+v", result, err, llm.request)
	}

	edit := EditUISetComponentsPayload{
		AssetID: 100, ProjectID: 42, CreativeBrief: "brighter",
		TargetAssetPaths: []string{"components.1", "components.3"},
	}
	editPayload, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	result, err = executor.Generate(context.Background(), EditUISetComponents, editPayload)
	if err != nil || result != nil {
		t.Fatalf("unexpected UI Set edit dispatch: result=%s err=%v", result, err)
	}
}

func TestPlanUISetComponentsPreservesProviderFailure(t *testing.T) {
	wantErr := errors.New("planning unavailable")
	executor := &executor{llm: &uiSetLLMStub{err: wantErr}}
	_, err := executor.planUISetComponents(context.Background(), validUISetPlanningPayload())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func validUISetPlanningPayload() CreateUISetPayload {
	return CreateUISetPayload{
		AssetName: "Fantasy Inventory", ProjectID: 42, CreativeBrief: "compact fantasy inventory",
		Style: "ornate brass", Dimensions: assetdomain.Size{Width: 1024, Height: 768},
		Components: []UISetComponentDefinition{
			{Name: "Inventory Panel", Description: "main item grid container"},
			{Name: "Close Button", Description: "icon-only close control"},
		},
		ProjectContext: UISetProjectContext{
			Name: "Moon Forge", GameType: "RPG", TargetPlatform: "PC", Description: "inventory-driven adventure",
		},
	}
}
