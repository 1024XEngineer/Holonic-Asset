package generator_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	events   *[]string
	request  *generator.AnimationGenerationRequest
	requests []*generator.AnimationGenerationRequest
	result   *generator.AnimationGenerationResult
	results  []*generator.AnimationGenerationResult
	err      error
}

func (s *animationGenerationServiceStub) Generate(
	_ context.Context,
	request *generator.AnimationGenerationRequest,
) (*generator.AnimationGenerationResult, error) {
	*s.events = append(*s.events, "generate_animation")
	copy := *request
	copy.TargetFrameIndices = append([]int(nil), request.TargetFrameIndices...)
	copy.ContextReferenceImages = append([]string(nil), request.ContextReferenceImages...)
	s.request = &copy
	s.requests = append(s.requests, &copy)
	call := len(s.requests) - 1
	if call < len(s.results) {
		return s.results[call], s.err
	}
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

type executorReferenceStoreStub struct {
	resolved      []string
	resolveValues map[string]string
	persisted     []string
	persistValue  string
	uploads       []referenceUpload
	events        *[]string
	resolveErr    error
	persistErr    error
}

func (s *executorReferenceStoreStub) ResolveReference(_ context.Context, reference string) (string, error) {
	s.resolved = append(s.resolved, reference)
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
	if value, ok := s.resolveValues[reference]; ok {
		return value, nil
	}
	return "signed:" + reference, nil
}

func (s *executorReferenceStoreStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persisted = append(s.persisted, reference)
	if s.persistErr != nil {
		return "", s.persistErr
	}
	if s.persistValue != "" {
		return s.persistValue, nil
	}
	return fmt.Sprintf("uploads/generated-%d.png", len(s.persisted)), nil
}

func (s *executorReferenceStoreStub) NewObjectKey(_ string) (string, error) {
	if s.events != nil {
		*s.events = append(*s.events, "allocate_key")
	}
	return "uploads/prototype.png", nil
}

func (s *executorReferenceStoreStub) PersistReferenceAt(_ context.Context, key, reference string) error {
	if s.events != nil {
		*s.events = append(*s.events, "persist:"+key)
	}
	s.uploads = append(s.uploads, referenceUpload{key: key, reference: reference})
	if s.persistErr != nil {
		return s.persistErr
	}
	return nil
}

func (s *executorReferenceStoreStub) DeleteObjects(context.Context, []string) error {
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
		MaskImage:       request.MaskImage,
		N:               request.N,
		Model:           request.Model,
		Size:            request.Size,
		Params:          request.Params,
	}
	return s.result, s.err
}

type generationAssetWriterStub struct {
	events                  *[]string
	parentAsset             assetdomain.Asset
	getDetailErr            error
	characterAsset          *assetdomain.Asset
	objectAsset             *assetdomain.Asset
	sceneryAsset            *assetdomain.Asset
	createdRecord           *assetdomain.AssetRecord
	recordVersion           uint
	expectedVersion         uint
	animationAssetID        uint
	animation               assetdomain.Animation
	animationName           string
	animationID             uint
	frames                  []assetdomain.Frame
	updatedAnimationAssetID uint
	updatedAnimationID      uint
	updatedFrames           []assetdomain.Frame
	updateAnimationErr      error
	updateCalls             int
	err                     error
	detailErr               error
	recordErr               error
	detailResult            *assetdomain.Asset
	nilRecord               bool
	emptyRecord             bool
	asset                   assetdomain.Asset
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
	if s.detailErr != nil {
		return assetdomain.Asset{}, s.detailErr
	}
	if s.err != nil {
		return assetdomain.Asset{}, s.err
	}
	if s.parentAsset.ID == assetID {
		if s.updateCalls > 0 && s.detailResult != nil {
			return *s.detailResult, nil
		}
		return s.parentAsset, nil
	}
	if s.detailResult != nil {
		return *s.detailResult, nil
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

func (s *generationAssetWriterStub) CreateTileSetAsset(
	_ context.Context,
	value *assetdomain.Asset,
) (uint, error) {
	s.asset = *value
	if s.err != nil {
		return 0, s.err
	}
	return 43, nil
}

func (s *generationAssetWriterStub) CreateAnimation(
	_ context.Context,
	assetID uint,
	animation assetdomain.Animation,
) (uint, error) {
	*s.events = append(*s.events, "create_animation")
	s.animationAssetID = assetID
	s.animation = animation
	s.animation.Frames = append([]assetdomain.Frame(nil), animation.Frames...)
	s.animationName = animation.Name
	s.frames = append([]assetdomain.Frame(nil), animation.Frames...)
	if s.err != nil {
		return 0, s.err
	}
	s.animationID = 3
	return 3, nil
}

func (s *generationAssetWriterStub) UpdateAnimationFrames(
	_ context.Context,
	assetID uint,
	animationID uint,
	frames []assetdomain.Frame,
) error {
	if s.events != nil {
		*s.events = append(*s.events, "update_animation_frames")
	}
	s.updatedAnimationAssetID = assetID
	s.updatedAnimationID = animationID
	s.updatedFrames = append([]assetdomain.Frame(nil), frames...)
	s.frames = append([]assetdomain.Frame(nil), frames...)
	s.updateCalls++
	if s.updateAnimationErr != nil {
		return s.updateAnimationErr
	}
	return s.err
}

func (s *generationAssetWriterStub) CreateRecord(
	_ context.Context,
	record *assetdomain.AssetRecord,
	expectedVersion uint,
) (*assetdomain.AssetRecord, error) {
	s.expectedVersion = expectedVersion
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
		Dimensions:  json.RawMessage(`{"width":128,"height":128}`),
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
