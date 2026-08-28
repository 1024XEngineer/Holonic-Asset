package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type derivationCoverageAssets struct {
	asset assetdomain.Asset
	err   error
}

func (s *derivationCoverageAssets) GetDetail(context.Context, uint) (assetdomain.Asset, error) {
	return s.asset, s.err
}

func (*derivationCoverageAssets) CreateCharacterAsset(context.Context, *assetdomain.Asset) (*assetdomain.Asset, error) {
	return nil, errors.New("unexpected character asset creation")
}

func (*derivationCoverageAssets) CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, errors.New("unexpected object asset creation")
}

func (*derivationCoverageAssets) CreateSceneryAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, errors.New("unexpected scenery asset creation")
}

func (*derivationCoverageAssets) CreateTileSetAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, errors.New("unexpected tileset asset creation")
}

type derivationCoverageReferences struct {
	mu          sync.Mutex
	resolved    map[string]string
	resolveErr  error
	persistErr  error
	persisted   []string
	deleted     []string
	persistDone chan struct{}
	doneOnce    sync.Once
}

func (s *derivationCoverageReferences) ResolveReference(_ context.Context, reference string) (string, error) {
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
	if resolved, ok := s.resolved[reference]; ok {
		return resolved, nil
	}
	return reference, nil
}

func (s *derivationCoverageReferences) PersistReference(_ context.Context, reference string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.persistErr != nil {
		return "", s.persistErr
	}
	s.persisted = append(s.persisted, reference)
	if s.persistDone != nil && len(s.persisted) >= 2 {
		s.doneOnce.Do(func() { close(s.persistDone) })
	}
	return fmt.Sprintf("uploads/coverage-%d.png", len(s.persisted)), nil
}

func (*derivationCoverageReferences) NewObjectKey(string) (string, error) {
	return "uploads/coverage.png", nil
}

func (*derivationCoverageReferences) PersistReferenceAt(context.Context, string, string) error {
	return nil
}

func (s *derivationCoverageReferences) DeleteObjects(_ context.Context, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, keys...)
	return nil
}

type derivationCoverageImages struct {
	result *imageclient.GenerateResult
	err    error
}

func (s *derivationCoverageImages) Generate(context.Context, *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	return s.result, s.err
}

type derivationCoverageAnimations struct {
	generate func(context.Context, *AnimationGenerationRequest) (*AnimationGenerationResult, error)
	result   *AnimationGenerationResult
	err      error
}

func (s *derivationCoverageAnimations) Generate(
	ctx context.Context,
	request *AnimationGenerationRequest,
) (*AnimationGenerationResult, error) {
	if s.generate != nil {
		return s.generate(ctx, request)
	}
	return s.result, s.err
}

type derivationCoverageProcessor struct {
	delegate      imageprocessor.Processor
	removeErr     error
	removeEmpty   bool
	resizeErr     error
	splitErr      error
	splitOverride bool
	splitResult   *imageprocessor.SplitImageResult
}

func (s *derivationCoverageProcessor) RemoveBackground(
	ctx context.Context,
	request *imageprocessor.RemoveBackgroundRequest,
) (*imageprocessor.RemoveBackgroundResult, error) {
	if s.removeErr != nil {
		return nil, s.removeErr
	}
	if s.removeEmpty {
		return &imageprocessor.RemoveBackgroundResult{}, nil
	}
	return s.delegate.RemoveBackground(ctx, request)
}

func (s *derivationCoverageProcessor) NormalizeReference(
	ctx context.Context,
	request *imageprocessor.NormalizeReferenceRequest,
) (*imageprocessor.NormalizeReferenceResult, error) {
	return s.delegate.NormalizeReference(ctx, request)
}

func (s *derivationCoverageProcessor) Resize(
	ctx context.Context,
	request *imageprocessor.ResizeRequest,
) (*imageprocessor.ResizeResult, error) {
	if s.resizeErr != nil {
		return nil, s.resizeErr
	}
	return s.delegate.Resize(ctx, request)
}

func (s *derivationCoverageProcessor) Verify(
	ctx context.Context,
	request *imageprocessor.VerifyRequest,
) (*imageprocessor.VerificationReport, error) {
	return s.delegate.Verify(ctx, request)
}

func (s *derivationCoverageProcessor) SplitImage(
	ctx context.Context,
	request *imageprocessor.SplitImageRequest,
) (*imageprocessor.SplitImageResult, error) {
	if s.splitErr != nil {
		return nil, s.splitErr
	}
	if s.splitOverride {
		return s.splitResult, nil
	}
	return s.delegate.SplitImage(ctx, request)
}

type derivationRoundTripper func(*http.Request) (*http.Response, error)

func (f derivationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type derivationReadError struct{}

func (derivationReadError) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (derivationReadError) Close() error             { return nil }

func TestDeriveAnimationValidatesPayloadAndAssetState(t *testing.T) {
	validAsset := derivationCoverageAsset(t)
	validPayload := DeriveAnimationPayload{
		AssetID: 7, ProjectID: 11, SourceAnimationID: 3, TargetDirections: []string{AnimationDirectionFront},
	}
	newExecutor := func(asset assetdomain.Asset, assetErr error) *executor {
		return &executor{
			assets:     &derivationCoverageAssets{asset: asset, err: assetErr},
			animations: &derivationCoverageAnimations{result: derivationCoverageAnimationResult(t)},
			images:     &derivationCoverageImages{},
			processor:  imageprocessor.NewProcessor(),
			references: &derivationCoverageReferences{},
		}
	}

	for _, test := range []struct {
		name    string
		payload DeriveAnimationPayload
		want    string
	}{
		{name: "asset", payload: DeriveAnimationPayload{}, want: "asset is required"},
		{name: "project", payload: DeriveAnimationPayload{AssetID: 7}, want: "project is required"},
		{name: "source", payload: DeriveAnimationPayload{AssetID: 7, ProjectID: 11}, want: "source animation is required"},
		{name: "targets", payload: DeriveAnimationPayload{AssetID: 7, ProjectID: 11, SourceAnimationID: 3}, want: "target directions are required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newExecutor(validAsset, nil).deriveAnimation(context.Background(), test.payload)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}

	t.Run("asset read", func(t *testing.T) {
		_, err := newExecutor(validAsset, errors.New("database unavailable")).deriveAnimation(context.Background(), validPayload)
		if err == nil || !strings.Contains(err.Error(), "get animation derivation asset") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	assetCases := []struct {
		name   string
		mutate func(*assetdomain.Asset, *assetdomain.AssetContent)
		want   string
	}{
		{name: "not found", mutate: func(asset *assetdomain.Asset, _ *assetdomain.AssetContent) { asset.ID = 0 }, want: "not found"},
		{name: "project mismatch", mutate: func(asset *assetdomain.Asset, _ *assetdomain.AssetContent) { asset.ProjectID = 12 }, want: "belongs to project"},
		{name: "unsupported direction count", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) { content.DirectionCount = 3 }, want: "requires 2, 4, or 8"},
		{name: "source missing", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) { content.Animations = nil }, want: "source 3 not found"},
		{name: "generation missing", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) { content.Animations[0].Generation = nil }, want: "has no generation configuration"},
		{name: "frames missing", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) { content.Animations[0].Frames = nil }, want: "has no frames"},
		{name: "source direction invalid", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) {
			content.Animations[0].Generation.Direction = "diagonal"
		}, want: "invalid source animation direction"},
		{name: "duplicate group direction", mutate: func(_ *assetdomain.Asset, content *assetdomain.AssetContent) {
			duplicate := content.Animations[0]
			duplicate.ID = 4
			duplicate.GroupID = 3
			content.Animations = append(content.Animations, duplicate)
		}, want: "contains duplicate direction"},
	}
	for _, test := range assetCases {
		t.Run(test.name, func(t *testing.T) {
			asset := validAsset
			content, err := asset.DecodeContent()
			if err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			test.mutate(&asset, &content)
			asset.Content, err = assetdomain.EncodeContent(content)
			if err != nil {
				t.Fatalf("encode fixture: %v", err)
			}
			_, err = newExecutor(asset, nil).deriveAnimation(context.Background(), validPayload)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}

	t.Run("malformed content", func(t *testing.T) {
		asset := validAsset
		asset.Content = json.RawMessage(`{`)
		_, err := newExecutor(asset, nil).deriveAnimation(context.Background(), validPayload)
		if err == nil || !strings.Contains(err.Error(), "decode animation derivation asset") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nonzero group ignores incomplete peers", func(t *testing.T) {
		asset := validAsset
		asset.Description = ""
		content, err := asset.DecodeContent()
		if err != nil {
			t.Fatalf("decode fixture: %v", err)
		}
		content.Animations[0].GroupID = 9
		content.Animations = append(content.Animations,
			assetdomain.Animation{ID: 10, GroupID: 8},
			assetdomain.Animation{ID: 11, GroupID: 9},
			assetdomain.Animation{ID: 12, GroupID: 9, Generation: &assetdomain.AnimationGenerationConfig{}},
		)
		asset.Content, err = assetdomain.EncodeContent(content)
		if err != nil {
			t.Fatalf("encode fixture: %v", err)
		}
		if _, err := newExecutor(asset, nil).deriveAnimation(context.Background(), validPayload); err != nil {
			t.Fatalf("derive with incomplete peers: %v", err)
		}
	})
}

func TestDerivationGenerationRequestValidatesSourceSettings(t *testing.T) {
	asset := derivationCoverageAsset(t)
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	source := content.Animations[0]

	withoutGeneration := source
	withoutGeneration.Generation = nil
	if _, err := derivationGenerationRequest(asset, withoutGeneration); err == nil {
		t.Fatal("expected missing generation error")
	}

	invalidAsset := asset
	invalidAsset.Dimensions = json.RawMessage(`{`)
	if _, err := derivationGenerationRequest(invalidAsset, source); err == nil {
		t.Fatal("expected invalid dimensions error")
	}

	mismatchedDimensions := source
	generation := *source.Generation
	generation.FrameWidth = 64
	generation.FrameHeight = 0
	mismatchedDimensions.Generation = &generation
	if _, err := derivationGenerationRequest(asset, mismatchedDimensions); err == nil {
		t.Fatal("expected mismatched frame dimensions error")
	}

	invalidFPS := source
	generation = *source.Generation
	generation.FPS = 61
	invalidFPS.Generation = &generation
	if _, err := derivationGenerationRequest(asset, invalidFPS); err == nil || !strings.Contains(err.Error(), "normalize") {
		t.Fatalf("expected normalized settings error, got %v", err)
	}

	fallback := source
	generation = *source.Generation
	generation.Columns = 20
	fallback.Generation = &generation
	asset.Description = ""
	request, err := derivationGenerationRequest(asset, fallback)
	if err != nil {
		t.Fatalf("build fallback request: %v", err)
	}
	if request.Description != asset.Name || request.Columns != 2 {
		t.Fatalf("unexpected fallback request: %+v", request)
	}
}

func TestDeriveAnimationCleansSuccessfulSiblingResourcesAfterFailure(t *testing.T) {
	asset := derivationCoverageAsset(t)
	persistDone := make(chan struct{})
	references := &derivationCoverageReferences{persistDone: persistDone}
	result := derivationCoverageAnimationResult(t)
	animations := &derivationCoverageAnimations{generate: func(
		ctx context.Context,
		request *AnimationGenerationRequest,
	) (*AnimationGenerationResult, error) {
		if strings.Contains(request.TargetOrientation, "Left") {
			return result, nil
		}
		select {
		case <-persistDone:
			return nil, errors.New("right generation failed")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	executor := &executor{
		assets:     &derivationCoverageAssets{asset: asset},
		animations: animations,
		images:     &derivationCoverageImages{},
		processor:  imageprocessor.NewProcessor(),
		references: references,
	}
	_, err := executor.deriveAnimation(context.Background(), DeriveAnimationPayload{
		AssetID: 7, ProjectID: 11, SourceAnimationID: 3,
		TargetDirections: []string{AnimationDirectionLeft, AnimationDirectionRight},
	})
	if err == nil || !strings.Contains(err.Error(), "right generation failed") {
		t.Fatalf("unexpected derivation error: %v", err)
	}
	if len(references.deleted) != 2 {
		t.Fatalf("deleted resources = %v, want both successful sibling frames", references.deleted)
	}
}

func TestAnimationDerivationHelpersRejectInvalidFramesAndReferences(t *testing.T) {
	executor := &executor{references: &derivationCoverageReferences{}, processor: imageprocessor.NewProcessor()}
	if _, err := executor.animationDerivationFrameSheet(context.Background(), assetdomain.Animation{}); err == nil {
		t.Fatal("expected missing generation error")
	}
	if _, err := executor.animationDerivationFrameSheet(context.Background(), assetdomain.Animation{
		Generation: &assetdomain.AnimationGenerationConfig{Columns: 1},
	}); err == nil {
		t.Fatal("expected missing frames error")
	}
	if _, err := executor.animationDerivationFrameSheet(context.Background(), assetdomain.Animation{
		Generation: &assetdomain.AnimationGenerationConfig{Columns: 1},
		Frames:     []assetdomain.Frame{{ID: 4}},
	}); err == nil {
		t.Fatal("expected missing frame URL error")
	}
	badReference := "invalid-image"
	if _, err := executor.animationDerivationFrameSheet(context.Background(), assetdomain.Animation{
		Generation: &assetdomain.AnimationGenerationConfig{Columns: 1},
		Frames:     []assetdomain.Frame{{ID: 4, URL: &badReference}},
	}); err == nil {
		t.Fatal("expected frame load error")
	}

	frame := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	if _, err := packAnimationDerivationFrames(nil, 1); err == nil {
		t.Fatal("expected empty frame error")
	}
	if _, err := packAnimationDerivationFrames([]image.Image{frame}, 0); err == nil {
		t.Fatal("expected invalid columns error")
	}
	if _, err := packAnimationDerivationFrames([]image.Image{image.NewNRGBA(image.Rect(0, 0, 0, 0))}, 1); err == nil {
		t.Fatal("expected invalid dimensions error")
	}
	if _, err := packAnimationDerivationFrames([]image.Image{frame, image.NewNRGBA(image.Rect(0, 0, 3, 4))}, 2); err == nil {
		t.Fatal("expected mismatched dimensions error")
	}

	for direction, want := range map[string]string{
		AnimationDirectionFront:      "Front / South",
		AnimationDirectionFrontRight: "Front-right",
		AnimationDirectionRight:      "Right / East",
		AnimationDirectionBackRight:  "Back-right",
		AnimationDirectionBack:       "Back / North",
		AnimationDirectionBackLeft:   "Back-left",
		AnimationDirectionLeft:       "Left / West",
		AnimationDirectionFrontLeft:  "Front-left",
		" custom ":                   "custom",
	} {
		if got := animationDirectionDescription(direction); !strings.Contains(got, want) {
			t.Errorf("direction %q: got %q, want %q", direction, got, want)
		}
	}
	if !animationBelongsToGroup(assetdomain.Animation{ID: 3}, assetdomain.Animation{ID: 3}, 9) ||
		!animationBelongsToGroup(assetdomain.Animation{ID: 9}, assetdomain.Animation{ID: 3}, 9) ||
		!animationBelongsToGroup(assetdomain.Animation{GroupID: 9}, assetdomain.Animation{ID: 3}, 9) {
		t.Fatal("expected all supported animation group relationships")
	}
}

func TestAnimationDerivationLoadsProcessedAndUnprocessedReferences(t *testing.T) {
	dataURL := derivationCoverageFrameDataURL(t)
	references := &derivationCoverageReferences{resolved: map[string]string{
		"uploads/frame.png":                 "invalid",
		"uploads/frame-unprocessed.png":     dataURL,
		"uploads/prototype.png":             dataURL,
		"uploads/prototype-unprocessed.png": "invalid",
	}}
	executor := &executor{references: references}
	if _, err := executor.loadAnimationFrameImage(context.Background(), "uploads/frame.png"); err != nil {
		t.Fatalf("load unprocessed frame fallback: %v", err)
	}
	if _, err := executor.loadAnimationFrameImage(context.Background(), "missing/frame.png"); err == nil ||
		!strings.Contains(err.Error(), "load processed frame") {
		t.Fatalf("expected both frame load errors, got %v", err)
	}

	asset := derivationCoverageAsset(t)
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	prototypeURL := "uploads/prototype.png"
	(*content.Prototype)[0].URL = &prototypeURL
	asset.Content, err = assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if _, err := executor.loadAnimationPrototypeImage(context.Background(), asset, AnimationDirectionFront); err != nil {
		t.Fatalf("load processed prototype fallback: %v", err)
	}

	asset.Content = json.RawMessage(`{`)
	if _, err := executor.loadAnimationPrototypeImage(context.Background(), asset, AnimationDirectionFront); err == nil {
		t.Fatal("expected malformed prototype content error")
	}
	asset = derivationCoverageAsset(t)
	if _, err := executor.loadAnimationPrototypeImage(context.Background(), asset, "diagonal"); err == nil {
		t.Fatal("expected invalid prototype direction error")
	}
	content, _ = asset.DecodeContent()
	content.Prototype = nil
	asset.Content, _ = assetdomain.EncodeContent(content)
	if _, err := executor.loadAnimationPrototypeImage(context.Background(), asset, AnimationDirectionFront); err == nil {
		t.Fatal("expected missing prototype error")
	}
	asset = derivationCoverageAsset(t)
	content, _ = asset.DecodeContent()
	(*content.Prototype)[0].URL = nil
	asset.Content, _ = assetdomain.EncodeContent(content)
	if _, err := executor.loadAnimationPrototypeImage(context.Background(), asset, AnimationDirectionFront); err == nil {
		t.Fatal("expected missing prototype URL error")
	}
}

func TestLoadAnimationDerivationImageHandlesDownloadFailures(t *testing.T) {
	dataURL := derivationCoverageFrameDataURL(t)
	encoded := strings.TrimPrefix(dataURL, "data:image/png;base64,")
	pngBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	if _, err := (&executor{references: &derivationCoverageReferences{}}).loadAnimationDerivationImage(
		context.Background(), " ",
	); err == nil {
		t.Fatal("expected empty reference error")
	}
	if _, err := (&executor{references: &derivationCoverageReferences{resolveErr: errors.New("resolve failed")}}).
		loadAnimationDerivationImage(context.Background(), "uploads/frame.png"); err == nil {
		t.Fatal("expected resolve error")
	}
	if _, err := (&executor{references: &derivationCoverageReferences{resolved: map[string]string{
		"private": "http://127.0.0.1/frame.png",
	}}}).loadAnimationDerivationImage(context.Background(), "private"); err == nil {
		t.Fatal("expected private URL rejection")
	}

	tests := []struct {
		name      string
		transport derivationRoundTripper
		wantError string
	}{
		{name: "transport", transport: func(*http.Request) (*http.Response, error) {
			return nil, errors.New("download failed")
		}, wantError: "download image reference"},
		{name: "status", transport: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("upstream failed"))}, nil
		}, wantError: "HTTP 502"},
		{name: "read", transport: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: derivationReadError{}}, nil
		}, wantError: "read image reference"},
		{name: "empty", transport: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		}, wantError: "size 0 is invalid"},
		{name: "decode", transport: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("not-an-image"))}, nil
		}, wantError: "decode downloaded image"},
		{name: "success", transport: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(pngBytes)))}, nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &executor{
				references: &derivationCoverageReferences{resolved: map[string]string{
					"remote": "https://example.com/frame.png",
				}},
				referenceHTTPClient: &http.Client{Transport: test.transport},
			}
			loaded, err := executor.loadAnimationDerivationImage(context.Background(), "remote")
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("got %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || loaded == nil {
				t.Fatalf("load downloaded image: image=%v err=%v", loaded, err)
			}
		})
	}
}

func TestAnimationDerivationGenerationPathsReturnActionableErrors(t *testing.T) {
	asset := derivationCoverageAsset(t)
	content, err := asset.DecodeContent()
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	source := content.Animations[0]
	base, err := derivationGenerationRequest(asset, source)
	if err != nil {
		t.Fatalf("build base request: %v", err)
	}

	videoTests := []struct {
		name       string
		target     string
		source     assetdomain.Animation
		animations *derivationCoverageAnimations
		references *derivationCoverageReferences
		want       string
	}{
		{name: "target", target: "diagonal", source: source, animations: &derivationCoverageAnimations{}, references: &derivationCoverageReferences{}, want: "unavailable"},
		{name: "source sheet", target: AnimationDirectionFront, source: assetdomain.Animation{}, animations: &derivationCoverageAnimations{}, references: &derivationCoverageReferences{}, want: "generation configuration"},
		{name: "provider", target: AnimationDirectionFront, source: source, animations: &derivationCoverageAnimations{err: errors.New("video failed")}, references: &derivationCoverageReferences{}, want: "generate direction video"},
		{name: "frame count", target: AnimationDirectionFront, source: source, animations: &derivationCoverageAnimations{result: &AnimationGenerationResult{}}, references: &derivationCoverageReferences{}, want: "expected 2"},
		{name: "persistence", target: AnimationDirectionFront, source: source, animations: &derivationCoverageAnimations{result: derivationCoverageAnimationResult(t)}, references: &derivationCoverageReferences{persistErr: errors.New("store failed")}, want: "persist animation frame"},
	}
	for _, test := range videoTests {
		t.Run("video "+test.name, func(t *testing.T) {
			executor := &executor{animations: test.animations, references: test.references}
			_, err := executor.deriveAnimationFramesWithVideo(context.Background(), asset, base, test.target, test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}

	transparentSheet := derivationCoverageSheetBase64(t)
	imageTests := []struct {
		name       string
		images     *derivationCoverageImages
		processor  imageprocessor.Processor
		references *derivationCoverageReferences
		want       string
	}{
		{name: "provider", images: &derivationCoverageImages{err: errors.New("image failed")}, processor: imageprocessor.NewProcessor(), references: &derivationCoverageReferences{}, want: "edit animation direction"},
		{name: "empty result", images: &derivationCoverageImages{}, processor: imageprocessor.NewProcessor(), references: &derivationCoverageReferences{}, want: "expected exactly one"},
		{name: "decode", images: &derivationCoverageImages{result: &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: "invalid"}}}}, processor: imageprocessor.NewProcessor(), references: &derivationCoverageReferences{}, want: "decode generated"},
		{name: "split", images: &derivationCoverageImages{result: derivationCoverageImageResult(transparentSheet)}, processor: &derivationCoverageProcessor{delegate: imageprocessor.NewProcessor(), splitErr: errors.New("split failed")}, references: &derivationCoverageReferences{}, want: "split generated"},
		{name: "split count", images: &derivationCoverageImages{result: derivationCoverageImageResult(transparentSheet)}, processor: &derivationCoverageProcessor{delegate: imageprocessor.NewProcessor(), splitOverride: true}, references: &derivationCoverageReferences{}, want: "expected 2"},
		{name: "resize", images: &derivationCoverageImages{result: derivationCoverageImageResult(transparentSheet)}, processor: &derivationCoverageProcessor{delegate: imageprocessor.NewProcessor(), resizeErr: errors.New("resize failed")}, references: &derivationCoverageReferences{}, want: "pixel-process"},
		{name: "persistence", images: &derivationCoverageImages{result: derivationCoverageImageResult(transparentSheet)}, processor: imageprocessor.NewProcessor(), references: &derivationCoverageReferences{persistErr: errors.New("store failed")}, want: "persist animation frame"},
	}
	for _, test := range imageTests {
		t.Run("image "+test.name, func(t *testing.T) {
			executor := &executor{images: test.images, processor: test.processor, references: test.references}
			_, err := executor.deriveAnimationFramesWithImage(
				context.Background(), asset, base, AnimationDirectionRight, source,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}

	greenImage, err := imageprocessor.DecodeBase64Image(strings.TrimPrefix(derivationCoverageOpaqueDataURL(t), "data:image/png;base64,"))
	if err != nil {
		t.Fatalf("decode opaque fixture: %v", err)
	}
	for _, test := range []struct {
		name      string
		processor *derivationCoverageProcessor
		want      string
	}{
		{name: "remove", processor: &derivationCoverageProcessor{delegate: imageprocessor.NewProcessor(), removeErr: errors.New("remove failed")}, want: "remove generated"},
		{name: "empty", processor: &derivationCoverageProcessor{delegate: imageprocessor.NewProcessor(), removeEmpty: true}, want: "empty result"},
	} {
		t.Run("background "+test.name, func(t *testing.T) {
			executor := &executor{processor: test.processor}
			_, err := executor.removeAnimationDerivationBackground(
				context.Background(), strings.TrimPrefix(derivationCoverageOpaqueDataURL(t), "data:image/png;base64,"), greenImage,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}
}

func TestDeriveAnimationEntryPointsAndTaskPayload(t *testing.T) {
	assetID := uint(7)
	validRequest := &Request{
		Kind: DeriveAnimation, ProjectID: 11, AssetID: &assetID,
		Parameters: json.RawMessage(`{"source_animation_id":3,"target_directions":["front"]}`),
	}
	payload, err := buildTaskPayload(validRequest)
	if err != nil {
		t.Fatalf("build derive payload: %v", err)
	}
	derived := payload.(DeriveAnimationPayload)
	if derived.SourceAnimationID != 3 || len(derived.TargetDirections) != 1 {
		t.Fatalf("unexpected derive payload: %+v", derived)
	}

	for _, test := range []struct {
		name    string
		request *Request
		want    string
	}{
		{name: "parameters", request: &Request{Kind: DeriveAnimation, AssetID: &assetID, Parameters: json.RawMessage(`{"unknown":1}`)}, want: "decode derive_animation parameters"},
		{name: "asset", request: &Request{Kind: DeriveAnimation}, want: "asset id is required"},
		{name: "path", request: &Request{Kind: DeriveAnimation, AssetID: &assetID, TargetAssetPaths: []string{"frames.3"}}, want: "does not identify an animation"},
		{name: "conflict", request: &Request{Kind: DeriveAnimation, AssetID: &assetID, TargetAssetPaths: []string{"animations.4"}, Parameters: json.RawMessage(`{"source_animation_id":3,"target_directions":["front"]}`)}, want: "conflicts"},
		{name: "source", request: &Request{Kind: DeriveAnimation, AssetID: &assetID, Parameters: json.RawMessage(`{"target_directions":["front"]}`)}, want: "source animation id is required"},
		{name: "directions", request: &Request{Kind: DeriveAnimation, AssetID: &assetID, Parameters: json.RawMessage(`{"source_animation_id":3}`)}, want: "target directions are required"},
	} {
		t.Run("payload "+test.name, func(t *testing.T) {
			_, err := buildTaskPayload(test.request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}
	pathRequest := *validRequest
	pathRequest.TargetAssetPaths = []string{"animations.5"}
	pathRequest.Parameters = json.RawMessage(`{"target_directions":["front"]}`)
	payload, err = buildTaskPayload(&pathRequest)
	if err != nil || payload.(DeriveAnimationPayload).SourceAnimationID != 5 {
		t.Fatalf("build path-selected derive payload: payload=%+v err=%v", payload, err)
	}

	validJSON := json.RawMessage(`{"asset_id":7,"project_id":11,"source_animation_id":3,"target_directions":["front"]}`)
	dummyAssets := &derivationCoverageAssets{}
	dummyAnimations := &derivationCoverageAnimations{}
	dummyImages := &derivationCoverageImages{}
	dummyProcessor := imageprocessor.NewProcessor()
	dummyReferences := &derivationCoverageReferences{}
	dependencyTests := []struct {
		name     string
		executor *executor
		want     error
	}{
		{name: "assets", executor: &executor{animations: dummyAnimations, images: dummyImages, processor: dummyProcessor, references: dummyReferences}, want: ErrAssetWriterRequired},
		{name: "animations", executor: &executor{assets: dummyAssets, images: dummyImages, processor: dummyProcessor, references: dummyReferences}, want: ErrAnimationServiceRequired},
		{name: "images", executor: &executor{assets: dummyAssets, animations: dummyAnimations, processor: dummyProcessor, references: dummyReferences}, want: ErrImageServiceRequired},
		{name: "processor", executor: &executor{assets: dummyAssets, animations: dummyAnimations, images: dummyImages, references: dummyReferences}, want: ErrImageProcessorRequired},
		{name: "references", executor: &executor{assets: dummyAssets, animations: dummyAnimations, images: dummyImages, processor: dummyProcessor}, want: ErrAnimationReferenceStoreRequired},
	}
	for _, test := range dependencyTests {
		t.Run("dependency "+test.name, func(t *testing.T) {
			_, err := test.executor.Generate(context.Background(), DeriveAnimation, validJSON)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	fullExecutor := &executor{assets: dummyAssets, animations: dummyAnimations, images: dummyImages, processor: dummyProcessor, references: dummyReferences}
	if _, err := fullExecutor.Generate(context.Background(), DeriveAnimation, json.RawMessage(`{`)); err == nil {
		t.Fatal("expected malformed execution payload")
	}

	request := &AnimationGenerationRequest{
		ReferenceImage: "ref", ReferenceImageContext: true, DerivationSourceImage: " sheet ",
		EndReferenceImage: "end", ContextReferenceImages: []string{"one"}, FrameCount: 1,
		Columns: 1, FrameWidth: 32, FrameHeight: 32, FPS: 10, Duration: 5,
	}
	if _, err := normalizeAnimationGenerationRequest(request); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected edit and derivation conflict, got %v", err)
	}

	handlerExecutor := &mockExecutor{result: json.RawMessage(`{"ok":true}`)}
	engine := &Engine{executor: handlerExecutor}
	if _, err := engine.handleDeriveAnimation(context.Background(), &taskdomain.Task{Payload: json.RawMessage(`{`)}); err == nil {
		t.Fatal("expected malformed handler payload")
	}
	if _, err := engine.handleDeriveAnimation(context.Background(), &taskdomain.Task{Payload: validJSON}); err != nil {
		t.Fatalf("handle derive animation: %v", err)
	}
}

func derivationCoverageAsset(t *testing.T) assetdomain.Asset {
	t.Helper()
	dataURL := derivationCoverageFrameDataURL(t)
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 4
	prototype := make(assetdomain.Prototype, 4)
	for index := range prototype {
		url := dataURL
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &url}
	}
	content.Prototype = &prototype
	content.Animations = []assetdomain.Animation{{
		ID: 3, Name: "spray",
		Frames: []assetdomain.Frame{{ID: 1, URL: &dataURL, Duration: 100}, {ID: 2, URL: &dataURL, Duration: 100}},
		Generation: &assetdomain.AnimationGenerationConfig{
			Direction: AnimationDirectionBack, Style: "pixel art", Action: "spray",
			FrameCount: 2, Columns: 2, FrameWidth: 32, FrameHeight: 32,
			FPS: 10, Resolution: "720p", Duration: 5, AspectRatio: "1:1",
		},
	}}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		t.Fatalf("encode coverage asset: %v", err)
	}
	return assetdomain.Asset{
		ID: 7, ProjectID: 11, Version: 2, Type: assetdomain.AssetTypeCharacter,
		Name: "coverage robot", Description: "blue coverage robot",
		Dimensions: json.RawMessage(`{"width":32,"height":32}`), Content: encoded,
	}
}

func derivationCoverageFrameDataURL(t *testing.T) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 8; y < 28; y++ {
		for x := 10; x < 22; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode frame fixture: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func derivationCoverageOpaqueDataURL(t *testing.T) string {
	t.Helper()
	frame := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			frame.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}
	for y := 8; y < 28; y++ {
		for x := 10; x < 22; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{R: 40, G: 80, B: 180, A: 255})
		}
	}
	encoded, err := imageprocessor.EncodePNGBase64(frame)
	if err != nil {
		t.Fatalf("encode opaque fixture: %v", err)
	}
	return "data:image/png;base64," + encoded
}

func derivationCoverageSheetBase64(t *testing.T) string {
	t.Helper()
	frame, err := imageprocessor.DecodeBase64Image(
		strings.TrimPrefix(derivationCoverageFrameDataURL(t), "data:image/png;base64,"),
	)
	if err != nil {
		t.Fatalf("decode frame fixture: %v", err)
	}
	sheet, err := packAnimationDerivationFrames([]image.Image{frame, frame}, 2)
	if err != nil {
		t.Fatalf("pack sheet fixture: %v", err)
	}
	encoded, err := imageprocessor.EncodePNGBase64(sheet)
	if err != nil {
		t.Fatalf("encode sheet fixture: %v", err)
	}
	return encoded
}

func derivationCoverageAnimationResult(t *testing.T) *AnimationGenerationResult {
	t.Helper()
	base64Frame := strings.TrimPrefix(derivationCoverageFrameDataURL(t), "data:image/png;base64,")
	return &AnimationGenerationResult{
		Frames: []imageprocessor.ImageRegion{
			{ImageBase64: base64Frame, MIMEType: "image/png"},
			{ImageBase64: base64Frame, MIMEType: "image/png"},
		},
		FrameDurationMS: 100,
	}
}

func derivationCoverageImageResult(base64Image string) *imageclient.GenerateResult {
	return &imageclient.GenerateResult{Images: []imageclient.GeneratedImage{{Base64: base64Image, MediaType: "image/png"}}}
}

var _ AssetWriter = (*derivationCoverageAssets)(nil)
var _ ReferenceStore = (*derivationCoverageReferences)(nil)
var _ imageclient.ImageGenerationService = (*derivationCoverageImages)(nil)
var _ AnimationGenerationService = (*derivationCoverageAnimations)(nil)
var _ imageprocessor.Processor = (*derivationCoverageProcessor)(nil)
