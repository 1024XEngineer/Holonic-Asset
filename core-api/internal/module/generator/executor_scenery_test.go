package generator_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type sceneryLLMStub struct {
	events   *[]string
	requests []*llmclient.CompletionRequest
	results  []*llmclient.CompletionResult
}

func (s *sceneryLLMStub) Complete(_ context.Context, request *llmclient.CompletionRequest) (*llmclient.CompletionResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "llm")
	}
	s.requests = append(s.requests, request)
	call := len(s.requests) - 1
	if call >= len(s.results) {
		return nil, errors.New("missing LLM result")
	}
	return s.results[call], nil
}

type sceneryImageStub struct {
	events   *[]string
	requests []*imageclient.GenerateRequest
	results  []*imageclient.GenerateResult
}

func (s *sceneryImageStub) Generate(_ context.Context, request *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "image")
	}
	s.requests = append(s.requests, request)
	call := len(s.requests) - 1
	if call >= len(s.results) {
		return nil, errors.New("missing image result")
	}
	return s.results[call], nil
}

type sceneryProcessorStub struct{ events *[]string }

func (s *sceneryProcessorStub) RemoveBackground(_ context.Context, request *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	*s.events = append(*s.events, "remove")
	return &imageprocessor.RemoveBackgroundResult{ImageBase64: "removed:" + request.ImageBase64, MIMEType: "image/png"}, nil
}

func (s *sceneryProcessorStub) Resize(_ context.Context, request *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	*s.events = append(*s.events, "resize")
	return &imageprocessor.ResizeResult{ImageBase64: base64.StdEncoding.EncodeToString([]byte("processed:" + request.ImageBase64)), MIMEType: "image/png"}, nil
}

func (s *sceneryProcessorStub) Verify(context.Context, *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	*s.events = append(*s.events, "verify")
	return &imageprocessor.VerificationReport{Passed: true}, nil
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
	wantEvents := []string{"llm", "image", "remove", "resize", "verify", "image", "remove", "resize", "verify", "llm", "create_scenery_asset"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected workflow: got %v want %v", events, wantEvents)
	}
	if len(llm.requests) != 2 || len(llm.requests[0].Images) != 0 || len(llm.requests[1].Images) != 2 {
		t.Fatalf("expected text planning followed by multimodal layout: %+v", llm.requests)
	}
	if len(images.requests) != 2 || images.requests[0].Size != "640x360" ||
		!strings.Contains(images.requests[0].Prompt, "warm sky") || !strings.Contains(images.requests[1].Prompt, "distant peaks") {
		t.Fatalf("planner output was not passed to image generation: %+v", images.requests)
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

func sceneryImageResults() []*imageclient.GenerateResult {
	return []*imageclient.GenerateResult{
		{Images: []imageclient.GeneratedImage{{Base64: "sky-source", MediaType: "image/webp"}}},
		{Images: []imageclient.GeneratedImage{{Base64: "mountain-source", MediaType: "image/jpeg"}}},
	}
}

func sceneryPayload(t *testing.T) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(generator.CreateSceneryPayload{
		AssetName: "Mountain Valley", CreativeBrief: "A valley at dawn", Style: "pixel art",
		Dimensions: assetdomain.Size{Width: 640, Height: 360}, Perspective: "Side-On", ProjectID: 42,
	})
	if err != nil {
		t.Fatalf("marshal scenery payload: %v", err)
	}
	return payload
}
