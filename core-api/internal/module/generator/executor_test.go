package generator_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
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

type referenceUpload struct {
	key       string
	reference string
}

type referenceStoreStub struct {
	resolved   []string
	persisted  []string
	uploads    []referenceUpload
	events     *[]string
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

func (s *referenceStoreStub) NewObjectKey(_ string) (string, error) {
	if s.events != nil {
		*s.events = append(*s.events, "allocate_key")
	}
	return "uploads/prototype.png", nil
}

func (s *referenceStoreStub) PersistReferenceAt(_ context.Context, key, reference string) error {
	if s.events != nil {
		*s.events = append(*s.events, "persist:"+key)
	}
	s.uploads = append(s.uploads, referenceUpload{key: key, reference: reference})
	if s.persistErr != nil {
		return s.persistErr
	}
	return nil
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
	request *imageprocessor.SplitImageRequest,
) (*imageprocessor.SplitImageResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "split_image")
	}
	if s.err != nil {
		return nil, s.err
	}
	regionCount := request.Columns * request.Rows
	regions := make([]imageprocessor.ImageRegion, regionCount)
	for index := range regions {
		regions[index] = imageprocessor.ImageRegion{
			Index: index, ImageBase64: fmt.Sprintf("direction-%d", index), MIMEType: "image/png",
		}
	}
	return &imageprocessor.SplitImageResult{Regions: regions}, nil
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
	sceneryAsset     *assetdomain.Asset
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
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
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

func (s *generationAssetWriterStub) CreateSceneryAsset(
	_ context.Context,
	value *assetdomain.Asset,
) (uint, error) {
	if s.events != nil {
		*s.events = append(*s.events, "create_scenery_asset")
	}
	s.sceneryAsset = value
	if s.err != nil {
		return 0, s.err
	}
	return 43, nil
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
			"dimensions":{"width":64,"height":64},
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
		"process_image",
		"split_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"create_character_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || !strings.Contains(images.request.Prompt, "pixel knight") ||
		!strings.Contains(images.request.Prompt, "<direction_count>\n4\n</direction_count>") ||
		images.request.Size != "" ||
		!reflect.DeepEqual(images.request.ReferenceImages, []string{"https://cdn.example/reference.png"}) {
		t.Fatalf("unexpected image request: %+v", images.request)
	}
	if len(processor.resizeRequests) != 4 || processor.resizeRequests[0].Options.Width != 64 || processor.resizeRequests[0].Options.Height != 64 {
		t.Fatalf("asset dimensions were not passed to processor: %+v", processor.resizeRequests)
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
	if string(assets.characterAsset.Dimensions) != `{"width":64,"height":64}` {
		t.Fatalf("unexpected character dimensions: %s", assets.characterAsset.Dimensions)
	}
	assertPrototypeResources(t, assets.characterAsset, 4)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 41})
}

func TestExecutorDerivesCharacterDirectionCountFromPerspectiveAndIgnoresLegacyInput(t *testing.T) {
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
			"dimensions":{"width":64,"height":64},
			"perspective":%q,
			"direction_count":"1"
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
	references := &referenceStoreStub{events: &events}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets, references)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
			"dimensions":{"width":64,"height":64},
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
	if len(references.uploads) != 8 {
		t.Fatalf("expected four unprocessed and four final uploads, got %d: %+v", len(references.uploads), references.uploads)
	}
	wantEvents := []string{"generate_image", "process_image", "split_image", "allocate_key"}
	for index := range 4 {
		wantEvents = append(wantEvents,
			fmt.Sprintf("persist:uploads/prototype-%d-unprocessed.png", index),
			"resize_image",
			fmt.Sprintf("persist:uploads/prototype-%d.png", index),
		)
	}
	wantEvents = append(wantEvents, "create_character_asset")
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected raw/final upload order: %v", events)
	}
	content, err := assets.characterAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode generated asset: %v", err)
	}
	if *(*content.Prototype)[0].URL != "uploads/prototype-0.png" ||
		*(*content.Prototype)[1].URL != "uploads/prototype-1.png" ||
		*(*content.Prototype)[2].URL != "uploads/prototype-2.png" ||
		*(*content.Prototype)[3].URL != "uploads/prototype-3.png" {
		t.Fatalf("expected object keys in generated asset: %+v", content.Prototype)
	}
	for index := range 4 {
		uploadOffset := index * 2
		if references.uploads[uploadOffset].key != fmt.Sprintf("uploads/prototype-%d-unprocessed.png", index) {
			t.Fatalf("unexpected unprocessed key at %d: %+v", index, references.uploads[uploadOffset])
		}
		if references.uploads[uploadOffset+1].key != fmt.Sprintf("uploads/prototype-%d.png", index) {
			t.Fatalf("unexpected final key at %d: %+v", index, references.uploads[uploadOffset+1])
		}
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
		"dimensions":{"width":128,"height":128},
		"perspective":"Isometric",
		"direction_count":"4",
		"project_id":12
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if err != nil {
		t.Fatalf("generate object prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"process_image",
		"split_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
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
	if images.request == nil || !strings.Contains(images.request.Prompt, "<direction_count>\n8\n</direction_count>") {
		t.Fatalf("object prompt did not include derived direction count: %+v", images.request)
	}
	assertPrototypeResources(t, assets.objectAsset, 8)
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
				assets.animationName != "walk" || len(assets.frames) != 1 {
				t.Fatalf("unexpected animation update: %+v", assets)
			}
			if assets.frames[0].ID != 1 || assets.frames[0].URL == nil ||
				*assets.frames[0].URL != "data:image/png;base64,sheet" {
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
					ID:         7,
					Type:       assetType,
					Dimensions: json.RawMessage(`{"width":64,"height":64}`),
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
		"dimensions":{"width":64,"height":64},
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
		{Base64: "sheet", MediaType: "image/png"},
	}}
}

func assertPrototypeResources(t *testing.T, asset *assetdomain.Asset, wantCount int) {
	t.Helper()
	if asset == nil {
		t.Fatal("expected created asset")
	}
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode asset content: %v", err)
	}
	if content.Prototype == nil || len(*content.Prototype) != wantCount {
		t.Fatalf("unexpected prototype: %+v", content.Prototype)
	}
	prototype := *content.Prototype
	for index, resource := range prototype {
		if resource.ID != uint(index+1) || resource.URL == nil ||
			*resource.URL != fmt.Sprintf("data:image/png;base64,direction-%d", index) {
			t.Fatalf("unexpected prototype resource at %d: %+v", index, resource)
		}
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

type sceneryExecutorLLMStub struct {
	events   *[]string
	requests []*llmclient.CompletionRequest
	results  []*llmclient.CompletionResult
}

func (s *sceneryExecutorLLMStub) Complete(
	_ context.Context,
	request *llmclient.CompletionRequest,
) (*llmclient.CompletionResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "llm")
	}
	s.requests = append(s.requests, &llmclient.CompletionRequest{
		Prompt: request.Prompt,
		Images: append([]llmclient.ImageInput(nil), request.Images...),
		Model:  request.Model,
		ResponseSchema: llmclient.JSONSchema{
			Name:   request.ResponseSchema.Name,
			Schema: append(json.RawMessage(nil), request.ResponseSchema.Schema...),
		},
	})
	call := len(s.requests) - 1
	if call >= len(s.results) {
		return nil, errors.New("missing LLM result")
	}
	return s.results[call], nil
}

type sceneryExecutorImageStub struct {
	events   *[]string
	requests []*imageclient.GenerateRequest
	results  []*imageclient.GenerateResult
}

func (s *sceneryExecutorImageStub) Generate(
	_ context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	if s.events != nil {
		*s.events = append(*s.events, "image")
	}
	s.requests = append(s.requests, &imageclient.GenerateRequest{
		Prompt:          request.Prompt,
		ReferenceImages: append([]string(nil), request.ReferenceImages...),
		Model:           request.Model,
		Size:            request.Size,
		Params:          request.Params,
	})
	call := len(s.requests) - 1
	if call >= len(s.results) {
		return nil, errors.New("missing image result")
	}
	return s.results[call], nil
}

type sceneryExecutorProcessorStub struct {
	events *[]string
}

func (s *sceneryExecutorProcessorStub) RemoveBackground(
	_ context.Context,
	request *imageprocessor.RemoveBackgroundRequest,
) (*imageprocessor.RemoveBackgroundResult, error) {
	*s.events = append(*s.events, "remove")
	return &imageprocessor.RemoveBackgroundResult{
		ImageBase64: "removed:" + request.ImageBase64,
		MIMEType:    "image/png",
	}, nil
}

func (s *sceneryExecutorProcessorStub) Resize(
	_ context.Context,
	request *imageprocessor.ResizeRequest,
) (*imageprocessor.ResizeResult, error) {
	*s.events = append(*s.events, "resize")
	return &imageprocessor.ResizeResult{
		ImageBase64: base64.StdEncoding.EncodeToString([]byte("processed:" + request.ImageBase64)),
		MIMEType:    "image/png",
	}, nil
}

func (s *sceneryExecutorProcessorStub) Verify(
	_ context.Context,
	_ *imageprocessor.VerifyRequest,
) (*imageprocessor.VerificationReport, error) {
	*s.events = append(*s.events, "verify")
	return &imageprocessor.VerificationReport{Passed: true}, nil
}

func (*sceneryExecutorProcessorStub) SplitImage(
	context.Context,
	*imageprocessor.SplitImageRequest,
) (*imageprocessor.SplitImageResult, error) {
	return &imageprocessor.SplitImageResult{}, nil
}

type sceneryResourceStoreStub struct {
	keys      []string
	data      [][]byte
	deleted   []string
	putErrAt  int
	putErr    error
	deleteErr error
	cancelAt  int
	cancel    context.CancelFunc
	deleteCtx []error
}

func (s *sceneryResourceStoreStub) PutObject(
	_ context.Context,
	key string,
	_ string,
	data []byte,
) error {
	call := len(s.keys) + 1
	if s.putErrAt == call {
		return s.putErr
	}
	s.keys = append(s.keys, key)
	s.data = append(s.data, append([]byte(nil), data...))
	if s.cancelAt == call && s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *sceneryResourceStoreStub) DeleteObject(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	s.deleteCtx = append(s.deleteCtx, ctx.Err())
	return s.deleteErr
}

func TestExecutorPlansAndAnalyzesSceneryAroundLayerGeneration(t *testing.T) {
	events := []string{}
	images := &sceneryExecutorImageStub{
		events: &events,
		results: []*imageclient.GenerateResult{
			{Images: []imageclient.GeneratedImage{{Base64: "sky-source", MediaType: "image/webp"}}},
			{Images: []imageclient.GeneratedImage{{Base64: "mountain-source", MediaType: "image/jpeg"}}},
		},
	}
	llm := &sceneryExecutorLLMStub{
		events: &events,
		results: []*llmclient.CompletionResult{
			{JSON: json.RawMessage(`{"layers":[{"name":"Sky","creative_brief":"warm sky filling the full canvas"},{"name":"Mountains","creative_brief":"distant peaks along the lower third"}]}`)},
			{JSON: json.RawMessage(`{"layers":[{"id":2,"position":{"x":100,"y":40},"scale":{"x":0.8,"y":0.8},"rotation":0,"opacity":0.75,"zIndex":20},{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-10}]}`)},
		},
	}
	processor := &sceneryExecutorProcessorStub{events: &events}
	assets := &generationAssetWriterStub{events: &events}
	resources := &sceneryResourceStoreStub{}
	executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{
		LLM: llm, Resources: resources,
	})
	payload, err := json.Marshal(generator.CreateSceneryPayload{
		AssetName:     "Mountain Valley",
		CreativeBrief: "A valley at dawn",
		Style:         "pixel art",
		Dimensions:    assetdomain.Size{Width: 640, Height: 360},
		Perspective:   "Side-On",
		ProjectContext: generator.SceneryProjectContext{
			Name: "Starbound", GameType: "RPG", TargetPlatform: "PC", Description: "quiet exploration",
		},
		ProjectID: 42,
	})
	if err != nil {
		t.Fatalf("marshal scenery payload: %v", err)
	}

	result, err := executor.Generate(context.Background(), generator.GenerateScenery, payload)
	if err != nil {
		t.Fatalf("generate scenery: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"llm", "image", "remove", "resize", "verify", "image", "remove", "resize", "verify", "llm", "create_scenery_asset",
	}) {
		t.Fatalf("unexpected generation workflow: %v", events)
	}
	if len(llm.requests) != 2 || len(llm.requests[0].Images) != 0 || len(llm.requests[1].Images) != 2 {
		t.Fatalf("expected text planning followed by one multi-image layout call: %+v", llm.requests)
	}
	if llm.requests[0].ResponseSchema.Name != "scenery_layer_plan" ||
		!strings.Contains(llm.requests[0].Prompt, "A valley at dawn") ||
		llm.requests[1].ResponseSchema.Name != "scenery_layer_layout" ||
		!strings.Contains(llm.requests[1].Prompt, `Attached image 1 corresponds to layer ID 1 named "Sky"`) {
		t.Fatalf("unexpected LLM requests: %+v", llm.requests)
	}
	if len(images.requests) != 2 ||
		images.requests[0].Size != "640x360" || images.requests[1].Size != "640x360" ||
		!strings.Contains(images.requests[0].Prompt, "warm sky filling the full canvas") ||
		!strings.Contains(images.requests[1].Prompt, "distant peaks along the lower third") {
		t.Fatalf("planner output was not passed to image prompts: %+v", images.requests)
	}

	var decoded generator.ExecutionResult
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("decode scenery result: %v", err)
	}
	if decoded.AssetID != 43 || assets.sceneryAsset == nil || len(resources.keys) != 2 {
		t.Fatalf("unexpected persisted scenery result: result=%+v asset=%+v keys=%v", decoded, assets.sceneryAsset, resources.keys)
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

func TestExecutorCleansUpUploadedSceneryResourcesWhenUploadFails(t *testing.T) {
	wantErr := errors.New("object storage unavailable")
	assets := &generationAssetWriterStub{}
	resources := &sceneryResourceStoreStub{putErrAt: 2, putErr: wantErr}
	executor := newSceneryPersistenceExecutor(assets, resources)
	_, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPersistencePayload(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected upload error, got %v", err)
	}
	if assets.sceneryAsset != nil || len(resources.keys) != 1 || len(resources.deleted) != 1 || resources.deleted[0] != resources.keys[0] {
		t.Fatalf("unexpected cleanup: asset=%+v keys=%v deleted=%v", assets.sceneryAsset, resources.keys, resources.deleted)
	}
}

func TestExecutorCleansUpSceneryResourcesWhenAssetCreationFails(t *testing.T) {
	wantErr := errors.New("database unavailable")
	assets := &generationAssetWriterStub{err: wantErr}
	resources := &sceneryResourceStoreStub{}
	executor := newSceneryPersistenceExecutor(assets, resources)
	_, err := executor.Generate(context.Background(), generator.GenerateScenery, sceneryPersistencePayload(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected database error, got %v", err)
	}
	if len(resources.keys) != 2 || len(resources.deleted) != 2 || resources.deleted[0] != resources.keys[1] || resources.deleted[1] != resources.keys[0] {
		t.Fatalf("expected reverse-order cleanup: keys=%v deleted=%v", resources.keys, resources.deleted)
	}
}

func TestExecutorUsesFreshContextToCleanUpCancelledScenery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	resources := &sceneryResourceStoreStub{cancelAt: 1, cancel: cancel}
	executor := newSceneryPersistenceExecutor(&generationAssetWriterStub{}, resources)
	_, err := executor.Generate(ctx, generator.GenerateScenery, sceneryPersistencePayload(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if len(resources.deleted) != 1 || len(resources.deleteCtx) != 1 || resources.deleteCtx[0] != nil {
		t.Fatalf("cleanup reused cancelled context: deleted=%v contextErrors=%v", resources.deleted, resources.deleteCtx)
	}
}

func newSceneryPersistenceExecutor(assets generator.AssetWriter, resources generator.ResourceStore) generator.Executor {
	events := []string{}
	return generator.NewExecutorWithDependencies(
		&sceneryExecutorImageStub{results: []*imageclient.GenerateResult{
			{Images: []imageclient.GeneratedImage{{Base64: "sky-source", MediaType: "image/webp"}}},
			{Images: []imageclient.GeneratedImage{{Base64: "mountain-source", MediaType: "image/jpeg"}}},
		}},
		&sceneryExecutorProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{LLM: validSceneryPersistenceLLM(), Resources: resources},
	)
}

func validSceneryPersistenceLLM() *sceneryExecutorLLMStub {
	return &sceneryExecutorLLMStub{results: []*llmclient.CompletionResult{
		{JSON: json.RawMessage(`{"layers":[{"name":"Sky","creative_brief":"warm sky"},{"name":"Mountains","creative_brief":"distant peaks"}]}`)},
		{JSON: json.RawMessage(`{"layers":[{"id":1,"position":{"x":0,"y":0},"scale":{"x":1,"y":1},"rotation":0,"opacity":1,"zIndex":-10},{"id":2,"position":{"x":100,"y":40},"scale":{"x":0.8,"y":0.8},"rotation":0,"opacity":0.75,"zIndex":20}]}`)},
	}}
}

func sceneryPersistencePayload(t *testing.T) json.RawMessage {
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

var _ imageclient.ImageGenerationService = (*imageGenerationServiceStub)(nil)
var _ imageprocessor.Processor = (*imageProcessorStub)(nil)
var _ generator.AssetWriter = (*generationAssetWriterStub)(nil)
var _ imageclient.ImageGenerationService = (*sceneryExecutorImageStub)(nil)
var _ llmclient.LLMService = (*sceneryExecutorLLMStub)(nil)
var _ imageprocessor.Processor = (*sceneryExecutorProcessorStub)(nil)
var _ generator.ResourceStore = (*sceneryResourceStoreStub)(nil)
