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
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestExecutorEditsCharacterPrototypeAndCreatesNewVersionRecord(t *testing.T) {
	events := []string{}
	originalURLs := []string{
		"assets/hero/up.png",
		"assets/hero/right.png",
		"assets/hero/down.png",
		"assets/hero/left.png",
	}
	prototype := make(assetdomain.Prototype, len(originalURLs))
	for index := range originalURLs {
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &originalURLs[index]}
	}
	content, err := assetdomain.EncodeContent(assetdomain.AssetContent{
		DirectionCount: 4,
		Prototype:      &prototype,
		Animations: []assetdomain.Animation{{
			ID: 7, Name: "idle", Frames: []assetdomain.Frame{},
		}},
	})
	if err != nil {
		t.Fatalf("encode source content: %v", err)
	}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{
		events:        &events,
		recordVersion: 3,
		asset: assetdomain.Asset{
			ID:          7,
			Name:        "hero",
			ProjectID:   11,
			Type:        assetdomain.AssetTypeCharacter,
			Description: "a red knight carrying a steel spear",
			Perspective: assetdomain.PerspectiveTopDown,
			Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
			Content:     content,
			Version:     2,
		},
	}
	references := &executorReferenceStoreStub{events: &events}
	executor := generator.NewExecutor(
		images,
		&imageProcessorStub{events: &events},
		assets,
		references,
	)

	result, err := executor.Generate(
		context.Background(),
		generator.EditCharacterProtoType,
		json.RawMessage(`{
			"asset_id":7,
			"project_id":11,
			"edit_instructions":"change only the cape to blue"
		}`),
	)
	if err != nil {
		t.Fatalf("edit character prototype: %v", err)
	}

	if !reflect.DeepEqual(references.resolved, originalURLs) {
		t.Fatalf("unexpected resolved references: got %v want %v", references.resolved, originalURLs)
	}
	wantImageReferences := make([]string, len(originalURLs))
	for index, reference := range originalURLs {
		wantImageReferences[index] = "signed:" + reference
	}
	if images.request == nil || !reflect.DeepEqual(images.request.ReferenceImages, wantImageReferences) {
		t.Fatalf("unexpected edit image references: %+v", images.request)
	}
	for _, expected := range []string{
		"a red knight carrying a steel spear",
		"change only the cape to blue",
		"Reference images 1 through 4",
		"No separate user or project reference image is supplied",
	} {
		if !strings.Contains(images.request.Prompt, expected) {
			t.Fatalf("edit prompt missing %q: %s", expected, images.request.Prompt)
		}
	}
	if assets.createdRecord == nil || assets.createdRecord.AssetID != 7 {
		t.Fatalf("expected version record for asset 7: %+v", assets.createdRecord)
	}
	updated, err := (assetdomain.Asset{
		Type: assetdomain.AssetTypeCharacter, Content: assets.createdRecord.Content,
	}).DecodeContent()
	if err != nil {
		t.Fatalf("decode version content: %v", err)
	}
	if updated.DirectionCount != 4 || updated.Prototype == nil || len(*updated.Prototype) != 4 {
		t.Fatalf("unexpected edited prototype content: %+v", updated)
	}
	if len(updated.Animations) != 1 || updated.Animations[0].ID != 7 || updated.Animations[0].Name != "idle" {
		t.Fatalf("existing animations were not preserved: %+v", updated.Animations)
	}
	for index, resource := range *updated.Prototype {
		want := fmt.Sprintf("uploads/prototype-%d.png", index)
		if resource.URL == nil || *resource.URL != want {
			t.Fatalf("unexpected edited prototype resource %d: %+v", index, resource)
		}
	}
	if events[len(events)-1] != "create_record" {
		t.Fatalf("record must be created after generated images are persisted: %v", events)
	}
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, Version: 3})
}

func TestExecutorEditCharacterPrototypeRejectsInvalidStateAndDependencyFailures(t *testing.T) {
	wantLoadErr := errors.New("asset unavailable")
	wantResolveErr := errors.New("reference unavailable")
	wantRecordErr := errors.New("record unavailable")

	tests := []struct {
		name      string
		payload   json.RawMessage
		configure func(*generationAssetWriterStub, *executorReferenceStoreStub)
		wantErr   error
		wantText  string
		withStore bool
	}{
		{name: "malformed payload", payload: json.RawMessage(`{`), wantText: "decode edit_character_prototype execution payload"},
		{name: "asset load failure", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) { assets.detailErr = wantLoadErr }, wantErr: wantLoadErr},
		{name: "asset not found", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			assets.detailResult = &assetdomain.Asset{}
		}, wantText: "character asset 7 not found"},
		{name: "wrong asset type", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Type = assetdomain.AssetTypeObject
			assets.detailResult = &asset
		}, wantText: "unsupported for asset type"},
		{name: "invalid perspective", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Perspective = assetdomain.Perspective("sideways")
			assets.detailResult = &asset
		}, wantText: "invalid perspective"},
		{name: "malformed dimensions", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Dimensions = json.RawMessage(`{`)
			assets.detailResult = &asset
		}, wantText: "decode asset 7 dimensions"},
		{name: "nonpositive dimensions", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Dimensions = json.RawMessage(`{"width":0,"height":64}`)
			assets.detailResult = &asset
		}, wantText: "dimensions must be positive"},
		{name: "malformed content", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Content = json.RawMessage(`{`)
			assets.detailResult = &asset
		}, wantText: "decode character asset 7 content"},
		{name: "missing prototype", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Content = json.RawMessage(`{}`)
			assets.detailResult = &asset
		}, wantText: "prototype images are required"},
		{name: "missing prototype URL", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			asset := editableCharacterAsset()
			asset.Content = json.RawMessage(`{"prototype":[{"id":1}]}`)
			assets.detailResult = &asset
		}, wantText: "prototype image 1 URL is required"},
		{name: "reference resolution failure", configure: func(_ *generationAssetWriterStub, references *executorReferenceStoreStub) {
			references.resolveErr = wantResolveErr
		}, wantErr: wantResolveErr, withStore: true},
		{name: "record creation failure", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) {
			assets.recordErr = wantRecordErr
		}, wantErr: wantRecordErr},
		{name: "nil record", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) { assets.nilRecord = true }, wantText: "version: empty result"},
		{name: "zero record version", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub) { assets.emptyRecord = true }, wantText: "version: empty result"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			asset := editableCharacterAsset()
			assets := &generationAssetWriterStub{events: &events, detailResult: &asset}
			references := &executorReferenceStoreStub{events: &events}
			if test.configure != nil {
				test.configure(assets, references)
			}

			var executor generator.Executor
			if test.withStore {
				executor = generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets, references)
			} else {
				executor = generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)
			}
			payload := test.payload
			if payload == nil {
				payload = json.RawMessage(`{"asset_id":7,"edit_instructions":"make the cape blue"}`)
			}

			_, err := executor.Generate(context.Background(), generator.EditCharacterProtoType, payload)
			if err == nil {
				t.Fatal("expected edit failure")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected wrapped error %v, got %v", test.wantErr, err)
			}
			if test.wantText != "" && !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("expected error containing %q, got %v", test.wantText, err)
			}
		})
	}
}

func editableCharacterAsset() assetdomain.Asset {
	return assetdomain.Asset{
		ID:          7,
		Name:        "hero",
		ProjectID:   11,
		Type:        assetdomain.AssetTypeCharacter,
		Description: "a red knight carrying a steel spear",
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Content: json.RawMessage(`{
			"directionCount":4,
			"prototype":[
				{"id":1,"url":"assets/hero/up.png"},
				{"id":2,"url":"assets/hero/right.png"},
				{"id":3,"url":"assets/hero/down.png"},
				{"id":4,"url":"assets/hero/left.png"}
			]
		}`),
		Version: 2,
	}
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
	references := &executorReferenceStoreStub{events: &events}
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

func TestExecutorEditsObjectPrototypeAndCreatesNewVersionRecord(t *testing.T) {
	events := []string{}
	originalURLs := []string{
		"assets/chest/front.png",
		"assets/chest/front_right.png",
		"assets/chest/back_right.png",
		"assets/chest/back.png",
		"assets/chest/back_left.png",
		"assets/chest/front_left.png",
		"assets/chest/top.png",
		"assets/chest/bottom.png",
	}
	prototype := make(assetdomain.Prototype, len(originalURLs))
	for index := range originalURLs {
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &originalURLs[index]}
	}
	content := assetdomain.AssetContent{
		DirectionCount: 2,
		Prototype:      &prototype,
		Items: []assetdomain.TileSetItem{{
			Name:  "loot",
			Tiles: []assetdomain.Tile{{Position: assetdomain.TilePosition{X: 1, Y: 2}}},
		}},
		Metadata: map[string]any{"material": "wood"},
	}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode source content: %v", err)
	}

	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{
		events:        &events,
		recordVersion: 6,
		asset: assetdomain.Asset{
			ID:          8,
			Name:        "chest",
			ProjectID:   12,
			Type:        assetdomain.AssetTypeObject,
			Description: "an ornate wooden treasure chest",
			Perspective: assetdomain.PerspectiveIsometric,
			Dimensions:  json.RawMessage(`{"width":128,"height":128}`),
			Content:     encoded,
			Version:     5,
		},
	}
	references := &executorReferenceStoreStub{events: &events}
	executor := generator.NewExecutorForTest(images, &imageProcessorStub{events: &events}, assets, references)

	result, err := executor.Generate(
		context.Background(),
		generator.EditObjectProtoType,
		json.RawMessage(`{"asset_id":8,"project_id":12,"edit_instructions":"change only the lock to gold"}`),
	)
	if err != nil {
		t.Fatalf("edit object prototype: %v", err)
	}
	if assets.expectedVersion != 5 {
		t.Fatalf("expected current asset version to be passed separately, got %d", assets.expectedVersion)
	}
	if !reflect.DeepEqual(references.resolved, originalURLs) {
		t.Fatalf("unexpected resolved references: got %v want %v", references.resolved, originalURLs)
	}
	if images.request == nil || !strings.Contains(images.request.Prompt, "an ornate wooden treasure chest") ||
		!strings.Contains(images.request.Prompt, "change only the lock to gold") ||
		!strings.Contains(images.request.Prompt, "backend supplied exactly 8 current prototype direction image") {
		t.Fatalf("unexpected edit prompt: %+v", images.request)
	}
	if assets.createdRecord == nil || assets.createdRecord.AssetID != 8 {
		t.Fatalf("expected object version record: %+v", assets.createdRecord)
	}
	updated, err := (assetdomain.Asset{
		Type: assetdomain.AssetTypeObject, Content: assets.createdRecord.Content,
	}).DecodeContent()
	if err != nil {
		t.Fatalf("decode version content: %v", err)
	}
	if updated.DirectionCount != 8 || updated.Prototype == nil || len(*updated.Prototype) != 8 {
		t.Fatalf("unexpected edited object content: %+v", updated)
	}
	if len(updated.Items) != 1 || updated.Items[0].Name != "loot" || updated.Metadata["material"] != "wood" {
		t.Fatalf("existing object content was not preserved: %+v", updated)
	}
	for index, resource := range *updated.Prototype {
		want := fmt.Sprintf("uploads/prototype-%d.png", index)
		if resource.URL == nil || *resource.URL != want {
			t.Fatalf("unexpected edited prototype resource %d: %+v", index, resource)
		}
	}
	if events[len(events)-1] != "create_record" {
		t.Fatalf("record must be created after generated images are persisted: %v", events)
	}
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 8, Version: 6})
}

func TestExecutorEditObjectPrototypeRejectsInvalidStateAndDependencyFailures(t *testing.T) {
	wantLoadErr := errors.New("object unavailable")
	wantResolveErr := errors.New("reference unavailable")
	wantRecordErr := errors.New("record unavailable")
	wantImageErr := errors.New("image unavailable")

	tests := []struct {
		name      string
		payload   json.RawMessage
		configure func(*generationAssetWriterStub, *executorReferenceStoreStub, *imageGenerationServiceStub)
		wantErr   error
		wantText  string
		withStore bool
	}{
		{name: "malformed payload", payload: json.RawMessage(`{`), wantText: "decode edit_object_prototype execution payload"},
		{name: "asset load failure", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			assets.detailErr = wantLoadErr
		}, wantErr: wantLoadErr},
		{name: "asset not found", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			assets.detailResult = &assetdomain.Asset{}
		}, wantText: "object asset 8 not found"},
		{name: "wrong asset type", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Type = assetdomain.AssetTypeCharacter
			assets.detailResult = &asset
		}, wantText: "unsupported for asset type"},
		{name: "invalid perspective", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Perspective = assetdomain.Perspective("sideways")
			assets.detailResult = &asset
		}, wantText: "invalid perspective"},
		{name: "malformed dimensions", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Dimensions = json.RawMessage(`{`)
			assets.detailResult = &asset
		}, wantText: "decode asset 8 dimensions"},
		{name: "nonpositive dimensions", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Dimensions = json.RawMessage(`{"width":0,"height":128}`)
			assets.detailResult = &asset
		}, wantText: "dimensions must be positive"},
		{name: "malformed content", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Content = json.RawMessage(`{`)
			assets.detailResult = &asset
		}, wantText: "decode object asset 8 content"},
		{name: "missing prototype", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Content = json.RawMessage(`{}`)
			assets.detailResult = &asset
		}, wantText: "prototype images are required"},
		{name: "missing prototype URL", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			asset := editableObjectAsset()
			asset.Content = json.RawMessage(`{"prototype":[{"id":1}]}`)
			assets.detailResult = &asset
		}, wantText: "prototype image 1 URL is required"},
		{name: "reference resolution failure", configure: func(_ *generationAssetWriterStub, references *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			references.resolveErr = wantResolveErr
		}, wantErr: wantResolveErr, withStore: true},
		{name: "image generation failure", configure: func(_ *generationAssetWriterStub, _ *executorReferenceStoreStub, images *imageGenerationServiceStub) {
			images.err = wantImageErr
		}, wantErr: wantImageErr},
		{name: "record creation failure", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			assets.recordErr = wantRecordErr
		}, wantErr: wantRecordErr},
		{name: "nil record", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			assets.nilRecord = true
		}, wantText: "version: empty result"},
		{name: "zero record version", configure: func(assets *generationAssetWriterStub, _ *executorReferenceStoreStub, _ *imageGenerationServiceStub) {
			assets.emptyRecord = true
		}, wantText: "version: empty result"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			asset := editableObjectAsset()
			assets := &generationAssetWriterStub{events: &events, detailResult: &asset}
			references := &executorReferenceStoreStub{events: &events}
			if test.configure != nil {
				test.configure(assets, references, images)
			}

			var executor generator.Executor
			if test.withStore {
				executor = generator.NewExecutorForTest(images, &imageProcessorStub{events: &events}, assets, references)
			} else {
				executor = generator.NewExecutorForTest(images, &imageProcessorStub{events: &events}, assets, nil)
			}
			payload := test.payload
			if payload == nil {
				payload = json.RawMessage(`{"asset_id":8,"edit_instructions":"change the lock"}`)
			}
			_, err := executor.Generate(context.Background(), generator.EditObjectProtoType, payload)
			if err == nil {
				t.Fatal("expected edit failure")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected wrapped error %v, got %v", test.wantErr, err)
			}
			if test.wantText != "" && !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("expected error containing %q, got %v", test.wantText, err)
			}
		})
	}
}

func editableObjectAsset() assetdomain.Asset {
	originalURLs := []string{
		"assets/chest/front.png", "assets/chest/right.png", "assets/chest/back.png", "assets/chest/left.png",
	}
	prototype := make(assetdomain.Prototype, len(originalURLs))
	for index := range originalURLs {
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &originalURLs[index]}
	}
	content := assetdomain.AssetContent{
		DirectionCount: 4,
		Prototype:      &prototype,
		Metadata:       map[string]any{"material": "wood"},
	}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		panic(err)
	}
	return assetdomain.Asset{
		ID:          8,
		Type:        assetdomain.AssetTypeObject,
		Description: "a wooden treasure chest",
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":128,"height":128}`),
		Content:     encoded,
		Version:     4,
	}
}
