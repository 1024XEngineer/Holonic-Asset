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
	prototypeAssetID uint
	prototypeImages  []assetdomain.ImageResource
	animationAssetID uint
	animationName    string
	animationID      uint
	frames           []assetdomain.Frame
	createdRecord    *assetdomain.AssetRecord
	recordVersion    uint
	detailErr        error
	recordErr        error
	detailResult     *assetdomain.Asset
	nilRecord        bool
	emptyRecord      bool
	err              error
	asset            assetdomain.Asset
}

func (s *generationAssetWriterStub) GetDetail(_ context.Context, assetID uint) (assetdomain.Asset, error) {
	if s.detailErr != nil {
		return assetdomain.Asset{}, s.detailErr
	}
	if s.err != nil {
		return assetdomain.Asset{}, s.err
	}
	if s.detailResult != nil {
		return *s.detailResult, nil
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

func (s *generationAssetWriterStub) CreateRecord(
	_ context.Context,
	record *assetdomain.AssetRecord,
) (*assetdomain.AssetRecord, error) {
	*s.events = append(*s.events, "create_record")
	if record != nil {
		copy := *record
		copy.Content = append(json.RawMessage(nil), record.Content...)
		s.createdRecord = &copy
	}
	if s.recordErr != nil {
		return nil, s.recordErr
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.nilRecord {
		return nil, nil //nolint:nilnil // Exercise the executor's defensive empty-result check.
	}
	version := s.recordVersion
	if version == 0 && !s.emptyRecord {
		version = 2
	}
	return &assetdomain.AssetRecord{AssetID: record.AssetID, Version: version, Content: record.Content}, nil
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

func TestExecutorEditsObjectPrototypeAndCreatesNewVersionRecord(t *testing.T) {
	events := []string{}
	images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
	asset := editableObjectAsset()
	assets := &generationAssetWriterStub{events: &events, detailResult: &asset}
	references := &referenceStoreStub{events: &events}
	executor := generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets, references)

	result, err := executor.Generate(context.Background(), generator.EditObjectProtoType, json.RawMessage(`{
		"asset_id":7,
		"project_id":11,
		"edit_instructions":"change only the lock to gold"
	}`))
	if err != nil {
		t.Fatalf("edit object prototype: %v", err)
	}

	wantResolved := []string{
		"assets/chest/up.png",
		"assets/chest/right.png",
		"assets/chest/down.png",
		"assets/chest/left.png",
	}
	if !reflect.DeepEqual(references.resolved, wantResolved) {
		t.Fatalf("unexpected resolved references: got %v want %v", references.resolved, wantResolved)
	}
	wantImageReferences := make([]string, len(wantResolved))
	for index, reference := range wantResolved {
		wantImageReferences[index] = "signed:" + reference
	}
	if images.request == nil || !reflect.DeepEqual(images.request.ReferenceImages, wantImageReferences) {
		t.Fatalf("unexpected edit image references: %+v", images.request)
	}
	for _, expected := range []string{
		"a wooden chest with an iron lock",
		"change only the lock to gold",
		"backend supplied exactly 4 current prototype direction image(s)",
		"No user or project reference image is supplied",
	} {
		if !strings.Contains(images.request.Prompt, expected) {
			t.Fatalf("edit prompt missing %q: %s", expected, images.request.Prompt)
		}
	}
	if assets.createdRecord == nil || assets.createdRecord.AssetID != 7 {
		t.Fatalf("expected version record for asset 7: %+v", assets.createdRecord)
	}
	updated, err := (assetdomain.Asset{
		Type: assetdomain.AssetTypeObject, Content: assets.createdRecord.Content,
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
	if len(events) == 0 || events[len(events)-1] != "create_record" {
		t.Fatalf("record must be created after generated images are persisted: %v", events)
	}
	assertExecutionResult(t, result, generator.ExecutionResult{AssetID: 7, Version: 2})
}

func TestExecutorEditObjectPrototypeRejectsInvalidStateAndDependencyFailures(t *testing.T) {
	wantLoadErr := errors.New("asset unavailable")
	wantResolveErr := errors.New("reference unavailable")
	wantRecordErr := errors.New("record unavailable")

	tests := []struct {
		name      string
		payload   json.RawMessage
		configure func(*generationAssetWriterStub, *referenceStoreStub)
		wantErr   error
		wantText  string
		withStore bool
	}{
		{
			name:     "malformed payload",
			payload:  json.RawMessage(`{`),
			wantText: "decode edit_object_prototype execution payload",
		},
		{
			name: "asset load failure",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				assets.detailErr = wantLoadErr
			},
			wantErr: wantLoadErr,
		},
		{
			name: "asset not found",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				assets.detailResult = &assetdomain.Asset{}
			},
			wantText: "object asset 7 not found",
		},
		{
			name: "wrong asset type",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Type = assetdomain.AssetTypeCharacter
				assets.detailResult = &asset
			},
			wantText: "unsupported for asset type",
		},
		{
			name: "invalid perspective",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Perspective = assetdomain.Perspective("sideways")
				assets.detailResult = &asset
			},
			wantText: "invalid perspective",
		},
		{
			name: "malformed dimensions",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Dimensions = json.RawMessage(`{`)
				assets.detailResult = &asset
			},
			wantText: "decode asset 7 dimensions",
		},
		{
			name: "nonpositive dimensions",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Dimensions = json.RawMessage(`{"width":0,"height":64}`)
				assets.detailResult = &asset
			},
			wantText: "dimensions must be positive",
		},
		{
			name: "malformed content",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Content = json.RawMessage(`{`)
				assets.detailResult = &asset
			},
			wantText: "decode object asset 7 content",
		},
		{
			name: "missing prototype",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Content = json.RawMessage(`{"directionCount":4}`)
				assets.detailResult = &asset
			},
			wantText: "prototype images are required",
		},
		{
			name: "missing prototype URL",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				asset := editableObjectAsset()
				asset.Content = json.RawMessage(`{"prototype":[{"id":1}]}`)
				assets.detailResult = &asset
			},
			wantText: "prototype image 1 URL is required",
		},
		{
			name: "reference resolution failure",
			configure: func(_ *generationAssetWriterStub, references *referenceStoreStub) {
				references.resolveErr = wantResolveErr
			},
			wantErr: wantResolveErr, withStore: true,
		},
		{
			name: "record creation failure",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				assets.recordErr = wantRecordErr
			},
			wantErr: wantRecordErr,
		},
		{
			name: "nil record",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				assets.nilRecord = true
			},
			wantText: "version: empty result",
		},
		{
			name: "zero record version",
			configure: func(assets *generationAssetWriterStub, _ *referenceStoreStub) {
				assets.emptyRecord = true
			},
			wantText: "version: empty result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []string{}
			images := &imageGenerationServiceStub{events: &events, result: generatedImages()}
			asset := editableObjectAsset()
			assets := &generationAssetWriterStub{events: &events, detailResult: &asset}
			references := &referenceStoreStub{events: &events}
			if tt.configure != nil {
				tt.configure(assets, references)
			}

			var executor generator.Executor
			if tt.withStore {
				executor = generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets, references)
			} else {
				executor = generator.NewExecutor(images, &imageProcessorStub{events: &events}, assets)
			}
			payload := tt.payload
			if payload == nil {
				payload = json.RawMessage(`{"asset_id":7,"edit_instructions":"change only the lock"}`)
			}

			_, err := executor.Generate(context.Background(), generator.EditObjectProtoType, payload)
			if err == nil {
				t.Fatal("expected edit failure")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected wrapped error %v, got %v", tt.wantErr, err)
			}
			if tt.wantText != "" && !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("expected error containing %q, got %v", tt.wantText, err)
			}
		})
	}
}

func editableObjectAsset() assetdomain.Asset {
	return assetdomain.Asset{
		ID:          7,
		Name:        "chest",
		ProjectID:   11,
		Type:        assetdomain.AssetTypeObject,
		Description: "a wooden chest with an iron lock",
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Content: json.RawMessage(`{
			"directionCount":4,
			"prototype":[
				{"id":1,"url":"assets/chest/up.png"},
				{"id":2,"url":"assets/chest/right.png"},
				{"id":3,"url":"assets/chest/down.png"},
				{"id":4,"url":"assets/chest/left.png"}
			],
			"animations":[{"id":7,"name":"idle","frames":[]}]
		}`),
		Version: 2,
	}
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

var _ imageclient.ImageGenerationService = (*imageGenerationServiceStub)(nil)
var _ imageprocessor.Processor = (*imageProcessorStub)(nil)
var _ generator.AssetWriter = (*generationAssetWriterStub)(nil)
