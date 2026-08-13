package generator_test

import (
	"context"
	"encoding/json"
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
	if animations.request == nil || animations.request.FrameCount != 11 || animations.request.Columns != 4 || !animations.request.ReferenceImageContext {
		t.Fatalf("unexpected edit animation request: %+v", animations.request)
	}
	if !reflect.DeepEqual(animations.request.TargetFrameIndices, []int{4, 6}) {
		t.Fatalf("unexpected target frame indices: %+v", animations.request.TargetFrameIndices)
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
	values := make(map[string]string, frameCount)
	for index := range frameCount {
		values[fmt.Sprintf("animations/original-%d-unprocessed.png", index+1)] = editFrameDataURL(t, uint8(index+1))
	}
	return &executorReferenceStoreStub{resolveValues: values}
}

func editFrameDataURL(t *testing.T, value uint8) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	draw.Draw(frame, image.Rect(2, 1, 6, 7), &image.Uniform{C: color.NRGBA{R: value, G: 40, B: 100, A: 255}}, image.Point{}, draw.Src)
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + encoded
}
