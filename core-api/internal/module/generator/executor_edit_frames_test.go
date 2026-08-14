package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestExecutorEditFramesReplacesOnlySelectedFramesAndPersistsRawFrames(t *testing.T) {
	parent := editFramesAsset(t, 12)
	var metadata = json.RawMessage(`{"anchor":"feet"}`)
	var content assetdomain.AssetContent
	if err := json.Unmarshal(parent.Content, &content); err != nil {
		t.Fatal(err)
	}
	content.Animations[0].Frames[4].Metadata = metadata
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	parent.Content = encoded
	updated := parent
	updated.Version = 9
	events := []string{}
	resultFrames := make([]imageprocessor.ImageRegion, 11)
	rawFrames := make([]imageprocessor.ImageRegion, 11)
	for index := range resultFrames {
		resultFrames[index] = imageprocessor.ImageRegion{Index: index, ImageBase64: fmt.Sprintf("edited-%d", index), MIMEType: "image/png"}
		rawFrames[index] = imageprocessor.ImageRegion{Index: index, ImageBase64: fmt.Sprintf("raw-%d", index), MIMEType: "image/png"}
	}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{Frames: resultFrames, RawFrames: rawFrames, FrameDurationMS: 111},
	}
	assets := &generationAssetWriterStub{
		events:       &events,
		parentAsset:  parent,
		detailResult: &updated,
	}
	references := editFrameReferenceStore(t, 12)
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations,
		References: references,
	})

	result, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(`{
		"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[5,7],"prompt":"make the sword glow"
	}`))
	if err != nil {
		t.Fatalf("edit frames: %v", err)
	}
	if len(animations.requests) != 1 {
		t.Fatalf("unexpected edit generation call count: %d", len(animations.requests))
	}
	request := animations.requests[0]
	if request.FrameCount != 11 || request.Columns != 4 || !request.ReferenceImageContext ||
		request.ReferenceImageContextSheet || request.ReferenceImagePrepared {
		t.Fatalf("unexpected single-reference edit request: %+v", request)
	}
	if request.ReferenceImage != "animations/original-5-unprocessed.png" {
		t.Fatalf("edit request did not use the raw single-frame reference: %q", request.ReferenceImage)
	}
	if animations.request.Style != "pixel art" ||
		animations.request.FrameWidth != 64 || animations.request.FrameHeight != 64 ||
		animations.request.FPS != 10 || animations.request.Resolution != "720p" ||
		animations.request.Duration != 5 || animations.request.AspectRatio != "1:1" {
		t.Fatalf("edit frame request did not inherit animation generation config: %+v", animations.request)
	}
	if !reflect.DeepEqual(request.TargetFrameIndices, []int{4, 6}) {
		t.Fatalf("unexpected target frame indices: %+v", request.TargetFrameIndices)
	}
	if animations.request.Action != "make the sword glow" {
		t.Fatalf("unexpected edit prompt: %+v", animations.request)
	}
	if assets.updateCalls != 1 || assets.updatedAnimationID != 42 || len(assets.frames) != 12 {
		t.Fatalf("unexpected updated animation: calls=%d id=%d frames=%d", assets.updateCalls, assets.updatedAnimationID, len(assets.frames))
	}
	for index, frame := range assets.frames {
		if frame.URL == nil {
			t.Fatalf("frame %d has no URL", index+1)
		}
		if index == 4 || index == 6 {
			if *frame.URL != fmt.Sprintf("uploads/generated-%d.png", map[int]int{4: 1, 6: 2}[index]) || frame.Duration != 111 || frame.ID != uint(index+1) {
				t.Fatalf("selected frame %d was not replaced: %+v", index+1, frame)
			}
			if index == 4 && !reflect.DeepEqual(frame.Metadata, metadata) {
				t.Fatalf("selected frame metadata was not preserved: %s", frame.Metadata)
			}
		} else if frame.Duration != uint(index+1)*10 || frame.ID != uint(index+1) {
			t.Fatalf("unselected frame %d changed: %+v", index+1, frame)
		}
	}
	if !reflect.DeepEqual(references.persisted, []string{
		"data:image/png;base64,edited-4", "data:image/png;base64,edited-6",
	}) {
		t.Fatalf("unexpected processed persistence: %v", references.persisted)
	}
	if len(references.uploads) != 2 || references.uploads[0].key != "uploads/generated-1-unprocessed.png" || references.uploads[1].key != "uploads/generated-2-unprocessed.png" {
		t.Fatalf("unexpected raw persistence: %+v", references.uploads)
	}
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, AnimationID: 42, Version: 9})
}

func TestExecutorEditFramesRejectsMissingRawOutput(t *testing.T) {
	parent := editFramesAsset(t, 1)
	events := []string{}
	animations := &animationGenerationServiceStub{
		events: &events,
		result: &generator.AnimationGenerationResult{
			Frames: []imageprocessor.ImageRegion{{ImageBase64: "edited", MIMEType: "image/png"}},
		},
	}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations, References: editFrameReferenceStore(t, 1),
	})
	_, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(
		`{"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[1],"prompt":"change pose"}`,
	))
	if err == nil || !strings.Contains(err.Error(), "edited raw frame result contains 0 frames") {
		t.Fatalf("expected missing raw frame error, got %v", err)
	}
	if assets.updateCalls != 0 {
		t.Fatalf("asset updated without raw frames: %d", assets.updateCalls)
	}
}

func TestExecutorEditFramesSupportsSingleFrameAndValidatesSelection(t *testing.T) {
	parent := editFramesAsset(t, 3)
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "missing animation", payload: `{"asset_id":7,"project_id":11,"frame_ids":[1],"prompt":"x"}`, want: "animation is required"},
		{name: "missing frames", payload: `{"asset_id":7,"project_id":11,"animation_id":42,"prompt":"x"}`, want: "frame ids are required"},
		{name: "duplicate frame", payload: `{"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[1,1],"prompt":"x"}`, want: "duplicated"},
		{name: "unknown frame", payload: `{"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[4],"prompt":"x"}`, want: "not found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
			animations := &animationGenerationServiceStub{events: &events}
			executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
				Animations: animations, References: &executorReferenceStoreStub{},
			})
			_, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(test.payload))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if animations.request != nil || assets.updateCalls != 0 {
				t.Fatalf("invalid selection started generation or update: request=%+v updates=%d", animations.request, assets.updateCalls)
			}
		})
	}
}

func editFramesAsset(t *testing.T, frameCount int) assetdomain.Asset {
	t.Helper()
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.Animations = []assetdomain.Animation{{ID: 42, Name: "walk", Frames: make([]assetdomain.Frame, frameCount), Generation: &assetdomain.AnimationGenerationConfig{
		Direction: "front", Style: "pixel art", FrameCount: frameCount, Columns: 4, FrameWidth: 64, FrameHeight: 64, FPS: 10, Resolution: "720p", Duration: 5, AspectRatio: "1:1",
	}}}
	for index := range content.Animations[0].Frames {
		value := fmt.Sprintf("animations/original-%d.png", index+1)
		content.Animations[0].Frames[index] = assetdomain.Frame{ID: uint(index + 1), URL: &value, Duration: uint(index+1) * 10}
	}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	return assetdomain.Asset{ID: 7, ProjectID: 11, Type: assetdomain.AssetTypeCharacter, Name: "hero", Description: "hero", Version: 3, Content: encoded}
}

func editFrameReferenceStore(t *testing.T, frameCount int) *executorReferenceStoreStub {
	t.Helper()
	values := make(map[string]string, frameCount*2)
	for index := range frameCount {
		// The processed frame is the canonical 8x8 asset. The raw companion is
		// deliberately larger to verify edit requests select the unprocessed
		// source while the generation service performs canonical normalization.
		values[fmt.Sprintf("animations/original-%d.png", index+1)] = editFrameDataURL(t, uint8(index+1), 8)
		values[fmt.Sprintf("animations/original-%d-unprocessed.png", index+1)] = editFrameDataURL(t, uint8(index+1), 32)
	}
	return &executorReferenceStoreStub{resolveValues: values}
}

func editFrameDataURL(t *testing.T, value uint8, size int) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(frame, image.Rect(size/4, size/8, size*3/4, size*7/8), &image.Uniform{C: color.NRGBA{R: value, G: 40, B: 100, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + encoded
}

func TestExecutorEditFramesRequiresGenerationConfiguration(t *testing.T) {
	parent := editFramesAsset(t, 1)
	var content assetdomain.AssetContent
	if err := json.Unmarshal(parent.Content, &content); err != nil {
		t.Fatalf("decode parent asset: %v", err)
	}
	content.Animations[0].Generation = nil
	parent.Content, _ = assetdomain.EncodeContent(content)
	events := []string{}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{events: &events}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{
		Animations: animations, References: editFrameReferenceStore(t, 1),
	})

	_, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(`{
		"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[1],"prompt":"change pose"
	}`))
	if err == nil || !strings.Contains(err.Error(), "has no generation configuration for frame editing") {
		t.Fatalf("expected missing generation configuration error, got %v", err)
	}
	if animations.request != nil || assets.updateCalls != 0 {
		t.Fatalf("missing generation configuration started edit: request=%+v updates=%d", animations.request, assets.updateCalls)
	}
}

func TestExecutorEditFramesValidatesContextAndAssetErrors(t *testing.T) {
	parent := editFramesAsset(t, 40)
	tests := []struct {
		name     string
		payload  generator.EditFramesPayload
		asset    assetdomain.Asset
		assetErr error
		want     string
	}{
		{name: "asset required", payload: generator.EditFramesPayload{ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, want: "asset is required"},
		{name: "project required", payload: generator.EditFramesPayload{AssetID: 7, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, want: "project is required"},
		{name: "prompt required", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}}, want: "prompt is required"},
		{name: "asset lookup", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, assetErr: errors.New("lookup failed"), want: "get edit frames asset 7"},
		{name: "asset missing", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, asset: assetdomain.Asset{}, want: "not found"},
		{name: "project mismatch", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 99, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, asset: parent, want: "belongs to project"},
		{name: "bad content", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, asset: assetdomain.Asset{ID: 7, ProjectID: 11, Content: json.RawMessage(`{`)}, want: "decode edit frames asset 7 content"},
		{name: "animation missing", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 99, FrameIDs: []uint{1}, Prompt: "x"}, asset: parent, want: "animation 99 not found"},
		{name: "animation has no frames", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, asset: emptyAnimationAsset(t), want: "has no frames"},
		{name: "zero frame", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{0}, Prompt: "x"}, asset: parent, want: "frame id must be positive"},
		{name: "context too large", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1, 40}, Prompt: "x"}, asset: parent, want: "context contains 40 frames"},
		{name: "missing URL", payload: generator.EditFramesPayload{AssetID: 7, ProjectID: 11, AnimationID: 42, FrameIDs: []uint{1}, Prompt: "x"}, asset: assetWithMissingFrameURL(t), want: "has no image URL"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			assetStub := &generationAssetWriterStub{events: &events, parentAsset: parent, getDetailErr: test.assetErr}
			if test.name == "asset missing" || test.asset.ID != 0 || test.asset.Content != nil {
				assetStub.parentAsset = test.asset
			}
			executor := generator.NewExecutorWithDependencies(nil, nil, assetStub, generator.ExecutorDependencies{
				Animations: &animationGenerationServiceStub{events: &events}, References: editFrameReferenceStore(t, 40),
			})
			_, err := executor.Generate(context.Background(), generator.EditFrames, mustJSON(t, test.payload))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if assetStub.updateCalls != 0 {
				t.Fatalf("invalid edit updated animation: %d", assetStub.updateCalls)
			}
		})
	}
}

func TestExecutorEditFramesHandlesGenerationAndPersistenceErrors(t *testing.T) {
	parent := editFramesAsset(t, 3)
	tests := []struct {
		name       string
		result     *generator.AnimationGenerationResult
		generation error
		store      *executorReferenceStoreStub
		updateErr  error
		want       string
	}{
		{name: "generation error", generation: errors.New("provider failed"), want: "generate edited frames"},
		{name: "nil result", result: nil, want: "edited frame result contains 0 frames"},
		{name: "wrong processed count", result: &generator.AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "edited"}}, RawFrames: []imageprocessor.ImageRegion{{ImageBase64: "raw"}}}, want: "expected 3"},
		{name: "wrong raw count", result: &generator.AnimationGenerationResult{Frames: makeImageRegions(3, "edited"), RawFrames: makeImageRegions(2, "raw")}, want: "edited raw frame result contains 2"},
		{name: "processed persist error", result: &generator.AnimationGenerationResult{Frames: makeImageRegions(3, "edited"), RawFrames: makeImageRegions(3, "raw")}, store: &executorReferenceStoreStub{persistErr: errors.New("processed upload failed")}, want: "persist edited animation frame"},
		{name: "invalid processed key", result: &generator.AnimationGenerationResult{Frames: makeImageRegions(3, "edited"), RawFrames: makeImageRegions(3, "raw")}, store: &executorReferenceStoreStub{persistValue: "data:image/png;base64,bad"}, want: "non-object-key"},
		{name: "update error", result: &generator.AnimationGenerationResult{Frames: makeImageRegions(3, "edited"), RawFrames: makeImageRegions(3, "raw"), FrameDurationMS: 80}, store: editFrameReferenceStore(t, 3), updateErr: errors.New("update failed")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			animations := &animationGenerationServiceStub{events: &events, result: test.result, err: test.generation}
			assets := &generationAssetWriterStub{events: &events, parentAsset: parent, updateAnimationErr: test.updateErr}
			store := test.store
			if store == nil {
				store = editFrameReferenceStore(t, 3)
			} else {
				store.resolveValues = editFrameReferenceStore(t, 3).resolveValues
			}
			executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{Animations: animations, References: store})
			_, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(`{"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[1],"prompt":"change pose"}`))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if test.name != "update error" && assets.updateCalls != 0 {
				t.Fatalf("failed edit updated animation: %d", assets.updateCalls)
			}
		})
	}
}

func TestExecutorEditFramesPreservesOriginalDurationWhenGeneratedDurationIsZero(t *testing.T) {
	parent := editFramesAsset(t, 1)
	events := []string{}
	assets := &generationAssetWriterStub{events: &events, parentAsset: parent}
	animations := &animationGenerationServiceStub{events: &events, result: &generator.AnimationGenerationResult{
		Frames: []imageprocessor.ImageRegion{{ImageBase64: "edited"}}, RawFrames: []imageprocessor.ImageRegion{{ImageBase64: "raw"}},
	}}
	executor := generator.NewExecutorWithDependencies(nil, nil, assets, generator.ExecutorDependencies{Animations: animations, References: editFrameReferenceStore(t, 1)})
	if _, err := executor.Generate(context.Background(), generator.EditFrames, json.RawMessage(`{"asset_id":7,"project_id":11,"animation_id":42,"frame_ids":[1],"prompt":"change pose"}`)); err != nil {
		t.Fatalf("edit frame: %v", err)
	}
	if assets.frames[0].Duration != 10 {
		t.Fatalf("generated zero duration did not preserve original duration: %+v", assets.frames[0])
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func makeImageRegions(count int, prefix string) []imageprocessor.ImageRegion {
	regions := make([]imageprocessor.ImageRegion, count)
	for index := range regions {
		regions[index] = imageprocessor.ImageRegion{Index: index, ImageBase64: fmt.Sprintf("%s-%d", prefix, index), MIMEType: "image/png"}
	}
	return regions
}

func emptyAnimationAsset(t *testing.T) assetdomain.Asset {
	t.Helper()
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.Animations = []assetdomain.Animation{{ID: 42}}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	return assetdomain.Asset{ID: 7, ProjectID: 11, Type: assetdomain.AssetTypeCharacter, Content: encoded}
}

func assetWithMissingFrameURL(t *testing.T) assetdomain.Asset {
	t.Helper()
	asset := editFramesAsset(t, 1)
	var content assetdomain.AssetContent
	if err := json.Unmarshal(asset.Content, &content); err != nil {
		t.Fatal(err)
	}
	content.Animations[0].Frames[0].URL = nil
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	asset.Content = encoded
	return asset
}
