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

type animationGenerationServiceStub struct {
	events  *[]string
	request *generator.AnimationGenerationRequest
	result  *generator.AnimationGenerationResult
	err     error
}

func (s *animationGenerationServiceStub) Generate(
	_ context.Context,
	request *generator.AnimationGenerationRequest,
) (*generator.AnimationGenerationResult, error) {
	*s.events = append(*s.events, "generate_animation")
	copy := *request
	s.request = &copy
	return s.result, s.err
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
	resolved     []string
	persisted    []string
	persistValue string
	uploads      []referenceUpload
	events       *[]string
	resolveErr   error
	persistErr   error
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
	if s.persistValue != "" {
		return s.persistValue, nil
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
	parentAsset      assetdomain.Asset
	getDetailErr     error
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

func (s *generationAssetWriterStub) GetDetail(
	_ context.Context,
	assetID uint,
) (assetdomain.Asset, error) {
	if s.events != nil {
		*s.events = append(*s.events, "get_asset")
	}
	if s.getDetailErr != nil {
		return assetdomain.Asset{}, s.getDetailErr
	}
	if s.parentAsset.ID == assetID {
		return s.parentAsset, nil
	}
	if s.asset.ID != 0 {
		return s.asset, nil
	}
	return assetdomain.Asset{}, nil
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
	frames []assetdomain.Frame,
) (uint, error) {
	*s.events = append(*s.events, "create_animation")
	s.animationAssetID = assetID
	s.animationName = name
	s.frames = append([]assetdomain.Frame(nil), frames...)
	if s.err != nil {
		return 0, s.err
	}
	s.animationID = 3
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
		"direction_count":"4",
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
	content, err := assets.objectAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode object content: %v", err)
	}
	if content.DirectionCount != 8 {
		t.Fatalf("unexpected object content: %+v", content)
	}
	if images.request == nil || !strings.Contains(images.request.Prompt, "<direction_count>\n8\n</direction_count>") {
		t.Fatalf("object prompt did not include derived direction count: %+v", images.request)
	}
	assertPrototypeResources(t, assets.objectAsset, 8)
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 42})
}

func TestExecutorGeneratesAnimationBeforeUpdatingFrames(t *testing.T) {
	events := []string{}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: "first", MIMEType: "image/png"},
				{Index: 1, ImageBase64: "second", MIMEType: "image/png"},
			},
			VideoRequestID:  "request-1",
			VideoAttempts:   1,
			FrameDurationMS: 100,
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
	references := &referenceStoreStub{}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, references)
	payload := json.RawMessage(`{
		"animation_name":"  walk  ",
		"creative_brief":"walking cycle",
		"asset_id":7,
		"project_id":11,
		"direction":"back_right",
		"style":"painted pixel art",
		"frame_count":8,
		"columns":4,
		"frame_width":128,
		"frame_height":128,
		"fps":12,
		"resolution":"1080p",
		"duration":8,
		"aspect_ratio":"1:1"
	}`)

	result, err := executor.Generate(context.Background(), generator.GenerateAnimation, payload)
	if err != nil {
		t.Fatalf("generate animation: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"get_asset",
		"generate_animation",
		"create_animation",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	wantRequest := &generator.AnimationGenerationRequest{
		Description:            "silver-haired knight",
		Style:                  "painted pixel art",
		Action:                 "walking cycle",
		ReferenceImage:         "https://cdn.example.com/hero/direction_03-unprocessed.png?version=7",
		ReferenceImagePrepared: false,
		FrameCount:             8,
		Columns:                4,
		FrameWidth:             128,
		FrameHeight:            128,
		FPS:                    12,
		Resolution:             "1080p",
		Duration:               8,
		AspectRatio:            "1:1",
	}
	if !reflect.DeepEqual(animations.request, wantRequest) {
		t.Fatalf("unexpected animation request: got %+v want %+v", animations.request, wantRequest)
	}
	if assets.animationAssetID != 7 || assets.animationID != 3 ||
		assets.animationName != "walk" || len(assets.frames) != 2 {
		t.Fatalf("unexpected animation update: %+v", assets)
	}
	if assets.frames[0].ID != 1 || assets.frames[0].URL == nil ||
		*assets.frames[0].URL != "uploads/generated-1.png" ||
		assets.frames[0].Duration != 100 ||
		assets.frames[1].ID != 2 || assets.frames[1].URL == nil ||
		*assets.frames[1].URL != "uploads/generated-2.png" ||
		assets.frames[1].Duration != 100 {
		t.Fatalf("unexpected animation frames: %+v", assets.frames)
	}
	if len(assets.frames[0].Metadata) != 0 || len(assets.frames[1].Metadata) != 0 {
		t.Fatalf("animation frames should not persist generator metadata: %+v", assets.frames)
	}
	if !reflect.DeepEqual(references.persisted, []string{
		"data:image/png;base64,first",
		"data:image/png;base64,second",
	}) {
		t.Fatalf("unexpected persisted animation frame inputs: %v", references.persisted)
	}
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, AnimationID: 3})
}

func TestExecutorRejectsMissingAnimationIdentityBeforeLookup(t *testing.T) {
	tests := []struct {
		name    string
		payload json.RawMessage
		want    string
	}{
		{
			name:    "asset id",
			payload: json.RawMessage(`{"animation_name":"walk"}`),
			want:    "animation asset is required",
		},
		{
			name:    "animation name",
			payload: json.RawMessage(`{"animation_name":"   ","asset_id":7}`),
			want:    "animation name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []string{}
			animations := &animationGenerationServiceStub{events: &events}
			assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
			executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

			_, err := executor.Generate(context.Background(), generator.GenerateAnimation, tt.payload)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
			if len(events) != 0 {
				t.Fatalf("animation validation should run before asset lookup or generation: %v", events)
			}
		})
	}
}

func TestExecutorRejectsNonObjectKeyAnimationFrameReference(t *testing.T) {
	events := []string{}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame", MIMEType: "image/png"}},
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
	references := &referenceStoreStub{persistValue: "https://private.example/frame.png?token=temporary"}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, references)

	_, err := executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"walk","asset_id":7,"direction":"front"}`),
	)
	if err == nil || !strings.Contains(err.Error(), "storage returned a non-object-key reference") {
		t.Fatalf("expected object-key validation error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
		t.Fatalf("asset should not change when frame persistence is invalid: %v", events)
	}
}

func TestExecutorDoesNotMutateAssetsWhenAnimationGenerationFails(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	events := []string{}
	animations := &animationGenerationServiceStub{events: &events, err: wantErr}
	assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err := executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"walk","asset_id":7,"direction":"front"}`),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected animation generation error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
		t.Fatalf("asset changed before animation generation succeeded: %v", events)
	}
}

func TestExecutorMapsTwoDirectionAssetLeftRight(t *testing.T) {
	events := []string{}
	referenceLeft := "https://cdn.example.com/hero/left.png"
	referenceRight := "https://cdn.example.com/hero/right.png"
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 2
	prototype := assetdomain.Prototype{
		{ID: 1, URL: &referenceLeft},
		{ID: 2, URL: &referenceRight},
	}
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent := assetdomain.Asset{ID: 7, ProjectID: 11, Type: assetdomain.AssetTypeCharacter, Name: "hero", Content: encoded}
	animations := &animationGenerationServiceStub{events: &events, result: &generator.AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"idle","asset_id":7,"direction":"left"}`),
	)
	if err != nil {
		t.Fatalf("generate left animation: %v", err)
	}
	if animations.request.ReferenceImage != "https://cdn.example.com/hero/left-unprocessed.png" {
		t.Fatalf("left direction mapped to wrong reference: %+v", animations.request)
	}

	events = nil
	animations = &animationGenerationServiceStub{events: &events, result: &generator.AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}}
	assets = &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor = generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"idle","asset_id":7,"direction":"right"}`),
	)
	if err != nil {
		t.Fatalf("generate right animation: %v", err)
	}
	if animations.request.ReferenceImage != "https://cdn.example.com/hero/right-unprocessed.png" {
		t.Fatalf("right direction mapped to wrong reference: %+v", animations.request)
	}
}

func TestExecutorGeneratesObjectAnimationForSelectedDirection(t *testing.T) {
	events := []string{}
	prototypeURLs := []string{
		"https://cdn.example.com/chest/front.png",
		"https://cdn.example.com/chest/right.png",
		"https://cdn.example.com/chest/back.png",
		"https://cdn.example.com/chest/left.png",
	}
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeObject)
	content.DirectionCount = 4
	prototype := assetdomain.Prototype{}
	for index, value := range prototypeURLs {
		url := value
		prototype = append(prototype, assetdomain.ImageResource{ID: uint(index + 1), URL: &url})
	}
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent := assetdomain.Asset{
		ID:          8,
		ProjectID:   11,
		Type:        assetdomain.AssetTypeObject,
		Name:        "chest",
		Description: "wooden treasure chest",
		Content:     encoded,
	}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}},
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"open","asset_id":8,"project_id":11,"direction":"right","creative_brief":"slowly open the chest lid, then close it"}`),
	)
	if err != nil {
		t.Fatalf("generate object animation: %v", err)
	}
	if animations.request.ReferenceImage != "https://cdn.example.com/chest/right-unprocessed.png" {
		t.Fatalf("object animation mapped to wrong reference: %+v", animations.request)
	}
	if animations.request.Action != "slowly open the chest lid, then close it" {
		t.Fatalf("unexpected object action: %+v", animations.request)
	}
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation", "create_animation"}) {
		t.Fatalf("unexpected object animation workflow: %v", events)
	}
}

func TestExecutorRejectsObjectAnimationWithoutDirection(t *testing.T) {
	events := []string{}
	reference := "https://cdn.example.com/chest/front.png"
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeObject)
	content.DirectionCount = 4
	prototype := assetdomain.Prototype{{ID: 1, URL: &reference}}
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent := assetdomain.Asset{ID: 8, ProjectID: 11, Type: assetdomain.AssetTypeObject, Name: "chest", Content: encoded}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(context.Background(), generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"open","asset_id":8,"direction":""}`))
	if err == nil || !strings.Contains(err.Error(), "animation direction is required") {
		t.Fatalf("expected required object direction error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset"}) {
		t.Fatalf("generation should not start without direction: %v", events)
	}
}

func TestExecutorRejectsUnsupportedSingleDirectionAsset(t *testing.T) {
	events := []string{}
	reference := "https://cdn.example.com/hero/front.png"
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 1
	prototype := assetdomain.Prototype{{ID: 1, URL: &reference}}
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent := assetdomain.Asset{
		ID: 7, ProjectID: 11, Type: assetdomain.AssetTypeCharacter, Name: "hero", Content: encoded,
	}
	animations := &animationGenerationServiceStub{events: &events}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"idle","asset_id":7,"direction":"front"}`),
	)
	if err == nil || !strings.Contains(err.Error(), "direction count must be one of 2, 4, or 8") {
		t.Fatalf("expected unsupported direction-count error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset"}) {
		t.Fatalf("generation should not start for unsupported direction count: %v", events)
	}
}

func TestExecutorRejectsAnimationDirectionOutsidePrototypeOrder(t *testing.T) {
	events := []string{}
	assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err := executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"walk","asset_id":7,"direction":"up"}`),
	)
	if err == nil || !strings.Contains(err.Error(), `direction "up" is unavailable`) {
		t.Fatalf("expected invalid direction error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset"}) {
		t.Fatalf("generation should not start for an invalid direction: %v", events)
	}
}

func TestExecutorRejectsMultiDirectionParentWithoutAnimationReference(t *testing.T) {
	events := []string{}
	parent := animationParentAsset(t)
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 8
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent.Content = encoded
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &referenceStoreStub{})

	_, err = executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"walk","asset_id":7,"direction":"front"}`),
	)
	if err == nil || !strings.Contains(err.Error(), `no prototype for direction "front"`) {
		t.Fatalf("expected missing multi-direction reference error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset"}) {
		t.Fatalf("generation should not start without a prototype: %v", events)
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

	executor = generator.NewExecutor(nil, nil, &generationAssetWriterStub{events: &events})
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationServiceRequired) {
		t.Fatalf("expected animation service required error, got %v", err)
	}

	executor = generator.NewExecutorWithAnimation(
		nil,
		&animationGenerationServiceStub{events: &events},
		nil,
		&generationAssetWriterStub{events: &events},
	)
	_, err = executor.Generate(context.Background(), generator.GenerateAnimation, nil)
	if !errors.Is(err, generator.ErrAnimationReferenceStoreRequired) {
		t.Fatalf("expected animation reference store required error, got %v", err)
	}
}

func animationParentAsset(t *testing.T) assetdomain.Asset {
	t.Helper()
	animationReference := "data:image/png;base64,legacy-multi-direction-source"
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 8
	content.Metadata = map[string]any{"animation_reference": animationReference}
	prototype := make(assetdomain.Prototype, 0, content.DirectionCount)
	for direction := range content.DirectionCount {
		reference := fmt.Sprintf("https://cdn.example.com/hero/direction_%02d.png?version=7", direction)
		prototype = append(prototype, assetdomain.ImageResource{ID: uint(direction) + 1, URL: &reference})
	}
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode animation parent content: %v", err)
	}
	return assetdomain.Asset{
		ID:          7,
		ProjectID:   11,
		Type:        assetdomain.AssetTypeCharacter,
		Name:        "hero",
		Description: "silver-haired knight",
		Content:     encoded,
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

var _ imageclient.ImageGenerationService = (*imageGenerationServiceStub)(nil)
var _ generator.AnimationGenerationService = (*animationGenerationServiceStub)(nil)
var _ imageprocessor.Processor = (*imageProcessorStub)(nil)
var _ generator.AssetWriter = (*generationAssetWriterStub)(nil)
