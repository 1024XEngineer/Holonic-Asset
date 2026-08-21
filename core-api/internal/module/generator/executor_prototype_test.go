package generator_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	generator "github.com/1024XEngineer/Holonic-Asset/internal/module/generator"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestExecutorEditsCharacterPrototypeAndReturnsApplicationCandidate(t *testing.T) {
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
		events: &events,
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
	executor := generator.NewExecutorWithDependencies(
		images,
		&imageProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{References: references},
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
	if images.request == nil || images.request.MaxAttempts != 3 || !reflect.DeepEqual(images.request.ReferenceImages, wantImageReferences) {
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
	application, updated := decodeExecutionContent(t, result, assetdomain.AssetTypeCharacter)
	if updated.DirectionCount != 4 || updated.Prototype == nil || len(*updated.Prototype) != 4 {
		t.Fatalf("unexpected edited prototype content: %+v", updated)
	}
	if len(updated.Animations) != 0 || len(updated.Items) != 0 || len(updated.Metadata) != 0 {
		t.Fatalf("prototype candidate included unrelated asset content: %+v", updated)
	}
	for index, resource := range *updated.Prototype {
		want := fmt.Sprintf("uploads/prototype-%d.png", index)
		if resource.URL == nil || *resource.URL != want {
			t.Fatalf("unexpected edited prototype resource %d: %+v", index, resource)
		}
	}
	if application.AssetID != 7 || application.Version != 2 || len(application.GeneratedResources) != 8 {
		t.Fatalf("unexpected application candidate: %+v", application)
	}
}

func TestExecutorEditCharacterPrototypeRejectsInvalidStateAndDependencyFailures(t *testing.T) {
	wantLoadErr := errors.New("asset unavailable")
	wantResolveErr := errors.New("reference unavailable")

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
				executor = generator.NewExecutorWithDependencies(
					images,
					&imageProcessorStub{events: &events},
					assets,
					generator.ExecutorDependencies{References: references},
				)
			} else {
				executor = generator.NewExecutorWithDependencies(
					images,
					&imageProcessorStub{events: &events},
					assets,
					generator.ExecutorDependencies{},
				)
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
	executor := generator.NewExecutorWithDependencies(images, processor, assets, generator.ExecutorDependencies{})
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
			"dimensions":{"width":64,"height":64},
		"perspective":"Top-Down",
		"creating_reference":"reference.png",
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
		"create_character_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || images.request.MaxAttempts != 3 || !strings.Contains(images.request.Prompt, "pixel knight") ||
		!strings.Contains(images.request.Prompt, "<direction_count>\n4\n</direction_count>") ||
		images.request.Size != "" ||
		!reflect.DeepEqual(images.request.ReferenceImages, []string{"reference.png"}) {
		t.Fatalf("unexpected image request: %+v", images.request)
	}
	if len(processor.splitRequests) != 1 {
		t.Fatalf("expected one prototype split request, got %d", len(processor.splitRequests))
	}
	splitRequest := processor.splitRequests[0]
	if splitRequest.Mode != imageprocessor.ImageSplitModeAnimation ||
		!splitRequest.ForceProportionalGrid ||
		splitRequest.Columns != 2 || splitRequest.Rows != 2 ||
		splitRequest.FrameWidth != 64 || splitRequest.FrameHeight != 64 ||
		splitRequest.Margin != imageprocessor.AnimationFrameMargin(64, 64) ||
		splitRequest.Anchor != imageprocessor.AnimationAnchorCenter ||
		!splitRequest.NormalizeContentScale || splitRequest.CropToContent {
		t.Fatalf("prototype directions were not normalized on a shared canvas: %+v", splitRequest)
	}
	if len(processor.resizeRequests) != 0 {
		t.Fatalf("normalized prototype PNGs must not be resampled again: %+v", processor.resizeRequests)
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
			executor := generator.NewExecutorWithDependencies(
				&imageGenerationServiceStub{events: &events, result: generatedImages()},
				&imageProcessorStub{events: &events},
				assets,
				generator.ExecutorDependencies{},
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
	executor := generator.NewExecutorWithDependencies(
		images,
		&imageProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{References: references},
	)
	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel knight",
			"dimensions":{"width":64,"height":64},
		"perspective":"Top-Down",
		"direction_count":"4",
		"project_reference":"projects/7/style.png",
		"creating_reference":"uploads/user-concept.png",
		"project_id":11
	}`)

	if _, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload); err != nil {
		t.Fatalf("generate prototype: %v", err)
	}
	if !reflect.DeepEqual(references.resolved, []string{"projects/7/style.png", "uploads/user-concept.png"}) {
		t.Fatalf("expected execution-time reference resolution, got %v", references.resolved)
	}
	if images.request == nil || !reflect.DeepEqual(images.request.ReferenceImages, []string{
		"signed:projects/7/style.png",
		"signed:uploads/user-concept.png",
	}) {
		t.Fatalf("unexpected resolved image reference order: %+v", images.request)
	}
	if len(references.uploads) != 8 {
		t.Fatalf("expected four unprocessed and four final uploads, got %d: %+v", len(references.uploads), references.uploads)
	}
	wantEvents := []string{"generate_image", "process_image", "split_image", "allocate_key"}
	for index := range 4 {
		wantEvents = append(wantEvents,
			fmt.Sprintf("persist:uploads/prototype-%d-unprocessed.png", index),
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

func TestExecutorNormalizesSmallResolvedReferencesBeforeGeneration(t *testing.T) {
	const downloaded = "small-pixel-reference"
	client := &http.Client{Transport: prototypeReferenceRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(strings.NewReader(downloaded)),
		}, nil
	})}

	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	processor := &imageProcessorStub{
		events: &events,
		normalizeResults: []*imageprocessor.NormalizeReferenceResult{
			{ImageBase64: "project-canonical", MIMEType: "image/png", Report: imageprocessor.ReferenceNormalizationReport{Scale: 1}},
			{ImageBase64: "user-upscaled", MIMEType: "image/png", Report: imageprocessor.ReferenceNormalizationReport{Scale: 16, Upscaled: true}},
		},
	}
	references := &executorReferenceStoreStub{resolveValues: map[string]string{
		"projects/7/style.png": "https://references.example/project",
		"uploads/user.png":     "https://references.example/user",
	}}
	executor := generator.NewExecutorWithPrototypeReferenceHTTPClientForTest(
		images,
		processor,
		assets,
		references,
		client,
	)
	payload := json.RawMessage(`{
		"asset_name":"sapling",
		"creative_brief":"a sapling",
		"dimensions":{"width":48,"height":48},
		"perspective":"Side-On",
		"project_reference":"projects/7/style.png",
		"creating_reference":"uploads/user.png"
	}`)

	if _, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload); err != nil {
		t.Fatalf("generate normalized-reference prototype: %v", err)
	}
	if len(processor.normalizeRequests) != 2 {
		t.Fatalf("normalization calls = %d, want 2", len(processor.normalizeRequests))
	}
	wantDownloaded := base64.StdEncoding.EncodeToString([]byte(downloaded))
	for index, request := range processor.normalizeRequests {
		if request.ImageBase64 != wantDownloaded {
			t.Fatalf("normalization request %d did not receive downloaded bytes", index)
		}
	}
	if images.request == nil || !reflect.DeepEqual(images.request.ReferenceImages, []string{
		"https://references.example/project",
		"data:image/png;base64,user-upscaled",
	}) {
		t.Fatalf("unexpected normalized reference order: %+v", images.request)
	}
}

func TestExecutorRejectsPrivatePrototypeReferenceBeforeDownload(t *testing.T) {
	downloadAttempted := false
	client := &http.Client{Transport: prototypeReferenceRoundTripFunc(func(*http.Request) (*http.Response, error) {
		downloadAttempted = true
		return nil, errors.New("unexpected download")
	})}
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	processor := &imageProcessorStub{events: &events}
	references := &executorReferenceStoreStub{resolveValues: map[string]string{
		"uploads/user.png": "http://169.254.169.254/latest/meta-data",
	}}
	executor := generator.NewExecutorWithPrototypeReferenceHTTPClientForTest(
		images,
		processor,
		&generationAssetWriterStub{events: &events},
		references,
		client,
	)
	payload := json.RawMessage(`{
		"asset_name":"sapling",
		"creative_brief":"a sapling",
		"dimensions":{"width":48,"height":48},
		"perspective":"Side-On",
		"creating_reference":"uploads/user.png"
	}`)

	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if err == nil || !strings.Contains(err.Error(), "not public") {
		t.Fatalf("generate error = %v, want non-public reference rejection", err)
	}
	if downloadAttempted {
		t.Fatal("private reference reached the HTTP transport")
	}
	if images.request != nil || len(processor.normalizeRequests) != 0 {
		t.Fatalf("private reference reached generation: request=%+v normalization=%d", images.request, len(processor.normalizeRequests))
	}
}

func TestExecutorRequiresReferenceStoreForPrototypeURL(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	processor := &imageProcessorStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{},
	)
	payload := json.RawMessage(`{
		"asset_name":"sapling",
		"creative_brief":"a sapling",
		"dimensions":{"width":48,"height":48},
		"perspective":"Side-On",
		"creating_reference":"https://attacker.example/reference.png"
	}`)

	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if err == nil || !strings.Contains(err.Error(), "object-storage reference store is required") {
		t.Fatalf("generate error = %v, want managed reference requirement", err)
	}
	if images.request != nil || len(processor.normalizeRequests) != 0 {
		t.Fatalf("unmanaged URL reached generation: request=%+v normalization=%d", images.request, len(processor.normalizeRequests))
	}
}

type prototypeReferenceRoundTripFunc func(*http.Request) (*http.Response, error)

func (function prototypeReferenceRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestExecutorGeneratesObjectPrototypeBeforeCreatingAsset(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		&imageProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{},
	)
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
		"create_object_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || images.request.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts 3, got %+v", images.request)
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
	executor := generator.NewExecutorWithDependencies(
		images,
		&imageProcessorStub{events: &events},
		assets,
		generator.ExecutorDependencies{},
	)
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

func TestExecutorEditsObjectPrototypeAndReturnsApplicationCandidate(t *testing.T) {
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
		events: &events,
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
	if !reflect.DeepEqual(references.resolved, originalURLs) {
		t.Fatalf("unexpected resolved references: got %v want %v", references.resolved, originalURLs)
	}
	if images.request == nil || !strings.Contains(images.request.Prompt, "an ornate wooden treasure chest") ||
		!strings.Contains(images.request.Prompt, "change only the lock to gold") ||
		!strings.Contains(images.request.Prompt, "backend supplied exactly 8 current prototype direction image") {
		t.Fatalf("unexpected edit prompt: %+v", images.request)
	}
	application, updated := decodeExecutionContent(t, result, assetdomain.AssetTypeObject)
	if updated.DirectionCount != 8 || updated.Prototype == nil || len(*updated.Prototype) != 8 {
		t.Fatalf("unexpected edited object content: %+v", updated)
	}
	if len(updated.Animations) != 0 || len(updated.Items) != 0 || len(updated.Metadata) != 0 {
		t.Fatalf("prototype candidate included unrelated asset content: %+v", updated)
	}
	for index, resource := range *updated.Prototype {
		want := fmt.Sprintf("uploads/prototype-%d.png", index)
		if resource.URL == nil || *resource.URL != want {
			t.Fatalf("unexpected edited prototype resource %d: %+v", index, resource)
		}
	}
	if application.AssetID != 8 || application.Version != 5 || len(application.GeneratedResources) != 16 {
		t.Fatalf("unexpected application candidate: %+v", application)
	}
}

func TestExecutorEditObjectPrototypeRejectsInvalidStateAndDependencyFailures(t *testing.T) {
	wantLoadErr := errors.New("object unavailable")
	wantResolveErr := errors.New("reference unavailable")
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
