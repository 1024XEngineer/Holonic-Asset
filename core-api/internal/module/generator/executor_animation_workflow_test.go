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
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestExecutorEditsAnimationUsingPersistedGeneration(t *testing.T) {
	events := []string{}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{
				{ImageBase64: "edited-first", MIMEType: "image/png"},
				{ImageBase64: "edited-second", MIMEType: "image/png"},
			},
			FrameDurationMS: 83,
		},
	}
	parent := animationParentAssetWithAnimation(t, &assetdomain.AnimationGenerationConfig{
		Direction:   "back_right",
		Style:       "painted pixel art",
		Action:      "walking cycle",
		FrameCount:  8,
		Columns:     4,
		FrameWidth:  128,
		FrameHeight: 128,
		FPS:         12,
		Resolution:  "1080p",
		Duration:    8,
		AspectRatio: "1:1",
	})
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	references := &executorReferenceStoreStub{}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: references,
	})

	result, err := executor.Generate(context.Background(), generator.EditAnimation, json.RawMessage(`{
		"asset_id":7,
		"animation_id":3,
		"project_id":11,
		"creative_brief":"attack with sword"
	}`))
	if err != nil {
		t.Fatalf("edit animation: %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
		t.Fatalf("unexpected edit workflow order: %v", events)
	}
	wantRequest := &generator.AnimationGenerationRequest{
		Description:            "silver-haired knight",
		Style:                  "painted pixel art",
		Action:                 "attack with sword",
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
		t.Fatalf("unexpected edit animation request: got %+v want %+v", animations.request, wantRequest)
	}
	application, content := decodeExecutionContent(t, result, assetdomain.AssetTypeCharacter)
	if content.Prototype != nil {
		t.Fatalf("animation edit result must not include the asset prototype: %+v", content.Prototype)
	}
	if len(content.Animations) != 1 || len(content.Animations[0].Frames) != 2 {
		t.Fatalf("generation must return the edited frames: result=%+v content=%+v", application, content)
	}
	frames := content.Animations[0].Frames
	if frames[0].ID != 1 || frames[0].URL == nil ||
		*frames[0].URL != "uploads/generated-1.png" || frames[0].Duration != 83 ||
		frames[1].ID != 2 || frames[1].URL == nil ||
		*frames[1].URL != "uploads/generated-2.png" {
		t.Fatalf("unexpected replacement frames: %+v", frames)
	}
	if len(references.persisted) != 2 ||
		references.persisted[0] != "data:image/png;base64,edited-first" ||
		references.persisted[1] != "data:image/png;base64,edited-second" {
		t.Fatalf("unexpected persisted edited frames: %#v", references.persisted)
	}
	if application.AssetID != 7 || application.AnimationID != 3 || len(application.GeneratedResources) != 2 {
		t.Fatalf("unexpected animation application candidate: %+v", application)
	}
}

func TestExecutorEditAnimationRejectsMissingGeneration(t *testing.T) {
	events := []string{}
	parent := animationParentAssetWithAnimation(t, nil)
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

	_, err := executor.Generate(context.Background(), generator.EditAnimation, json.RawMessage(`{
		"asset_id":7,"animation_id":3,"project_id":11,"creative_brief":"attack"
	}`))
	if err == nil || !strings.Contains(err.Error(), "has no generation configuration") {
		t.Fatalf("expected missing generation configuration error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset"}) {
		t.Fatalf("edit must stop before generation when metadata is missing: %v", events)
	}
}

func TestExecutorEditAnimationRejectsUnknownAnimationAndProjectMismatch(t *testing.T) {
	tests := []struct {
		name    string
		project uint
		id      uint
		want    string
	}{
		{name: "unknown animation", project: 11, id: 99, want: "animation 99 not found"},
		{name: "project mismatch", project: 12, id: 3, want: "belongs to project 11, not project 12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []string{}
			parent := animationParentAssetWithAnimation(t, &assetdomain.AnimationGenerationConfig{Direction: "front"})
			assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
			animations := &animationGenerationServiceStub{events: &events}
			executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
				Animations: animations,
				References: &executorReferenceStoreStub{},
			})
			payload := fmt.Sprintf(`{"asset_id":7,"animation_id":%d,"project_id":%d,"creative_brief":"attack"}`, tt.id, tt.project)
			_, err := executor.Generate(context.Background(), generator.EditAnimation, json.RawMessage(payload))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
			if !reflect.DeepEqual(events, []string{"get_asset"}) {
				t.Fatalf("edit must stop before generation: %v", events)
			}
		})
	}
}

func TestExecutorEditAnimationKeepsExistingFramesWhenRegenerationFails(t *testing.T) {
	events := []string{}
	parent := animationParentAssetWithAnimation(t, &assetdomain.AnimationGenerationConfig{Direction: "front"})
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	wantErr := errors.New("video provider unavailable")
	animations := &animationGenerationServiceStub{events: &events, err: wantErr}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

	_, err := executor.Generate(context.Background(), generator.EditAnimation, json.RawMessage(`{
		"asset_id":7,"animation_id":3,"project_id":11,"creative_brief":"attack"
	}`))
	if err == nil || !strings.Contains(err.Error(), "regenerate animation frames") {
		t.Fatalf("expected regeneration error, got %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("unexpected workflow after generation failure: %v", events)
	}
}

func TestExecutorEditAnimationValidatesIdentityAndPromptBeforeLookup(t *testing.T) {
	tests := []struct {
		name    string
		payload json.RawMessage
		want    string
	}{
		{name: "asset id", payload: json.RawMessage(`{"animation_id":3,"creative_brief":"attack"}`), want: "animation asset is required"},
		{name: "animation id", payload: json.RawMessage(`{"asset_id":7,"creative_brief":"attack"}`), want: "animation id is required"},
		{name: "creative brief", payload: json.RawMessage(`{"asset_id":7,"animation_id":3,"creative_brief":"   "}`), want: "animation creative brief is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []string{}
			assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
			executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
				Animations: &animationGenerationServiceStub{events: &events},
				References: &executorReferenceStoreStub{},
			})
			_, err := executor.Generate(context.Background(), generator.EditAnimation, tt.payload)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
			if len(events) != 0 {
				t.Fatalf("edit validation must happen before lookup: %v", events)
			}
		})
	}
}

func TestExecutorEditAnimationDoesNotReplaceFramesWhenPersistenceFails(t *testing.T) {
	events := []string{}
	parent := animationParentAssetWithAnimation(t, &assetdomain.AnimationGenerationConfig{Direction: "front"})
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "edited", MIMEType: "image/png"}},
		},
	}
	references := &executorReferenceStoreStub{persistErr: errors.New("storage unavailable")}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: references,
	})

	_, err := executor.Generate(context.Background(), generator.EditAnimation, json.RawMessage(`{
		"asset_id":7,"animation_id":3,"project_id":11,"creative_brief":"attack"
	}`))
	if err == nil || !strings.Contains(err.Error(), "persist animation frame 1") {
		t.Fatalf("expected persistence error, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
		t.Fatalf("unexpected workflow after persistence failure: %v", events)
	}
}

func animationParentAssetWithAnimation(t *testing.T, generation *assetdomain.AnimationGenerationConfig) assetdomain.Asset {
	t.Helper()
	parent := animationParentAsset(t)
	content, err := parent.DecodeContent()
	if err != nil {
		t.Fatalf("decode animation parent content: %v", err)
	}
	oldURL := "uploads/old-frame.png"
	content.Animations = []assetdomain.Animation{{
		ID:   3,
		Name: "walk",
		Frames: []assetdomain.Frame{{
			ID: 1, URL: &oldURL, Duration: 100,
		}},
		Generation: generation,
	}}
	parent.Content, err = assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode animation parent content with animation: %v", err)
	}
	return parent
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
			RawFrames: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: "raw-first", MIMEType: "image/png"},
				{Index: 1, ImageBase64: "raw-second", MIMEType: "image/png"},
			},
			VideoRequestID:  "request-1",
			VideoAttempts:   1,
			FrameDurationMS: 100,
		},
	}
	parent := animationParentAssetWithAnimation(t, &assetdomain.AnimationGenerationConfig{Direction: "front"})
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	references := &executorReferenceStoreStub{}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: references,
	})
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
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	wantRequest := &generator.AnimationGenerationRequest{
		Description:            "silver-haired knight",
		Style:                  "painted pixel art",
		Action:                 "walking cycle",
		ReferenceImage:         "https://cdn.example.com/hero/direction_03-unprocessed.png?version=7",
		ReferenceImagePrepared: false,
		FrameCount:             8,
		Columns:                3,
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
	application, content := decodeExecutionContent(t, result, assetdomain.AssetTypeCharacter)
	if content.Prototype != nil {
		t.Fatalf("animation generation result must not include the asset prototype: %+v", content.Prototype)
	}
	if len(content.Animations) != 1 {
		t.Fatalf("generation must return only the generated animation: result=%+v content=%+v", application, content)
	}
	generatedAnimation := content.Animations[0]
	if generatedAnimation.ID != 0 {
		t.Fatalf("generated animation candidate must not allocate an ID: %+v", generatedAnimation)
	}
	var rawResult map[string]json.RawMessage
	if err := json.Unmarshal(result, &rawResult); err != nil {
		t.Fatalf("decode raw animation result: %v", err)
	}
	if _, exists := rawResult["animation_id"]; exists {
		t.Fatalf("generated animation result must omit animation_id: %s", result)
	}
	var rawContent struct {
		Animations []map[string]json.RawMessage `json:"animations"`
	}
	if err := json.Unmarshal(rawResult["content"], &rawContent); err != nil {
		t.Fatalf("decode raw animation candidate content: %v", err)
	}
	if len(rawContent.Animations) != 1 {
		t.Fatalf("expected one raw animation candidate, got %+v", rawContent.Animations)
	}
	if _, exists := rawContent.Animations[0]["id"]; exists {
		t.Fatalf("generated animation candidate must omit id: %s", rawResult["content"])
	}
	wantGeneration := &assetdomain.AnimationGenerationConfig{
		Direction:   "back_right",
		Style:       "painted pixel art",
		Action:      "walking cycle",
		FrameCount:  8,
		Columns:     3,
		FrameWidth:  128,
		FrameHeight: 128,
		FPS:         12,
		Resolution:  "1080p",
		Duration:    8,
		AspectRatio: "1:1",
	}
	if !reflect.DeepEqual(generatedAnimation.Generation, wantGeneration) {
		t.Fatalf("unexpected generated animation config: got %+v want %+v", generatedAnimation.Generation, wantGeneration)
	}
	frames := generatedAnimation.Frames
	if frames[0].ID != 1 || frames[0].URL == nil ||
		*frames[0].URL != "uploads/generated-1.png" ||
		frames[0].Duration != 100 ||
		frames[1].ID != 2 || frames[1].URL == nil ||
		*frames[1].URL != "uploads/generated-2.png" ||
		frames[1].Duration != 100 {
		t.Fatalf("unexpected animation frames: %+v", frames)
	}
	if len(frames[0].Metadata) != 0 || len(frames[1].Metadata) != 0 {
		t.Fatalf("animation frames should not include generator metadata: %+v", frames)
	}
	if !reflect.DeepEqual(references.persisted, []string{
		"data:image/png;base64,first",
		"data:image/png;base64,second",
	}) {
		t.Fatalf("unexpected persisted animation frame inputs: %v", references.persisted)
	}
	if !reflect.DeepEqual(references.uploads, []referenceUpload{
		{key: "uploads/generated-1-unprocessed.png", reference: "data:image/png;base64,raw-first"},
		{key: "uploads/generated-2-unprocessed.png", reference: "data:image/png;base64,raw-second"},
	}) {
		t.Fatalf("unexpected persisted raw animation frames: %+v", references.uploads)
	}
	if application.AssetID != 7 || application.AnimationID != 0 || len(application.GeneratedResources) != 2 {
		t.Fatalf("unexpected animation application candidate: %+v", application)
	}
}

func TestExecutorPersistsEffectiveAnimationGenerationDefaults(t *testing.T) {
	events := []string{}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame", MIMEType: "image/png"}},
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: animationParentAsset(t)}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

	result, err := executor.Generate(
		context.Background(),
		generator.GenerateAnimation,
		json.RawMessage(`{"animation_name":"idle","asset_id":7,"direction":" FRONT "}`),
	)
	if err != nil {
		t.Fatalf("generate animation with defaults: %v", err)
	}
	want := &assetdomain.AnimationGenerationConfig{
		Direction:   "front",
		Style:       "finely drawn production-quality 2D game asset art",
		Action:      "idle",
		FrameCount:  16,
		Columns:     4,
		FrameWidth:  128,
		FrameHeight: 128,
		FPS:         10,
		Resolution:  "720p",
		Duration:    5,
		AspectRatio: "1:1",
	}
	_, content := decodeExecutionContent(t, result, assetdomain.AssetTypeCharacter)
	if len(content.Animations) != 1 || !reflect.DeepEqual(content.Animations[0].Generation, want) {
		t.Fatalf("unexpected generated animation defaults: %+v", content.Animations)
	}
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
			executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
				Animations: animations,
				References: &executorReferenceStoreStub{},
			})

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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: references,
	})

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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	parent := assetdomain.Asset{ID: 7, ProjectID: 11, Type: assetdomain.AssetTypeCharacter, Name: "hero", Dimensions: json.RawMessage(`{"width":128,"height":128}`), Content: encoded}
	animations := &animationGenerationServiceStub{events: &events, result: &generator.AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	executor = generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
		Dimensions:  json.RawMessage(`{"width":48,"height":64}`),
		Content:     encoded,
	}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}},
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	if !reflect.DeepEqual(events, []string{"get_asset", "generate_animation"}) {
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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: &executorReferenceStoreStub{},
	})

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
