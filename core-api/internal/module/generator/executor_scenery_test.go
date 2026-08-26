package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type sceneryLLMStub struct {
	events   *[]string
	requests []*llmclient.CompletionRequest
	results  []*llmclient.CompletionResult
	errors   []error
}

func (s *sceneryLLMStub) Complete(_ context.Context, request *llmclient.CompletionRequest) (*llmclient.CompletionResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "llm")
	}
	s.requests = append(s.requests, request)
	call := len(s.requests) - 1
	if call < len(s.errors) && s.errors[call] != nil {
		return nil, s.errors[call]
	}
	if call >= len(s.results) {
		return nil, errors.New("missing LLM result")
	}
	return s.results[call], nil
}

type sceneryImageStub struct {
	events   *[]string
	requests []*imageclient.GenerateRequest
	results  []*imageclient.GenerateResult
	errors   []error
}

func (s *sceneryImageStub) Generate(_ context.Context, request *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "image")
	}
	s.requests = append(s.requests, request)
	call := len(s.requests) - 1
	if call < len(s.errors) && s.errors[call] != nil {
		return nil, s.errors[call]
	}
	if call >= len(s.results) {
		return nil, errors.New("missing image result")
	}
	return s.results[call], nil
}

type sceneryProcessorStub struct {
	events          *[]string
	removeErr       error
	resizeErr       error
	verifyErr       error
	removed         *imageprocessor.RemoveBackgroundResult
	resized         *imageprocessor.ResizeResult
	verified        *imageprocessor.VerificationReport
	verifiedResults []*imageprocessor.VerificationReport
	verifyRequests  []*imageprocessor.VerifyRequest
	resizeRequests  []*imageprocessor.ResizeRequest
	verifyCalls     int
}

func (s *sceneryProcessorStub) RemoveBackground(_ context.Context, request *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	*s.events = append(*s.events, "remove")
	if s.removeErr != nil {
		return nil, s.removeErr
	}
	if s.removed != nil {
		return s.removed, nil
	}
	return &imageprocessor.RemoveBackgroundResult{ImageBase64: "removed:" + request.ImageBase64, MIMEType: "image/png"}, nil
}

func (*sceneryProcessorStub) NormalizeReference(_ context.Context, request *imageprocessor.NormalizeReferenceRequest) (*imageprocessor.NormalizeReferenceResult, error) {
	return &imageprocessor.NormalizeReferenceResult{ImageBase64: request.ImageBase64, MIMEType: "image/png", Report: imageprocessor.ReferenceNormalizationReport{Scale: 1}}, nil
}

func (s *sceneryProcessorStub) Resize(_ context.Context, request *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	*s.events = append(*s.events, "resize")
	s.resizeRequests = append(s.resizeRequests, request)
	if s.resizeErr != nil {
		return nil, s.resizeErr
	}
	if s.resized != nil {
		return s.resized, nil
	}
	fixture := image.NewRGBA(image.Rect(0, 0, request.Options.Width, request.Options.Height))
	fixture.SetRGBA(0, 0, color.RGBA{R: uint8(len(s.resizeRequests) & 0xff), A: 255})
	encoded, err := imageprocessor.EncodePNGBase64(fixture)
	if err != nil {
		return nil, err
	}
	return &imageprocessor.ResizeResult{ImageBase64: encoded, MIMEType: "image/png"}, nil
}

func (s *sceneryProcessorStub) Verify(_ context.Context, request *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	*s.events = append(*s.events, "verify")
	s.verifyRequests = append(s.verifyRequests, request)
	if s.verifyErr != nil {
		return nil, s.verifyErr
	}
	if s.verifyCalls < len(s.verifiedResults) {
		result := s.verifiedResults[s.verifyCalls]
		s.verifyCalls++
		return result, nil
	}
	s.verifyCalls++
	if s.verified != nil {
		return s.verified, nil
	}
	return &imageprocessor.VerificationReport{Passed: true}, nil
}

func TestExecutorStopsSceneryWorkflowAtFailedStage(t *testing.T) {
	wantErr := errors.New("stage failed")
	tests := []struct {
		name      string
		llm       *sceneryLLMStub
		images    *sceneryImageStub
		processor *sceneryProcessorStub
		wantCause bool
	}{
		{name: "planning provider", llm: &sceneryLLMStub{errors: []error{wantErr}}, wantCause: true},
		{name: "empty planning result", llm: &sceneryLLMStub{results: []*llmclient.CompletionResult{nil}}},
		{name: "invalid plan", llm: &sceneryLLMStub{results: []*llmclient.CompletionResult{{JSON: json.RawMessage(`{}`)}}}},
		{name: "image provider", images: &sceneryImageStub{errors: []error{wantErr}}, wantCause: true},
		{name: "empty image", images: &sceneryImageStub{results: []*imageclient.GenerateResult{nil}}},
		{name: "remove background", llm: validSceneryLLM(nil), images: &sceneryImageStub{results: sceneryImageResults()}, processor: &sceneryProcessorStub{removeErr: wantErr}, wantCause: true},
		{name: "empty removed image", llm: validSceneryLLM(nil), images: &sceneryImageStub{results: sceneryImageResults()}, processor: &sceneryProcessorStub{removed: &imageprocessor.RemoveBackgroundResult{}}},
		{name: "resize", processor: &sceneryProcessorStub{resizeErr: wantErr}, wantCause: true},
		{name: "invalid resized image", processor: &sceneryProcessorStub{resized: &imageprocessor.ResizeResult{ImageBase64: "png", MIMEType: "image/jpeg"}}},
		{name: "verify", processor: &sceneryProcessorStub{verifyErr: wantErr}, wantCause: true},
		{name: "verification rejected", processor: &sceneryProcessorStub{verified: &imageprocessor.VerificationReport{Passed: false}}},
		{name: "layout provider", llm: &sceneryLLMStub{results: []*llmclient.CompletionResult{{JSON: json.RawMessage(`{"layers":[{"name":"Sky","creative_brief":"warm sky"}]}`)}}, errors: []error{nil, wantErr}}, wantCause: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			llm := test.llm
			if llm == nil {
				llm = validSingleLayerSceneryLLM()
			}
			llm.events = &events
			images := test.images
			if images == nil {
				images = &sceneryImageStub{results: sceneryImageResults()[:1]}
			}
			images.events = &events
			processor := test.processor
			if processor == nil {
				processor = &sceneryProcessorStub{}
			}
			processor.events = &events
			assets := &generationAssetWriterStub{events: &events}
			executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{
				LLM: llm, Resources: &sceneryResourceStoreStub{},
			})
			_, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
			if err == nil || assets.sceneryAsset != nil {
				t.Fatalf("expected workflow failure before asset creation, got err=%v asset=%+v events=%v", err, assets.sceneryAsset, events)
			}
			if test.wantCause && !errors.Is(err, wantErr) {
				t.Fatalf("workflow lost the injected failure cause: err=%v events=%v", err, events)
			}
		})
	}
}

func (*sceneryProcessorStub) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	return &imageprocessor.SplitImageResult{}, nil
}

type sceneryResourceStoreStub struct {
	keys      []string
	deleted   []string
	putErrAt  int
	putErr    error
	cancelAt  int
	cancel    context.CancelFunc
	deleteCtx []error
}

func (s *sceneryResourceStoreStub) PutObject(_ context.Context, key, _ string, _ []byte) error {
	call := len(s.keys) + 1
	if s.putErrAt == call {
		return s.putErr
	}
	s.keys = append(s.keys, key)
	if s.cancelAt == call && s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *sceneryResourceStoreStub) DeleteObject(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	s.deleteCtx = append(s.deleteCtx, ctx.Err())
	return nil
}

func TestExecutorPlansAndAnalyzesSceneryAroundLayerGeneration(t *testing.T) {
	events := []string{}
	images := &sceneryImageStub{events: &events, results: sceneryImageResults()}
	llm := validSceneryLLM(&events)
	processor := &sceneryProcessorStub{events: &events}
	assets := &generationAssetWriterStub{events: &events}
	resources := &sceneryResourceStoreStub{}
	executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{LLM: llm, Resources: resources})

	result, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
	if err != nil {
		t.Fatalf("generate scenery: %v", err)
	}
	wantEvents := []string{"llm", "image", "remove", "resize", "verify", "image", "resize", "verify", "llm", "create_scenery_asset"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected workflow: got %v want %v", events, wantEvents)
	}
	if len(llm.requests) != 2 || len(llm.requests[0].Images) != 0 || len(llm.requests[1].Images) != 2 {
		t.Fatalf("expected text planning followed by multimodal layout: %+v", llm.requests)
	}
	if len(images.requests) != 2 || images.requests[0].Size != "640x360" ||
		!strings.Contains(images.requests[0].Prompt, "distant peaks") || !strings.Contains(images.requests[1].Prompt, "warm sky") {
		t.Fatalf("planner output was not passed to image generation: %+v", images.requests)
	}
	if len(images.requests[0].ReferenceImages) != 0 || len(images.requests[1].ReferenceImages) != 1 ||
		!strings.HasPrefix(images.requests[1].ReferenceImages[0], "data:image/png;base64,") {
		t.Fatalf("foreground preview was not chained into backdrop generation: %+v", images.requests)
	}
	if !strings.Contains(images.requests[1].Prompt, "cumulative transparent preview") ||
		!strings.Contains(images.requests[1].Prompt, "Generate only the requested layer behind the preview") {
		t.Fatalf("backdrop prompt omitted chained reference semantics: %s", images.requests[1].Prompt)
	}
	if len(processor.verifyRequests) != 2 || processor.verifyRequests[0].Profile != imageprocessor.ProfileGeneric ||
		processor.verifyRequests[1].Profile != imageprocessor.ProfileOpaqueBackground {
		t.Fatalf("unexpected scenery verification profiles: %+v", processor.verifyRequests)
	}
	if len(processor.resizeRequests) != 2 || processor.resizeRequests[0].Options.CoverCanvas ||
		!processor.resizeRequests[1].Options.CoverCanvas {
		t.Fatalf("unexpected scenery resize modes: %+v", processor.resizeRequests)
	}
	var decoded generator.ExecutionResult
	if err := json.Unmarshal(result, &decoded); err != nil || decoded.AssetID != 43 {
		t.Fatalf("unexpected execution result: result=%s err=%v", result, err)
	}
	if assets.sceneryAsset == nil || len(resources.keys) != 2 {
		t.Fatalf("scenery was not persisted: asset=%+v keys=%v", assets.sceneryAsset, resources.keys)
	}
	content, err := assets.sceneryAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode scenery content: %v", err)
	}
	if len(content.Layers) != 2 || content.Layers[0].ID != 1 || *content.Layers[0].ZIndex != -10 ||
		content.Layers[1].ID != 2 || content.Layers[1].Position.X != 100 || *content.Layers[1].ZIndex != 20 {
		t.Fatalf("layouts were not associated by stable ID: %+v", content.Layers)
	}
}

func TestExecutorRetriesRejectedSceneryLayer(t *testing.T) {
	events := []string{}
	images := &sceneryImageStub{events: &events, results: []*imageclient.GenerateResult{
		{Images: []imageclient.GeneratedImage{{Base64: "first"}}},
		{Images: []imageclient.GeneratedImage{{Base64: "second"}}},
	}}
	processor := &sceneryProcessorStub{
		events: &events,
		verifiedResults: []*imageprocessor.VerificationReport{
			{Passed: false, FailureReasons: []string{"empty_subject"}},
			{Passed: true},
		},
	}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{
		LLM: validSingleLayerSceneryLLM(), Resources: &sceneryResourceStoreStub{},
	})

	_, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
	if err != nil {
		t.Fatalf("retry scenery layer: %v", err)
	}
	if len(images.requests) != 2 || images.requests[0].MaxAttempts != 2 || processor.verifyCalls != 2 || assets.sceneryAsset == nil {
		t.Fatalf("expected one automatic retry: imageRequests=%d verifyCalls=%d asset=%+v", len(images.requests), processor.verifyCalls, assets.sceneryAsset)
	}
}

func TestExecutorDoesNotUseQualityRetriesForProviderErrors(t *testing.T) {
	events := []string{}
	providerErr := &imageclient.ProviderError{Kind: imageclient.ErrorKindUnavailable, Transient: true}
	images := &sceneryImageStub{events: &events, errors: []error{providerErr}}
	executor := generator.NewExecutorWithDependencies(
		images,
		&sceneryProcessorStub{events: &events},
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{LLM: validSingleLayerSceneryLLM(), Resources: &sceneryResourceStoreStub{}},
	)

	_, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
	if err == nil || !errors.Is(err, providerErr) || len(images.requests) != 1 {
		t.Fatalf("provider failure consumed quality retries: requests=%d err=%v", len(images.requests), err)
	}
}

func TestCreateBuildsSceneryPayloadFromProjectContext(t *testing.T) {
	tasks := &taskManagerStub{createID: 17}
	projects := &projectReaderStub{project: &projectdomain.Project{
		Name: "Moon Valley", GameType: "RPG", TargetPlatform: "PC", Description: "exploration",
		Style: "pixel art", Perspective: "Side-On", Reference: "projects/42/reference.png",
	}}
	references := &referenceStoreStub{}
	engine := generator.NewEngine(tasks, nil, generator.EngineDependencies{Projects: projects, References: references})

	_, err := engine.Create(context.Background(), &generator.Request{
		ProjectID: 42, Kind: generator.GenerateScenery, CreativeBrief: "a valley at dawn",
		Parameters: json.RawMessage(`{"asset_name":"Dawn Valley","dimensions":{"width":640,"height":360},"creating_reference":"https://cdn.example.com/creating.png"}`),
	})
	if err != nil {
		t.Fatalf("create scenery: %v", err)
	}
	var payload generator.CreateSceneryPayload
	if err := json.Unmarshal(tasks.createdTask.Payload, &payload); err != nil {
		t.Fatalf("decode scenery payload: %v", err)
	}
	if payload.AssetName != "Dawn Valley" || payload.Perspective != "Side-On" ||
		payload.ProjectContext.Name != "Moon Valley" || payload.ProjectContext.GameType != "RPG" ||
		payload.ProjectContext.TargetPlatform != "PC" || payload.ProjectContext.Description != "exploration" ||
		payload.CreatingReference != "uploads/generated-2.png" || payload.ProjectReference != "uploads/generated-1.png" || projects.calls != 1 ||
		!reflect.DeepEqual(references.persisted, []string{
			"projects/42/reference.png",
			"https://cdn.example.com/creating.png",
		}) {
		t.Fatalf("unexpected scenery preparation: payload=%+v project_calls=%d persisted=%v", payload, projects.calls, references.persisted)
	}
}

func TestCreateRejectsInvalidSceneryRequests(t *testing.T) {
	assetID := uint(7)
	tests := []struct {
		name    string
		request *generator.Request
	}{
		{name: "asset target", request: &generator.Request{ProjectID: 42, AssetID: &assetID, Kind: generator.GenerateScenery, CreativeBrief: "forest", Parameters: json.RawMessage(`{"asset_name":"Forest","dimensions":{"width":640,"height":360}}`)}},
		{name: "unknown parameter", request: &generator.Request{ProjectID: 42, Kind: generator.GenerateScenery, CreativeBrief: "forest", Parameters: json.RawMessage(`{"asset_name":"Forest","dimensions":{"width":640,"height":360},"layers":[]}`)}},
		{name: "missing name", request: &generator.Request{ProjectID: 42, Kind: generator.GenerateScenery, CreativeBrief: "forest", Parameters: json.RawMessage(`{"dimensions":{"width":640,"height":360}}`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tasks := &taskManagerStub{}
			_, err := generator.NewEngine(tasks, nil).Create(context.Background(), test.request)
			if !errors.Is(err, generator.ErrInvalidSceneryPayload) || tasks.createdTask != nil {
				t.Fatalf("expected invalid scenery payload without publish, got err=%v task=%+v", err, tasks.createdTask)
			}
		})
	}
}

func TestExecutorCleansUpSceneryResourcesAfterFailures(t *testing.T) {
	t.Run("upload", func(t *testing.T) {
		wantErr := errors.New("object storage unavailable")
		assets := &generationAssetWriterStub{}
		resources := &sceneryResourceStoreStub{putErrAt: 2, putErr: wantErr}
		_, err := newSceneryExecutor(assets, resources).Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
		if !errors.Is(err, wantErr) || assets.sceneryAsset != nil || len(resources.deleted) != 1 || resources.deleted[0] != resources.keys[0] {
			t.Fatalf("unexpected upload cleanup: err=%v asset=%+v keys=%v deleted=%v", err, assets.sceneryAsset, resources.keys, resources.deleted)
		}
	})

	t.Run("asset creation", func(t *testing.T) {
		wantErr := errors.New("database unavailable")
		assets := &generationAssetWriterStub{err: wantErr}
		resources := &sceneryResourceStoreStub{}
		_, err := newSceneryExecutor(assets, resources).Generate(context.Background(), generator.GenerateScenery, sceneryPayload(t))
		if !errors.Is(err, wantErr) || len(resources.deleted) != 2 || resources.deleted[0] != resources.keys[1] || resources.deleted[1] != resources.keys[0] {
			t.Fatalf("unexpected asset cleanup: err=%v keys=%v deleted=%v", err, resources.keys, resources.deleted)
		}
	})
}

func TestExecutorUsesFreshContextToCleanUpCancelledScenery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	resources := &sceneryResourceStoreStub{cancelAt: 1, cancel: cancel}
	_, err := newSceneryExecutor(&generationAssetWriterStub{}, resources).Generate(ctx, generator.GenerateScenery, sceneryPayload(t))
	if !errors.Is(err, context.Canceled) || len(resources.deleted) != 1 || resources.deleteCtx[0] != nil {
		t.Fatalf("cleanup reused cancelled context: err=%v deleted=%v contextErrors=%v", err, resources.deleted, resources.deleteCtx)
	}
}

func newSceneryExecutor(assets generator.AssetWriter, resources generator.ResourceStore) generator.Executor {
	events := []string{}
	return generator.NewExecutorWithDependencies(
		&sceneryImageStub{results: sceneryImageResults()},
		&sceneryProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{LLM: validSceneryLLM(nil), Resources: resources},
	)
}

func validSceneryLLM(events *[]string) *sceneryLLMStub {
	return &sceneryLLMStub{events: events, results: []*llmclient.CompletionResult{
		{JSON: json.RawMessage(`{"layers":[{"name":"Sky","creative_brief":"warm sky"},{"name":"Mountains","creative_brief":"distant peaks"}]}`)},
		{JSON: json.RawMessage(`{"layers":[{"id":2,"position":{"x":100,"y":40},"scale":{"x":0.8,"y":0.8},"rotation":0,"opacity":0.75,"zIndex":20},{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-10}]}`)},
	}}
}

func validSingleLayerSceneryLLM() *sceneryLLMStub {
	return &sceneryLLMStub{results: []*llmclient.CompletionResult{
		{JSON: json.RawMessage(`{"layers":[{"name":"Sky","creative_brief":"warm sky"}]}`)},
		{JSON: json.RawMessage(`{"layers":[{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":0}]}`)},
	}}
}

func sceneryImageResults() []*imageclient.GenerateResult {
	return []*imageclient.GenerateResult{
		{Images: []imageclient.GeneratedImage{{Base64: "sky-source", MediaType: "image/webp"}}},
		{Images: []imageclient.GeneratedImage{{Base64: "mountain-source", MediaType: "image/jpeg"}}},
	}
}

func sceneryPayload(t *testing.T) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(generator.CreateSceneryPayload{
		AssetName: "Mountain Valley", CreativeBrief: "A valley at dawn",
		Dimensions: assetdomain.Size{Width: 640, Height: 360}, Perspective: "Side-On", ProjectID: 42,
	})
	if err != nil {
		t.Fatalf("marshal scenery payload: %v", err)
	}
	return payload
}
