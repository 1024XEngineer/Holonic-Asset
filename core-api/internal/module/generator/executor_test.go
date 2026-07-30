package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type imageGenerationServiceStub struct {
	events  *[]string
	request *imageclient.GenerateRequest
	result  *imageclient.GenerateResult
	err     error
}

func (s *imageGenerationServiceStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	*s.events = append(*s.events, "generate_image")
	s.request = &imageclient.GenerateRequest{
		Prompt:          request.Prompt,
		ReferenceImages: append([]string(nil), request.ReferenceImages...),
		Model:           request.Model,
		Size:            request.Size,
		Params:          request.Params,
	}
	return s.result, s.err
}

type generationAssetWriterStub struct {
	events           *[]string
	characterAsset   *assetdomain.Asset
	objectAsset      *assetdomain.Asset
	prototypeAssetID uint
	prototypeImages  []assetdomain.ImageResource
	animationAssetID uint
	animationName    string
	animationID      uint
	frames           []assetdomain.Frame
	err              error
}

func (s *generationAssetWriterStub) CreateCharacterAsset(
	_ context.Context,
	value *assetdomain.Asset,
) (*assetdomain.Asset, error) {
	*s.events = append(*s.events, "create_character_asset")
	s.characterAsset = value
	if s.err != nil {
		return nil, s.err
	}
	return &assetdomain.Asset{ID: 41}, nil
}

func (s *generationAssetWriterStub) CreateObjectAsset(
	_ context.Context,
	value *assetdomain.Asset,
) (uint, error) {
	*s.events = append(*s.events, "create_object_asset")
	s.objectAsset = value
	if s.err != nil {
		return 0, s.err
	}
	return 42, nil
}

func (s *generationAssetWriterStub) CreateAnimation(
	_ context.Context,
	assetID uint,
	name string,
) (uint, error) {
	*s.events = append(*s.events, "create_animation")
	s.animationAssetID = assetID
	s.animationName = name
	if s.err != nil {
		return 0, s.err
	}
	return 3, nil
}

func (s *generationAssetWriterStub) UpdatePrototypeImages(
	_ context.Context,
	assetID uint,
	images []assetdomain.ImageResource,
) error {
	*s.events = append(*s.events, "update_prototype")
	s.prototypeAssetID = assetID
	s.prototypeImages = append([]assetdomain.ImageResource(nil), images...)
	return s.err
}

func (s *generationAssetWriterStub) UpdateAnimationFrames(
	_ context.Context,
	assetID uint,
	animationID uint,
	frames []assetdomain.Frame,
) error {
	*s.events = append(*s.events, "update_animation_frames")
	s.animationAssetID = assetID
	s.animationID = animationID
	s.frames = append([]assetdomain.Frame(nil), frames...)
	return s.err
}

func TestExecutorGeneratesCharacterPrototypeBeforeUpdatingAsset(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{
		events: &events,
		result: generatedImages(),
	}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, assets)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
		"canvas_size":"64x64",
		"perspective":"top-down",
		"direction_count":"4",
		"reference":"https://cdn.example/reference.png",
		"project_id":11
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload)
	if err != nil {
		t.Fatalf("generate character prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"create_character_asset",
		"update_prototype",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || images.request.Prompt != "pixel knight" ||
		images.request.Size != "64x64" ||
		!reflect.DeepEqual(images.request.ReferenceImages, []string{"https://cdn.example/reference.png"}) {
		t.Fatalf("unexpected image request: %+v", images.request)
	}
	if assets.characterAsset == nil || assets.characterAsset.Name != "hero" ||
		assets.characterAsset.ProjectID != 11 ||
		assets.characterAsset.Description != "pixel knight" {
		t.Fatalf("unexpected character asset: %+v", assets.characterAsset)
	}
	content, err := assets.characterAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode character content: %v", err)
	}
	if content.ViewMode != assetdomain.ViewModeTopDown || content.DirectionCount != 4 {
		t.Fatalf("unexpected character content: %+v", content)
	}
	assertPrototypeResources(t, assets, 41)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 41})
}

func TestExecutorGeneratesObjectPrototypeBeforeUpdatingAsset(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, assets)
	payload := json.RawMessage(`{
		"asset_name":"chest",
		"creative_brief":"wooden chest",
		"canvas_size":"128x128",
		"perspective":"isometric",
		"project_id":12
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if err != nil {
		t.Fatalf("generate object prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"create_object_asset",
		"update_prototype",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if assets.objectAsset == nil || assets.objectAsset.Name != "chest" ||
		assets.objectAsset.ProjectID != 12 || assets.objectAsset.Type != assetdomain.AssetTypeObject {
		t.Fatalf("unexpected object asset: %+v", assets.objectAsset)
	}
	assertPrototypeResources(t, assets, 42)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 42})
}

func TestExecutorGeneratesAnimationBeforeUpdatingFrames(t *testing.T) {
	tests := []generator.TaskType{
		generator.GenerateCharacterAnimation,
		generator.GenerateObjectAnimation,
	}
	for _, taskType := range tests {
		t.Run(string(taskType), func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			assets := &generationAssetWriterStub{events: &events}
			executor := generator.NewExecutor(images, assets)
			payload := json.RawMessage(`{
				"asset_name":"walk",
				"creative_brief":"walking cycle",
				"parent_id":7,
				"project_id":11
			}`)

			result, err := executor.Generate(context.Background(), taskType, payload)
			if err != nil {
				t.Fatalf("generate animation: %v", err)
			}
			if !reflect.DeepEqual(events, []string{
				"generate_image",
				"create_animation",
				"update_animation_frames",
			}) {
				t.Fatalf("unexpected workflow order: %v", events)
			}
			if images.request == nil || images.request.Prompt != "walking cycle" ||
				len(images.request.ReferenceImages) != 0 || images.request.Size != "" {
				t.Fatalf("unexpected image request: %+v", images.request)
			}
			if assets.animationAssetID != 7 || assets.animationID != 3 ||
				assets.animationName != "walk" || len(assets.frames) != 2 {
				t.Fatalf("unexpected animation update: %+v", assets)
			}
			if assets.frames[0].ID != 1 || assets.frames[0].URL == nil ||
				*assets.frames[0].URL != "data:image/png;base64,first" ||
				assets.frames[1].ID != 2 || assets.frames[1].URL == nil ||
				*assets.frames[1].URL != "data:image/webp;base64,second" {
				t.Fatalf("unexpected animation frames: %+v", assets.frames)
			}
			assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, AnimationID: 3})
		})
	}
}

func TestExecutorDoesNotMutateAssetsWhenImageGenerationFails(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, err: wantErr}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, assets)

	_, err := executor.Generate(
		context.Background(),
		generator.GenerateCharacterAnimation,
		json.RawMessage(`{"asset_name":"walk","parent_id":7}`),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected image generation error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"generate_image"}) {
		t.Fatalf("asset changed before image generation succeeded: %v", events)
	}
}

func TestExecutorRequiresDependencies(t *testing.T) {
	executor := generator.NewExecutor(nil, nil)
	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageServiceRequired) {
		t.Fatalf("expected image service required error, got %v", err)
	}

	events := []string{}
	executor = generator.NewExecutor(&imageGenerationServiceStub{events: &events}, nil)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrAssetWriterRequired) {
		t.Fatalf("expected asset writer required error, got %v", err)
	}
}

func generatedImages() *imageclient.GenerateResult {
	return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{
		{Base64: "first", MediaType: "image/png"},
		{Base64: "second", MediaType: "image/webp"},
	}}
}

func assertPrototypeResources(t *testing.T, assets *generationAssetWriterStub, assetID uint) {
	t.Helper()
	if assets.prototypeAssetID != assetID || len(assets.prototypeImages) != 2 {
		t.Fatalf("unexpected prototype update: id=%d images=%+v", assets.prototypeAssetID, assets.prototypeImages)
	}
	if assets.prototypeImages[0].ID != 1 || assets.prototypeImages[0].URL == nil ||
		*assets.prototypeImages[0].URL != "data:image/png;base64,first" ||
		assets.prototypeImages[1].ID != 2 || assets.prototypeImages[1].URL == nil ||
		*assets.prototypeImages[1].URL != "data:image/webp;base64,second" {
		t.Fatalf("unexpected prototype resources: %+v", assets.prototypeImages)
	}
}

func assertExecutionResult(t *testing.T, raw json.RawMessage, want generator.ExecutionResult) {
	t.Helper()
	var got generator.ExecutionResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode execution result: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected execution result: got %+v want %+v", got, want)
	}
}

var _ imageclient.ImageGenerationService = (*imageGenerationServiceStub)(nil)
var _ generator.AssetWriter = (*generationAssetWriterStub)(nil)
