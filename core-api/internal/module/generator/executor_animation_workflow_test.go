package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

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
	references := &executorReferenceStoreStub{}
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
			executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	references := &executorReferenceStoreStub{persistValue: "https://private.example/frame.png?token=temporary"}
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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor = generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
	executor := generator.NewExecutorWithAnimation(nil, animations, nil, assets, &executorReferenceStoreStub{})

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
