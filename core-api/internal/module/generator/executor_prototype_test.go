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
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
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
		"resize_image",
		"resize_image",
		"resize_image",
		"resize_image",
		"create_character_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if images.request == nil || images.request.MaxAttempts != 3 || !strings.Contains(images.request.Prompt, "pixel knight") ||
		!strings.Contains(images.request.Prompt, "The subject's correct colours always take precedence") ||
		!strings.Contains(images.request.Prompt, "<direction_count>\n4\n</direction_count>") ||
		!strings.Contains(images.request.Prompt, "<asset_dimensions>\n{\"width\":64,\"height\":64}\n</asset_dimensions>") ||
		!strings.Contains(images.request.Prompt, "at most 40 x 40 drawable logical pixels") ||
		images.request.Size != "896x896" ||
		!reflect.DeepEqual(images.request.ReferenceImages, []string{"reference.png"}) {
		t.Fatalf("unexpected image request: %+v", images.request)
	}
	if len(images.requests) != 1 {
		t.Fatalf("image generation call count = %d, want one high-quality direction sheet", len(images.requests))
	}
	if request := images.requests[0]; request.MaxAttempts != 3 || request.Params["quality"] != "high" {
		t.Fatalf("image request did not ask for high quality with retries: %+v", request)
	}
	if len(processor.removeRequests) != 1 || processor.removeRequests[0].MatteColor != "auto" {
		t.Fatalf("prototype background removal did not auto-detect the matte: %+v", processor.removeRequests)
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
		!splitRequest.NormalizeContentScale || splitRequest.CropToContent ||
		!splitRequest.RejectGridBoundaryContent || splitRequest.GridBoundaryMargin != 14 {
		t.Fatalf("prototype directions were not normalized on a shared canvas: %+v", splitRequest)
	}
	if len(processor.resizeRequests) != 4 {
		t.Fatalf("pixel post-process request count = %d, want 4: %+v", len(processor.resizeRequests), processor.resizeRequests)
	}
	for index, request := range processor.resizeRequests {
		if request.Options.Width != 64 || request.Options.Height != 64 ||
			request.Options.Margin != 0 || request.Options.CropContent {
			t.Fatalf("prototype direction %d changed canonical frame geometry: %+v", index, request.Options)
		}
		if request.Options.Mode != imageprocessor.RasterModePixel || !request.Options.HardAlpha || request.Options.PaletteSize != 14 {
			t.Fatalf("prototype direction %d did not use character pixel output options: %+v", index, request.Options)
		}
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

func TestExecutorNormalizesSideOnPrototypeToOppositeDirections(t *testing.T) {
	events := []string{}
	assets := &generationAssetWriterStub{events: &events}
	processor := &imageProcessorStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events, result: generatedImages()},
		processor,
		assets,
		generator.ExecutorDependencies{},
	)

	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel basketball player",
		"dimensions":{"width":64,"height":64},
		"perspective":"Side-On",
		"project_id":17
	}`)
	if _, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload); err != nil {
		t.Fatalf("generate side-on character prototype: %v", err)
	}
	if !reflect.DeepEqual(events, []string{
		"generate_image",
		"process_image",
		"split_image",
		"flip_horizontal",
		"resize_image",
		"resize_image",
		"create_character_asset",
	}) {
		t.Fatalf("unexpected workflow order: %v", events)
	}
	if len(processor.resizeRequests) != 2 {
		t.Fatalf("Side-On pixel post-process request count = %d, want 2", len(processor.resizeRequests))
	}
	content, err := assets.characterAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode character content: %v", err)
	}
	if content.DirectionCount != 2 || content.Prototype == nil || len(*content.Prototype) != 2 {
		t.Fatalf("unexpected side-on content: %+v", content)
	}
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
	processor := &imageProcessorStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
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
	if images.request == nil || images.request.MaxAttempts != 3 || images.request.Size != "1536x768" {
		t.Fatalf("expected MaxAttempts 3, got %+v", images.request)
	}
	if len(processor.resizeRequests) != 8 {
		t.Fatalf("pixel post-process request count = %d, want 8", len(processor.resizeRequests))
	}
	for index, request := range processor.resizeRequests {
		if request.Options.Width != 128 || request.Options.Height != 128 ||
			request.Options.Margin != 0 || request.Options.CropContent ||
			request.Options.PaletteSize != 16 || !request.Options.NormalizeNearRound {
			t.Fatalf("object direction %d did not preserve canonical geometry with object pixel options: %+v", index, request.Options)
		}
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

func TestExecutorUsesTargetAspectRatioAndRetriesGridBoundaryCandidates(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	processor := &imageProcessorStub{
		events:      &events,
		splitErrors: []error{imageprocessor.ErrGridBoundaryContent, nil},
	}
	assets := &generationAssetWriterStub{events: &events}
	references := &executorReferenceStoreStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		assets,
		generator.ExecutorDependencies{References: references},
	)
	payload := json.RawMessage(`{
		"asset_name":"king_sofa",
		"creative_brief":"ornate black and white striped sofa",
		"dimensions":{"width":188,"height":128},
		"perspective":"Top-Down"
	}`)

	if _, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload); err != nil {
		t.Fatalf("generate rectangular prototype after boundary retry: %v", err)
	}
	if len(images.requests) != 2 {
		t.Fatalf("generation requests = %d, want 2", len(images.requests))
	}
	for index, request := range images.requests {
		if request.Size != "1504x1024" {
			t.Fatalf("generation request %d size = %q, want 1504x1024", index, request.Size)
		}
	}
	if len(processor.splitRequests) != 2 {
		t.Fatalf("split requests = %d, want 2", len(processor.splitRequests))
	}
	for index, request := range processor.splitRequests {
		if request.Mode != imageprocessor.ImageSplitModeAnimation ||
			request.Columns != 2 || request.Rows != 2 ||
			request.FrameWidth != 188 || request.FrameHeight != 128 ||
			request.Anchor != imageprocessor.AnimationAnchorCenter ||
			!request.NormalizeContentScale || !request.RejectGridBoundaryContent ||
			request.GridBoundaryMargin != 16 {
			t.Fatalf("unexpected split request %d: %+v", index, request)
		}
	}
	if len(processor.resizeRequests) != 4 {
		t.Fatalf("pixel post-process request count = %d, want 4", len(processor.resizeRequests))
	}
	if len(references.uploads) != 8 {
		t.Fatalf("uploads = %d, want only 8 from the successful candidate", len(references.uploads))
	}
	if assets.objectAsset == nil {
		t.Fatal("expected object asset after successful boundary retry")
	}
}

func TestExecutorRejectsPrototypeAfterAllGridBoundaryCandidatesFail(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	processor := &imageProcessorStub{
		events: &events,
		splitErrors: []error{
			imageprocessor.ErrGridBoundaryContent,
			imageprocessor.ErrGridBoundaryContent,
			imageprocessor.ErrGridBoundaryContent,
		},
	}
	assets := &generationAssetWriterStub{events: &events}
	references := &executorReferenceStoreStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		assets,
		generator.ExecutorDependencies{References: references},
	)
	payload := json.RawMessage(`{
		"asset_name":"king_sofa",
		"creative_brief":"ornate black and white striped sofa",
		"dimensions":{"width":188,"height":128},
		"perspective":"Top-Down"
	}`)

	_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
	if !errors.Is(err, imageprocessor.ErrGridBoundaryContent) {
		t.Fatalf("error = %v, want ErrGridBoundaryContent", err)
	}
	if len(images.requests) != 3 || len(processor.splitRequests) != 3 {
		t.Fatalf("generation/split attempts = %d/%d, want 3/3", len(images.requests), len(processor.splitRequests))
	}
	if assets.objectAsset != nil {
		t.Fatalf("asset must not be created from invalid candidates: %+v", assets.objectAsset)
	}
	if len(references.uploads) != 0 {
		t.Fatalf("invalid candidates must not be persisted: %+v", references.uploads)
	}
}

func TestExecutorRejectsInvalidGeneratedPrototypeCandidates(t *testing.T) {
	wantSplitErr := errors.New("split failed")
	wantPersistErr := errors.New("persist failed")
	tests := []struct {
		name      string
		result    *imageclient.GenerateResult
		configure func(*imageProcessorStub, *executorReferenceStoreStub)
		wantErr   error
		wantText  string
		withStore bool
	}{
		{name: "nil generation result", wantText: "image result is required"},
		{name: "empty generation result", result: &imageclient.GenerateResult{}, wantText: "image result is required"},
		{
			name: "multiple direction sheets",
			result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{
				{Base64: "first", MediaType: "image/png"},
				{Base64: "second", MediaType: "image/png"},
			}},
			wantText: "expected one direction sheet, got 2",
		},
		{
			name:     "empty generated image",
			result:   &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{MediaType: "image/png"}}},
			wantText: "image result is required",
		},
		{
			name:   "non-boundary split failure",
			result: generatedImages(),
			configure: func(processor *imageProcessorStub, _ *executorReferenceStoreStub) {
				processor.splitErrors = []error{wantSplitErr}
			},
			wantErr: wantSplitErr,
		},
		{
			name:   "wrong region count",
			result: generatedImages(),
			configure: func(processor *imageProcessorStub, _ *executorReferenceStoreStub) {
				processor.splitResults = []*imageprocessor.SplitImageResult{{
					Regions: []imageprocessor.ImageRegion{{ImageBase64: "only-one"}},
				}}
			},
			wantText: "got 1 regions, want 4",
		},
		{
			name:   "empty direction image",
			result: generatedImages(),
			configure: func(processor *imageProcessorStub, _ *executorReferenceStoreStub) {
				processor.splitResults = []*imageprocessor.SplitImageResult{{Regions: []imageprocessor.ImageRegion{
					{ImageBase64: "first"},
					{ImageBase64: "second"},
					{},
					{ImageBase64: "fourth"},
				}}}
			},
			wantText:  "direction 2 is empty",
			withStore: true,
		},
		{
			name:   "unprocessed image persistence failure",
			result: generatedImages(),
			configure: func(_ *imageProcessorStub, references *executorReferenceStoreStub) {
				references.persistErr = wantPersistErr
			},
			wantErr:   wantPersistErr,
			withStore: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: test.result}
			processor := &imageProcessorStub{events: &events}
			references := &executorReferenceStoreStub{events: &events}
			if test.configure != nil {
				test.configure(processor, references)
			}
			dependencies := generator.ExecutorDependencies{}
			if test.withStore {
				dependencies.References = references
			}
			executor := generator.NewExecutorWithDependencies(
				images,
				processor,
				&generationAssetWriterStub{events: &events},
				dependencies,
			)
			payload := json.RawMessage(`{
				"asset_name":"king_sofa",
				"creative_brief":"ornate sofa",
				"dimensions":{"width":128,"height":128},
				"perspective":"Top-Down"
			}`)

			_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
			if err == nil {
				t.Fatal("expected prototype generation failure")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want wrapped %v", err, test.wantErr)
			}
			if test.wantText != "" && !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error = %v, want text %q", err, test.wantText)
			}
		})
	}
}

func TestExecutorRejectsUnsafePrototypeSheetDimensions(t *testing.T) {
	tests := []struct {
		name       string
		dimensions string
		wantText   string
	}{
		{name: "zero width", dimensions: `{"width":0,"height":64}`, wantText: "dimensions must be positive"},
		{
			name:       "sheet dimension overflow",
			dimensions: `{"width":18446744073709551615,"height":18446744073709551615}`,
			wantText:   "overflow sheet dimensions",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			executor := generator.NewExecutorWithDependencies(
				images,
				&imageProcessorStub{events: &events},
				&generationAssetWriterStub{events: &events},
				generator.ExecutorDependencies{},
			)
			payload := json.RawMessage(`{
				"asset_name":"unsafe_sofa",
				"creative_brief":"unsafe dimensions",
				"dimensions":` + test.dimensions + `,
				"perspective":"Top-Down"
			}`)

			_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error = %v, want text %q", err, test.wantText)
			}
			if len(images.requests) != 0 {
				t.Fatalf("image requests = %d, want none", len(images.requests))
			}
		})
	}
}

func TestExecutorDownscalesOversizedTargetSheetWithinProviderLimits(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	processor := &imageProcessorStub{events: &events}
	assets := &generationAssetWriterStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		assets,
		generator.ExecutorDependencies{},
	)
	payload := json.RawMessage(`{
		"asset_name":"large_sofa",
		"creative_brief":"large sofa",
		"dimensions":{"width":2048,"height":1024},
		"perspective":"Top-Down"
	}`)

	if _, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload); err != nil {
		t.Fatalf("generate oversized-target prototype: %v", err)
	}
	if images.request == nil || images.request.Size != "3840x1920" {
		t.Fatalf("provider size = %+v, want 3840x1920", images.request)
	}
	if len(processor.splitRequests) != 1 || processor.splitRequests[0].GridBoundaryMargin != 30 {
		t.Fatalf("unexpected oversized-target split request: %+v", processor.splitRequests)
	}
}

func TestExecutorRejectsPrototypeSheetAspectBeyondProviderLimit(t *testing.T) {
	for _, dimensions := range []string{
		`{"width":1024,"height":64}`,
		`{"width":64,"height":1024}`,
	} {
		t.Run(dimensions, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			executor := generator.NewExecutorWithDependencies(
				images,
				&imageProcessorStub{events: &events},
				&generationAssetWriterStub{events: &events},
				generator.ExecutorDependencies{},
			)
			payload := json.RawMessage(`{
				"asset_name":"extreme_sofa",
				"creative_brief":"extreme sofa",
				"dimensions":` + dimensions + `,
				"perspective":"Top-Down"
			}`)

			_, err := executor.Generate(context.Background(), generator.GenerateObjectProtoType, payload)
			if err == nil || !strings.Contains(err.Error(), "exceeding 3:1") {
				t.Fatalf("error = %v, want provider aspect-ratio rejection", err)
			}
			if len(images.requests) != 0 {
				t.Fatalf("image requests = %d, want none", len(images.requests))
			}
		})
	}
}

func TestExecutorUsesGeneratedPrototypeSheetForFinalProcessing(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{
		events: &events,
		result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: "generated-sheet", MediaType: "image/webp"}}},
	}
	processor := &imageProcessorStub{events: &events}
	executor := generator.NewExecutorWithDependencies(
		images,
		processor,
		&generationAssetWriterStub{events: &events},
		generator.ExecutorDependencies{},
	)

	_, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, json.RawMessage(`{
		"asset_name":"player",
		"creative_brief":"a readable fantasy athlete",
		"dimensions":{"width":64,"height":64},
		"perspective":"Side-On"
	}`))
	if err != nil {
		t.Fatalf("generate prototype: %v", err)
	}
	if len(images.requests) != 1 {
		t.Fatalf("prototype used %d image generation calls, want one", len(images.requests))
	}
	if len(processor.removeRequests) != 1 || processor.removeRequests[0].ImageBase64 != "generated-sheet" {
		t.Fatalf("final processing did not use generated sheet: %+v", processor.removeRequests)
	}
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

func TestExecutorRejectsSideOnPrototypeWhenDirectionNormalizationFails(t *testing.T) {
	events := []string{}
	wantErr := errors.New("horizontal flip unavailable")
	assets := &generationAssetWriterStub{events: &events}
	processor := &imageProcessorStub{events: &events, flipErr: wantErr}
	executor := generator.NewExecutorWithDependencies(
		&imageGenerationServiceStub{events: &events, result: generatedImages()},
		processor,
		assets,
		generator.ExecutorDependencies{},
	)

	payload := json.RawMessage(`{
		"asset_name":"hero",
		"creative_brief":"pixel basketball player",
		"dimensions":{"width":64,"height":64},
		"perspective":"Side-On",
		"project_id":17
	}`)
	if _, err := executor.Generate(context.Background(), generator.GenerateCharacterProtoType, payload); err == nil {
		t.Fatal("expected Side-On normalization failure")
	} else if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped error %v", err, wantErr)
	}
	if assets.characterAsset != nil {
		t.Fatal("invalid Side-On prototype was persisted")
	}
	if reflect.DeepEqual(events, []string{"generate_image", "process_image", "split_image", "flip_horizontal"}) == false {
		t.Fatalf("unexpected workflow after normalization failure: %v", events)
	}
}
