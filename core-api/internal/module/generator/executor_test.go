package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type imageGenerationServiceStub struct {
	events  *[]string
	request *imageclient.GenerateRequest
	result  *imageclient.GenerateResult
	err     error
}

type imageProcessorStub struct {
	events         *[]string
	resizeRequests []*imageprocessor.ResizeRequest
	err            error
}

type referenceStoreStub struct {
	resolved   []string
	persisted  []string
	resolveErr error
	persistErr error
}

func (s *referenceStoreStub) ResolveReference(_ context.Context, reference string) (string, error) {
	s.resolved = append(s.resolved, reference)
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
	return "signed:" + reference, nil
}

func (s *referenceStoreStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persisted = append(s.persisted, reference)
	if s.persistErr != nil {
		return "", s.persistErr
	}
	return fmt.Sprintf("uploads/generated-%d.png", len(s.persisted)), nil
}

func (s *imageProcessorStub) RemoveBackground(
	_ context.Context,
	request *imageprocessor.RemoveBackgroundRequest,
) (*imageprocessor.RemoveBackgroundResult, error) {
	*s.events = append(*s.events, "process_image")
	if s.err != nil {
		return nil, s.err
	}
	return &imageprocessor.RemoveBackgroundResult{
		ImageBase64: request.ImageBase64,
		MIMEType:    "image/png",
	}, nil
}

func (s *imageProcessorStub) Resize(
	_ context.Context,
	request *imageprocessor.ResizeRequest,
) (*imageprocessor.ResizeResult, error) {
	*s.events = append(*s.events, "resize_image")
	s.resizeRequests = append(s.resizeRequests, request)
	if s.err != nil {
		return nil, s.err
	}
	return &imageprocessor.ResizeResult{ImageBase64: request.ImageBase64, MIMEType: "image/png"}, nil
}

func (s *imageProcessorStub) Verify(
	_ context.Context,
	_ *imageprocessor.VerifyRequest,
) (*imageprocessor.VerificationReport, error) {
	return &imageprocessor.VerificationReport{Passed: true}, nil
}

func (s *imageProcessorStub) SplitImage(
	_ context.Context,
	_ *imageprocessor.SplitImageRequest,
) (*imageprocessor.SplitImageResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &imageprocessor.SplitImageResult{}, nil
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
	asset            assetdomain.Asset
}

func (s *generationAssetWriterStub) GetDetail(_ context.Context, assetID uint) (assetdomain.Asset, error) {
	if s.err != nil {
		return assetdomain.Asset{}, s.err
	}
	if s.asset.ID != 0 {
		return s.asset, nil
	}
	return assetdomain.Asset{
		ID:          assetID,
		Type:        assetdomain.AssetTypeCharacter,
		Perspective: assetdomain.PerspectiveTopDown,
		Scale:       json.RawMessage(`{"width":64,"height":64}`),
	}, nil
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

func TestExecutorGeneratesCharacterPrototypeBeforeCreatingAsset(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{
		events: &events,
		result: generatedImages(),
	}
	assets := &generationAssetWriterStub{events: &events}
	processor := &imageProcessorStub{events: &events}
	executor := generator.NewExecutor(images, processor, assets)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
			"scale":{"width":64,"height":64},
		"perspective":"Top-Down",
		"reference":"https://cdn.example/reference.png",
		"project_id":11
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload)
	if err != nil {
		t.Fatalf("generate character prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"resize_image",
		"resize_image",
		"create_character_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || images.request.Prompt != "pixel knight" ||
		images.request.Size != "" ||
		!reflect.DeepEqual(images.request.ReferenceImages, []string{"https://cdn.example/reference.png"}) {
		t.Fatalf("unexpected image request: %+v", images.request)
	}
	if len(processor.resizeRequests) != 2 || processor.resizeRequests[0].Options.Width != 64 || processor.resizeRequests[0].Options.Height != 64 {
		t.Fatalf("asset scale was not passed to processor: %+v", processor.resizeRequests)
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
	if assets.characterAsset.Perspective != assetdomain.PerspectiveTopDown || content.DirectionCount != 4 {
		t.Fatalf("unexpected character content: %+v", content)
	}
	if string(assets.characterAsset.Scale) != `{"width":64,"height":64}` {
		t.Fatalf("unexpected character scale: %s", assets.characterAsset.Scale)
	}
	assertPrototypeResources(t, assets.characterAsset)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 41})
}

func TestExecutorDerivesCharacterDirectionCountFromPerspective(t *testing.T) {
	for _, test := range []struct {
		perspective assetdomain.Perspective
		want        uint
	}{
		{perspective: assetdomain.PerspectiveSideOn, want: 2},
		{perspective: assetdomain.PerspectiveTopDown, want: 4},
		{perspective: assetdomain.PerspectiveIsometric, want: 8},
	} {
		t.Run(string(test.perspective), func(t *testing.T) {
			events := []string{}
			assets := &generationAssetWriterStub{events: &events}
			executor := generator.NewExecutor(
				&imageGenerationServiceStub{events: &events, result: generatedImages()},
				&imageProcessorStub{events: &events},
				assets,
			)

			payload := json.RawMessage(fmt.Sprintf(`{
			"asset_name":"hero",
			"creative_brief":"pixel knight",
			"scale":{"width":64,"height":64},
			"perspective":%q
		}`, test.perspective))
			if _, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload); err != nil {
				t.Fatalf("generate character prototype: %v", err)
			}
			content, err := assets.characterAsset.DecodeContent()
			if err != nil {
				t.Fatalf("decode character content: %v", err)
			}
			if assets.characterAsset.Perspective != test.perspective || content.DirectionCount != test.want {
				t.Fatalf("unexpected character asset: %+v content=%+v", assets.characterAsset, content)
			}
		})
	}
}

func TestExecutorResolvesReferencesAtExecutionAndPersistsGeneratedImagesAsKeys(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	references := &referenceStoreStub{}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets, references)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
			"scale":{"width":64,"height":64},
		"perspective":"Top-Down",
		"reference":"projects/7/reference.png",
		"project_id":11
	}`)

	if _, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload); err != nil {
		t.Fatalf("generate prototype: %v", err)
	}
	if len(references.resolved) != 1 || references.resolved[0] != "projects/7/reference.png" {
		t.Fatalf("expected execution-time reference resolution, got %v", references.resolved)
	}
	if len(references.persisted) != 2 || references.persisted[0] != "data:image/png;base64,first" || references.persisted[1] != "data:image/png;base64,second" {
		t.Fatalf("expected generated images to be persisted as data URLs, got %v", references.persisted)
	}
	content, err := assets.characterAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode generated asset: %v", err)
	}
	if *(*content.Prototype)[0].URL != "uploads/generated-1.png" || *(*content.Prototype)[1].URL != "uploads/generated-2.png" {
		t.Fatalf("expected object keys in generated asset: %+v", content.Prototype)
	}
}

func TestExecutorGeneratesObjectPrototypeBeforeCreatingAsset(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)
	payload := json.RawMessage(`{
		"asset_name":"chest",
		"creative_brief":"wooden chest",
			"scale":{"width":128,"height":128},
		"perspective":"Isometric",
		"project_id":12
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if err != nil {
		t.Fatalf("generate object prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"process_image",
		"resize_image",
		"process_image",
		"resize_image",
		"create_object_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if assets.objectAsset == nil || assets.objectAsset.Name != "chest" ||
		assets.objectAsset.ProjectID != 12 || assets.objectAsset.Type != assetdomain.AssetTypeObject {
		t.Fatalf("unexpected object asset: %+v", assets.objectAsset)
	}
	if assets.objectAsset.Perspective != assetdomain.PerspectiveIsometric {
		t.Fatalf("unexpected object perspective: %q", assets.objectAsset.Perspective)
	}
	assertPrototypeResources(t, assets.objectAsset)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 42})
}

func TestExecutorGeneratesAnimationBeforeUpdatingFrames(t *testing.T) {
	tests := []generator.TaskType{
		generator.GenerateAnimation,
	}
	for _, taskType := range tests {
		t.Run(string(taskType), func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			assets := &generationAssetWriterStub{events: &events}
			executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)
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
				"resize_image",
				"resize_image",
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
				*assets.frames[1].URL != "data:image/png;base64,second" {
				t.Fatalf("unexpected animation frames: %+v", assets.frames)
			}
			assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, AnimationID: 3})
		})
	}
}

func TestExecutorRejectsAnimationForNonFrameAssetTypes(t *testing.T) {
	for _, assetType := range []assetdomain.AssetType{
		assetdomain.AssetTypeTileSet,
		assetdomain.AssetTypeUISet,
		assetdomain.AssetTypeScenery,
		assetdomain.AssetTypeAudio,
	} {
		t.Run(string(assetType), func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			assets := &generationAssetWriterStub{
				events: &events,
				asset: assetdomain.Asset{
					ID:    7,
					Type:  assetType,
					Scale: json.RawMessage(`{"width":64,"height":64}`),
				},
			}
			executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)

			_, err := executor.Generate(context.Background(), generator.GenerateAnimation, json.RawMessage(`{"parent_id":7}`))
			if err == nil {
				t.Fatal("expected unsupported asset type error")
			}
			if len(events) != 0 {
				t.Fatalf("animation should stop before image generation: %v", events)
			}
		})
	}
}

func TestExecutorDoesNotMutateAssetsWhenImageGenerationFails(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, err: wantErr}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)

	_, err := executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"asset_name":"walk","parent_id":7}`),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected image generation error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"generate_image"}) {
		t.Fatalf("asset changed before image generation succeeded: %v", events)
	}
}

func TestExecutorRejectsInvalidPrototypePerspectiveBeforeImageGeneration(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
		"scale":{"width":64,"height":64},
		"perspective":"top-down",
		"project_id":11
	}`)

	_, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if len(events) != 0 {
		t.Fatalf("workflow should stop before side effects: %v", events)
	}
}

func TestExecutorRequiresDependencies(t *testing.T) {
	executor := generator.NewExecutor(nil, nil, nil)
	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageServiceRequired) {
		t.Fatalf("expected image service required error, got %v", err)
	}

	events := []string{}
	executor = generator.NewExecutor(&imageGenerationServiceStub{events: &events}, nil, nil)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrAssetWriterRequired) {
		t.Fatalf("expected asset writer required error, got %v", err)
	}

	executor = generator.NewExecutor(
		&imageGenerationServiceStub{events: &events},
		nil,
		&generationAssetWriterStub{events: &events},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateObjectProtoType, nil)
	if !errors.Is(err, generator.ErrImageProcessorRequired) {
		t.Fatalf("expected image processor required error, got %v", err)
	}
}

func generatedImages() *imageclient.GenerateResult {
	return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{
		{Base64: "first", MediaType: "image/png"},
		{Base64: "second", MediaType: "image/webp"},
	}}
}

func assertPrototypeResources(t *testing.T, asset *assetdomain.Asset) {
	t.Helper()
	if asset == nil {
		t.Fatal("expected created asset")
	}
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if content.Prototype == nil || len(*content.Prototype) != 2 {
		t.Fatalf("unexpected prototype: %+v", content.Prototype)
	}
	prototype := *content.Prototype
	if prototype[0].ID != 1 || prototype[0].URL == nil ||
		*prototype[0].URL != "data:image/png;base64,first" ||
		prototype[1].ID != 2 || prototype[1].URL == nil ||
		*prototype[1].URL != "data:image/png;base64,second" {
		t.Fatalf("unexpected prototype resources: %+v", prototype)
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
var _ imageprocessor.Processor = (*imageProcessorStub)(nil)
var _ generator.AssetWriter = (*generationAssetWriterStub)(nil)
