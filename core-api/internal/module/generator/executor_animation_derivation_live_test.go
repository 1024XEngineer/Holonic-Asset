package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/viperx"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

const productionDerivationLiveEnv = "HOLONIC_PRODUCTION_DERIVATION"

type productionDerivationAssetStore struct {
	mu    sync.Mutex
	asset assetdomain.Asset
}

func (s *productionDerivationAssetStore) GetDetail(_ context.Context, id uint) (assetdomain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.asset.ID != id {
		return assetdomain.Asset{}, nil
	}
	return s.asset, nil
}

func (s *productionDerivationAssetStore) CreateCharacterAsset(
	_ context.Context,
	value *assetdomain.Asset,
) (*assetdomain.Asset, error) {
	if value == nil {
		return nil, fmt.Errorf("live derivation: character asset is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	created := *value
	created.ID = 7001
	created.Version = 1
	s.asset = created
	result := created
	return &result, nil
}

func (*productionDerivationAssetStore) CreateObjectAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, fmt.Errorf("live derivation: object creation is unsupported")
}

func (*productionDerivationAssetStore) CreateSceneryAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, fmt.Errorf("live derivation: scenery creation is unsupported")
}

func (*productionDerivationAssetStore) CreateTileSetAsset(context.Context, *assetdomain.Asset) (uint, error) {
	return 0, fmt.Errorf("live derivation: tileset creation is unsupported")
}

func (s *productionDerivationAssetStore) applyAnimations(candidates []assetdomain.Animation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, err := s.asset.DecodeContent()
	if err != nil {
		return err
	}
	for index := range candidates {
		candidate := candidates[index]
		candidate.ID = uint(len(content.Animations) + 1)
		content.Animations = append(content.Animations, candidate)
	}
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return err
	}
	s.asset.Content = encoded
	s.asset.Version++
	return nil
}

type productionDerivationReferenceStore struct {
	mu      sync.Mutex
	counter int
	root    string
	objects map[string]string
}

func (s *productionDerivationReferenceStore) ResolveReference(_ context.Context, reference string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.HasPrefix(reference, "data:") {
		return reference, nil
	}
	value, ok := s.objects[reference]
	if !ok {
		return "", fmt.Errorf("live derivation: reference %q was not persisted", reference)
	}
	return value, nil
}

func (s *productionDerivationReferenceStore) PersistReference(
	ctx context.Context,
	reference string,
) (string, error) {
	s.mu.Lock()
	s.counter++
	key := fmt.Sprintf("production/animation/frame-%04d.png", s.counter)
	s.mu.Unlock()
	if err := s.PersistReferenceAt(ctx, key, reference); err != nil {
		return "", err
	}
	return key, nil
}

func (s *productionDerivationReferenceStore) NewObjectKey(string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return fmt.Sprintf("production/prototype/character-%04d.png", s.counter), nil
}

func (s *productionDerivationReferenceStore) PersistReferenceAt(
	_ context.Context,
	key string,
	reference string,
) error {
	if err := writeLiveDataURL(filepath.Join(s.root, "object_store", filepath.FromSlash(key)), reference); err != nil {
		return err
	}
	s.mu.Lock()
	s.objects[key] = reference
	s.mu.Unlock()
	return nil
}

func (s *productionDerivationReferenceStore) DeleteObjects(_ context.Context, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.objects, key)
		_ = os.Remove(filepath.Join(s.root, "object_store", filepath.FromSlash(key)))
	}
	return nil
}

type productionDerivationImageCall struct {
	Phase          string `json:"phase"`
	Model          string `json:"model"`
	ReferenceCount int    `json:"referenceCount"`
}

type productionDerivationImageService struct {
	mu       sync.Mutex
	delegate imageclient.ImageGenerationService
	root     string
	phase    string
	counter  int
	calls    []productionDerivationImageCall
}

func (s *productionDerivationImageService) setPhase(phase string) {
	s.mu.Lock()
	s.phase = phase
	s.mu.Unlock()
}

func (s *productionDerivationImageService) Generate(
	ctx context.Context,
	request *imageclient.GenerateRequest,
) (*imageclient.GenerateResult, error) {
	s.mu.Lock()
	s.counter++
	phase, number := s.phase, s.counter
	call := productionDerivationImageCall{
		Phase: phase, Model: strings.TrimSpace(request.Model), ReferenceCount: len(request.ReferenceImages),
	}
	s.calls = append(s.calls, call)
	s.mu.Unlock()

	dir := filepath.Join(s.root, phase, fmt.Sprintf("image_model_call_%02d", number))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(request.Prompt), 0o600); err != nil {
		return nil, err
	}
	if err := writeLiveJSON(filepath.Join(dir, "request.json"), map[string]any{
		"model": request.Model, "size": request.Size, "n": request.N,
		"referenceCount": len(request.ReferenceImages), "params": request.Params,
	}); err != nil {
		return nil, err
	}
	for index, reference := range request.ReferenceImages {
		if err := writeLiveDataURL(filepath.Join(dir, fmt.Sprintf("reference_%02d.png", index+1)), reference); err != nil {
			return nil, err
		}
	}

	result, err := s.delegate.Generate(ctx, request)
	if err != nil {
		_ = os.WriteFile(filepath.Join(dir, "error.txt"), []byte(err.Error()), 0o600)
		return nil, err
	}
	for index, generated := range result.Images {
		if err := writeLiveBase64(filepath.Join(dir, fmt.Sprintf("raw_output_%02d.png", index+1)), generated.Base64); err != nil {
			return nil, err
		}
	}
	if err := writeLiveJSON(filepath.Join(dir, "response.json"), map[string]any{
		"model": result.Model, "size": result.Size, "usage": result.Usage,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

type productionDerivationVideoCall struct {
	Phase          string `json:"phase"`
	Model          string `json:"model"`
	ReferenceCount int    `json:"referenceCount"`
	RequestID      string `json:"requestId,omitempty"`
}

type productionDerivationVideoService struct {
	mu       sync.Mutex
	delegate videoclient.VideoGenerationService
	root     string
	phase    string
	counter  int
	calls    []productionDerivationVideoCall
	outputs  map[string]string
}

func (s *productionDerivationVideoService) setPhase(phase string) {
	s.mu.Lock()
	s.phase = phase
	s.mu.Unlock()
}

func (s *productionDerivationVideoService) Generate(
	ctx context.Context,
	request *videoclient.GenerateRequest,
) (*videoclient.GenerateResult, error) {
	s.mu.Lock()
	s.counter++
	phase, number := s.phase, s.counter
	dir := filepath.Join(s.root, phase, fmt.Sprintf("video_model_call_%02d", number))
	s.mu.Unlock()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(request.Prompt), 0o600); err != nil {
		return nil, err
	}
	if err := writeLiveJSON(filepath.Join(dir, "request.json"), map[string]any{
		"model": request.Model, "resolution": request.Resolution, "duration": request.Duration,
		"aspectRatio": request.AspectRatio, "referenceCount": len(request.ReferenceImages),
	}); err != nil {
		return nil, err
	}
	if request.StartImage.Base64 != "" {
		if err := writeLiveBase64(filepath.Join(dir, "start_image.png"), request.StartImage.Base64); err != nil {
			return nil, err
		}
	}
	for index, reference := range request.ReferenceImages {
		if err := writeLiveBase64(filepath.Join(dir, fmt.Sprintf("reference_%02d.png", index+1)), reference.Base64); err != nil {
			return nil, err
		}
	}

	result, err := s.delegate.Generate(ctx, request)
	if err != nil {
		_ = os.WriteFile(filepath.Join(dir, "error.txt"), []byte(err.Error()), 0o600)
		return nil, err
	}
	s.mu.Lock()
	s.calls = append(s.calls, productionDerivationVideoCall{
		Phase: phase, Model: strings.TrimSpace(request.Model),
		ReferenceCount: len(request.ReferenceImages), RequestID: result.RequestID,
	})
	s.outputs[result.VideoURL] = dir
	s.mu.Unlock()
	if err := writeLiveJSON(filepath.Join(dir, "response.json"), map[string]any{"requestId": result.RequestID}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *productionDerivationVideoService) Download(ctx context.Context, videoURL string) ([]byte, error) {
	video, err := s.delegate.Download(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	dir := s.outputs[videoURL]
	s.mu.Unlock()
	if dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "raw_output.mp4"), video, 0o600); err != nil {
			return nil, err
		}
	}
	return video, nil
}

type productionDerivationAnimationService struct {
	mu       sync.Mutex
	delegate AnimationGenerationService
	root     string
	phase    string
}

func (s *productionDerivationAnimationService) setPhase(phase string) {
	s.mu.Lock()
	s.phase = phase
	s.mu.Unlock()
}

func (s *productionDerivationAnimationService) Generate(
	ctx context.Context,
	request *AnimationGenerationRequest,
) (*AnimationGenerationResult, error) {
	s.mu.Lock()
	phase := s.phase
	s.mu.Unlock()
	direction := strings.ToLower(strings.TrimSpace(request.TargetOrientation))
	if direction == "" {
		direction = "source"
	}
	if separator := strings.IndexByte(direction, ' '); separator >= 0 {
		direction = direction[:separator]
	}
	dir := filepath.Join(s.root, phase, "formal_animation_service", direction)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	pathName := "standard_video"
	if request.DerivationSourceImage != "" {
		pathName = "seedance_2_5_dual_reference_video"
		if err := writeLiveDataURL(filepath.Join(dir, "source_animation_sheet.png"), request.DerivationSourceImage); err != nil {
			return nil, err
		}
	}
	if err := writeLiveJSON(filepath.Join(dir, "request.json"), map[string]any{
		"path": pathName, "targetOrientation": request.TargetOrientation,
		"sourceOrientation": request.SourceOrientation, "frameCount": request.FrameCount,
		"columns": request.Columns, "frameWidth": request.FrameWidth,
		"frameHeight": request.FrameHeight, "fps": request.FPS,
	}); err != nil {
		return nil, err
	}
	result, err := s.delegate.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	if result.Spritesheet != "" {
		if err := writeLiveBase64(filepath.Join(dir, "processed_spritesheet.png"), result.Spritesheet); err != nil {
			return nil, err
		}
	}
	if err := writeLiveJSON(filepath.Join(dir, "result.json"), map[string]any{
		"videoRequestId": result.VideoRequestID, "videoAttempts": result.VideoAttempts,
		"loop": result.Loop, "normalization": result.Normalization,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// TestLiveProductionTopDownAnimationDerivation exercises the real production
// image, video, processing, executor, and persistence paths. The in-memory asset
// store only performs the user-approval step between generated candidates.
func TestLiveProductionTopDownAnimationDerivation(t *testing.T) {
	if strings.TrimSpace(os.Getenv(productionDerivationLiveEnv)) != "1" {
		t.Skip("set HOLONIC_PRODUCTION_DERIVATION=1 to run real Top-Down derivation generation")
	}

	configPath := strings.TrimSpace(os.Getenv("HOLONIC_PRODUCTION_CONFIG"))
	if configPath == "" {
		configPath = "../../config/config.yaml"
	}
	var cfg config.Config
	if err := viperx.LoadConfig(configPath, &cfg); err != nil {
		t.Fatalf("load production generation config: %v", err)
	}
	if strings.TrimSpace(cfg.Image.APIKey) == "" || strings.TrimSpace(cfg.Video.APIKey) == "" {
		t.Fatal("production image and video API keys are required")
	}

	artifactRoot := strings.TrimSpace(os.Getenv("HOLONIC_DERIVATION_ARTIFACT_DIR"))
	if artifactRoot == "" {
		artifactRoot = filepath.Join("..", "..", "..", "scratch", "production_top_down_derivation", time.Now().UTC().Format("20060102T150405Z"))
	}
	if absolute, err := filepath.Abs(artifactRoot); err == nil {
		artifactRoot = absolute
	}
	// #nosec G703 -- the operator explicitly selects this opt-in live-test artifact directory.
	if err := os.MkdirAll(artifactRoot, 0o750); err != nil {
		t.Fatalf("create artifact directory: %v", err)
	}
	t.Logf("production derivation artifacts: %s", artifactRoot)

	imageModels := make([]imageclient.ModelConfig, 0, len(cfg.Image.Models))
	for _, model := range cfg.Image.Models {
		imageModels = append(imageModels, imageclient.ModelConfig{
			Name: model.Name, Protocol: model.Protocol, BaseURL: model.BaseURL, APIKey: model.APIKey,
		})
	}
	imageProvider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL: cfg.Image.BaseURL, APIKey: cfg.Image.APIKey,
		DefaultModel: cfg.Image.DefaultModel, FallbackModel: cfg.Image.FallbackModel,
		Provider: cfg.Image.Provider, Models: imageModels,
	})
	imageAudit := &productionDerivationImageService{
		delegate: imageclient.NewImageGenerationService(imageProvider), root: artifactRoot,
	}

	videoModels := make([]videoclient.ModelConfig, 0, len(cfg.Video.Models))
	for _, model := range cfg.Video.Models {
		videoModels = append(videoModels, videoclient.ModelConfig{
			Name: model.Name, Protocol: model.Protocol, BaseURL: model.BaseURL, APIKey: model.APIKey,
		})
	}
	videoProvider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL: cfg.Video.BaseURL, APIKey: cfg.Video.APIKey, Models: videoModels,
		PollInterval: cfg.Video.PollInterval, PollTimeout: cfg.Video.PollTimeout,
		MaxRetries: cfg.Video.MaxRetries, RetryDelay: cfg.Video.RetryDelay,
	})
	videoAudit := &productionDerivationVideoService{
		delegate: videoclient.NewVideoGenerationService(videoProvider), root: artifactRoot,
		outputs: make(map[string]string),
	}
	processor := imageprocessor.NewProcessor()
	references := &productionDerivationReferenceStore{root: artifactRoot, objects: make(map[string]string)}
	assets := &productionDerivationAssetStore{}
	resumePrototype := strings.TrimSpace(os.Getenv("HOLONIC_DERIVATION_RESUME_PROTOTYPE")) == "1"
	resumeSource := strings.TrimSpace(os.Getenv("HOLONIC_DERIVATION_RESUME_SOURCE")) == "1"
	resumeDerivedVideos := strings.TrimSpace(os.Getenv("HOLONIC_DERIVATION_RESUME_DERIVED_VIDEOS")) == "1"
	if resumePrototype {
		if err := resumeProductionDerivationPrototype(artifactRoot, assets, references); err != nil {
			t.Fatalf("resume production prototype: %v", err)
		}
		imageAudit.calls = append(imageAudit.calls, productionDerivationImageCall{
			Phase: "01_character_prototype", Model: cfg.Image.DefaultModel,
		})
	}
	if resumeSource && !resumePrototype {
		t.Fatal("source resume requires prototype resume")
	}
	if resumeDerivedVideos && !resumeSource {
		t.Fatal("derived video resume requires source resume")
	}
	formalAnimationService := NewAnimationGenerationServiceWithDependencies(
		videoAudit,
		processor,
		AnimationGenerationDependencies{ReferenceResolver: references},
	)
	animationAudit := &productionDerivationAnimationService{
		delegate: formalAnimationService, root: artifactRoot,
	}
	executor := NewExecutorWithDependencies(
		imageAudit,
		processor,
		assets,
		ExecutorDependencies{Animations: animationAudit, References: references},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	prototypePhase := "01_character_prototype"
	if !resumePrototype {
		imageAudit.setPhase(prototypePhase)
		prototypePayload := CreateCharacterPrototypePayload{
			AssetName:     "Top-Down Signal Scout",
			CreativeBrief: "A compact friendly field scout in a teal hooded jacket, dark navy trousers, brown boots, and a small amber shoulder radio. Clear full-body silhouette, simple readable limbs, no weapon, no text, production-quality crisp pixel art.",
			Dimensions:    assetdomain.Size{Width: 64, Height: 64},
			Perspective:   string(assetdomain.PerspectiveTopDown), ProjectID: 1101,
		}
		executeLiveTask(t, ctx, executor, GenerateCharacterProtoType, prototypePayload, artifactRoot, prototypePhase)
	}
	prototypeAsset, err := assets.GetDetail(ctx, 7001)
	if err != nil || prototypeAsset.ID == 0 {
		t.Fatalf("load generated character: asset=%+v err=%v", prototypeAsset, err)
	}
	if err := exportLivePrototype(ctx, references, prototypeAsset, filepath.Join(artifactRoot, prototypePhase, "accepted_prototype")); err != nil {
		t.Fatalf("export accepted prototype: %v", err)
	}

	sourcePhase := "02_source_left_animation"
	if resumeSource {
		if err := resumeProductionDerivationSource(artifactRoot, sourcePhase, assets, videoAudit); err != nil {
			t.Fatalf("resume production source animation: %v", err)
		}
	} else {
		videoAudit.setPhase(sourcePhase)
		animationAudit.setPhase(sourcePhase)
		sourcePayload := CreateAnimationPayload{
			AnimationName: "small_wave", ProjectID: 1101, AssetID: prototypeAsset.ID,
			Direction:     AnimationDirectionLeft,
			CreativeBrief: "A small friendly two-beat wave with one hand: begin in a calm standing pose, raise one hand beside the head, make two subtle wrist waves, lower the hand, and return exactly to the starting pose. Keep both feet planted and the body stationary.",
			Style:         "crisp production-quality pixel art matching the supplied character exactly",
			FrameCount:    8, FrameWidth: 96, FrameHeight: 96, FPS: 10, Resolution: "720p", Duration: 5,
		}
		sourceResult := executeLiveTask(t, ctx, executor, GenerateAnimation, sourcePayload, artifactRoot, sourcePhase)
		sourceCandidates := liveExecutionAnimations(t, sourceResult)
		if len(sourceCandidates) != 1 {
			t.Fatalf("source animation candidates = %d, want 1", len(sourceCandidates))
		}
		if err := assets.applyAnimations(sourceCandidates); err != nil {
			t.Fatalf("accept source animation: %v", err)
		}
	}
	sourceAsset, _ := assets.GetDetail(ctx, prototypeAsset.ID)
	sourceContent, err := sourceAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode source asset: %v", err)
	}
	sourceAnimation := sourceContent.Animations[0]
	if sourceAnimation.Generation == nil || sourceAnimation.Generation.Direction != AnimationDirectionLeft {
		t.Fatalf("unexpected source animation: %+v", sourceAnimation)
	}
	if err := exportLiveAnimation(ctx, references, sourceAnimation, filepath.Join(artifactRoot, sourcePhase, "accepted_animation")); err != nil {
		t.Fatalf("export source animation: %v", err)
	}

	derivationPhase := "03_derive_remaining_parallel"
	if resumeDerivedVideos {
		if err := resumeProductionDerivationVideoDirections(
			artifactRoot,
			derivationPhase,
			assets,
			videoAudit,
		); err != nil {
			t.Fatalf("resume production derived video directions: %v", err)
		}
		derivationPhase = "04_rederive_right_after_matte_fix"
	}
	imageAudit.setPhase(derivationPhase)
	videoAudit.setPhase(derivationPhase)
	animationAudit.setPhase(derivationPhase)
	targetDirections := []string{
		AnimationDirectionFront,
		AnimationDirectionRight,
		AnimationDirectionBack,
	}
	if resumeDerivedVideos {
		targetDirections = []string{AnimationDirectionRight}
	}
	derivationResult := executeLiveTask(t, ctx, executor, DeriveAnimation, DeriveAnimationPayload{
		AssetID: prototypeAsset.ID, ProjectID: 1101, SourceAnimationID: sourceAnimation.ID,
		TargetDirections: targetDirections,
	}, artifactRoot, derivationPhase)
	derivedCandidates := liveExecutionAnimations(t, derivationResult)
	if len(derivedCandidates) != len(targetDirections) {
		t.Fatalf("parallel derivation returned %d candidates, want %d", len(derivedCandidates), len(targetDirections))
	}
	for index, direction := range targetDirections {
		candidate := derivedCandidates[index]
		if candidate.Generation == nil || candidate.Generation.Direction != direction {
			t.Fatalf("unexpected parallel derivation %d: %+v", index, candidate)
		}
	}
	if err := assets.applyAnimations(derivedCandidates); err != nil {
		t.Fatalf("accept parallel derivations: %v", err)
	}
	for index, candidate := range derivedCandidates {
		candidate.ID = uint(len(sourceContent.Animations) + 1)
		sourceContent.Animations = append(sourceContent.Animations, candidate)
		direction := candidate.Generation.Direction
		exportDirectory := filepath.Join(
			artifactRoot,
			derivationPhase,
			"accepted_animations",
			fmt.Sprintf("%02d_%s", index+1, direction),
		)
		if resumeDerivedVideos {
			exportDirectory = filepath.Join(artifactRoot, derivationPhase, "accepted_animation")
		}
		if err := exportLiveAnimation(
			ctx,
			references,
			candidate,
			exportDirectory,
		); err != nil {
			t.Fatalf("export %s animation: %v", direction, err)
		}
	}

	finalAsset, _ := assets.GetDetail(ctx, prototypeAsset.ID)
	finalContent, err := finalAsset.DecodeContent()
	if err != nil {
		t.Fatalf("decode final asset: %v", err)
	}
	if len(finalContent.Animations) != 4 {
		t.Fatalf("final animation count = %d, want 4", len(finalContent.Animations))
	}
	byDirection := make(map[string]assetdomain.Animation, len(finalContent.Animations))
	for _, animation := range finalContent.Animations {
		if animation.Generation != nil {
			byDirection[animation.Generation.Direction] = animation
		}
	}
	for _, direction := range animationDirectionLayouts[4] {
		if _, ok := byDirection[direction]; !ok {
			t.Fatalf("final asset is missing %s animation", direction)
		}
	}
	if err := exportLiveDirectionOverview(ctx, references, byDirection, filepath.Join(artifactRoot, "06_final", "top_down_four_direction_sheet.png")); err != nil {
		t.Fatalf("export final direction overview: %v", err)
	}

	manifest := map[string]any{
		"assetId": finalAsset.ID, "version": finalAsset.Version,
		"perspective": finalAsset.Perspective, "directionOrder": animationDirectionLayouts[4],
		"sourceDirection": AnimationDirectionLeft,
		"paths": []map[string]string{
			{"direction": "left", "kind": "standard video", "model": "bytedance/seedance-2.0"},
			{"direction": "front", "kind": "parallel dual-reference video", "model": animationDerivationVideoModel},
			{"direction": "right", "kind": "composite image edit from existing left", "model": animationDerivationImageModel},
			{"direction": "back", "kind": "parallel dual-reference video", "model": animationDerivationVideoModel},
		},
		"imageCalls": imageAudit.calls, "videoCalls": videoAudit.calls,
		"parallelDerivation": map[string]any{
			"phase": "03_derive_remaining_parallel", "targetDirections": []string{"front", "right", "back"},
			"request": "one derive_animation task; all target generators start concurrently from the initial asset snapshot",
		},
	}
	if err := writeLiveJSON(filepath.Join(artifactRoot, "manifest.json"), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeLiveJSON(filepath.Join(artifactRoot, "final_asset.json"), finalAsset); err != nil {
		t.Fatalf("write final asset: %v", err)
	}
	t.Logf("completed production Top-Down derivation with directions: %s", strings.Join(animationDirectionLayouts[4], ", "))
	for _, call := range videoAudit.calls {
		t.Logf("video path phase=%s model=%q references=%d request_id=%s", call.Phase, call.Model, call.ReferenceCount, call.RequestID)
	}
	for _, call := range imageAudit.calls {
		t.Logf("image path phase=%s model=%q references=%d", call.Phase, call.Model, call.ReferenceCount)
	}
}

func resumeProductionDerivationPrototype(
	artifactRoot string,
	assets *productionDerivationAssetStore,
	references *productionDerivationReferenceStore,
) error {
	objectRoot := filepath.Join(artifactRoot, "object_store")
	// #nosec G703 -- objectRoot is an explicit live-test checkpoint directory.
	if err := filepath.WalkDir(objectRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path) // #nosec G304,G122 -- path belongs to the explicit live-test checkpoint.
		if err != nil {
			return err
		}
		key, err := filepath.Rel(objectRoot, path)
		if err != nil {
			return err
		}
		references.objects[filepath.ToSlash(key)] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(content)
		return nil
	}); err != nil {
		return err
	}
	references.counter = 2000

	prototype := make(assetdomain.Prototype, 4)
	for index := range prototype {
		key := fmt.Sprintf("production/prototype/character-0001-%d.png", index)
		if _, ok := references.objects[key]; !ok {
			return fmt.Errorf("prototype checkpoint %q is missing", key)
		}
		prototype[index] = assetdomain.ImageResource{ID: uint(index + 1), URL: &key}
	}
	content := assetdomain.NewAssetContent(assetdomain.AssetTypeCharacter)
	content.DirectionCount = 4
	content.Prototype = &prototype
	encoded, err := assetdomain.EncodeContent(content)
	if err != nil {
		return err
	}
	assets.asset = assetdomain.Asset{
		ID:          7001,
		Name:        "Top-Down Signal Scout",
		ProjectID:   1101,
		Type:        assetdomain.AssetTypeCharacter,
		Description: "A compact friendly field scout in a teal hooded jacket, dark navy trousers, brown boots, and a small amber shoulder radio. Clear full-body silhouette, simple readable limbs, no weapon, no text, production-quality crisp pixel art.",
		Perspective: assetdomain.PerspectiveTopDown,
		Dimensions:  json.RawMessage(`{"width":64,"height":64}`),
		Content:     encoded,
		Version:     1,
	}
	return nil
}

func resumeProductionDerivationSource(
	artifactRoot string,
	sourcePhase string,
	assets *productionDerivationAssetStore,
	videos *productionDerivationVideoService,
) error {
	raw, err := os.ReadFile( // #nosec G304,G703 -- the operator explicitly selects this live-test checkpoint.
		filepath.Join(artifactRoot, sourcePhase, "task_result.json"),
	)
	if err != nil {
		return err
	}
	var execution ExecutionResult
	if err := json.Unmarshal(raw, &execution); err != nil {
		return err
	}
	var content struct {
		Animations []assetdomain.Animation `json:"animations"`
	}
	if err := json.Unmarshal(execution.Content, &content); err != nil {
		return err
	}
	if len(content.Animations) != 1 || content.Animations[0].Generation == nil ||
		content.Animations[0].Generation.Direction != AnimationDirectionLeft {
		return fmt.Errorf("left source animation checkpoint is invalid")
	}
	if err := assets.applyAnimations(content.Animations); err != nil {
		return err
	}
	var response struct {
		RequestID string `json:"requestId"`
	}
	responseRaw, err := os.ReadFile( // #nosec G304,G703 -- the operator explicitly selects this live-test checkpoint.
		filepath.Join(artifactRoot, sourcePhase, "video_model_call_01", "response.json"),
	)
	if err == nil {
		_ = json.Unmarshal(responseRaw, &response)
	}
	videos.calls = append(videos.calls, productionDerivationVideoCall{
		Phase: sourcePhase, Model: "bytedance/seedance-2.0", ReferenceCount: 0, RequestID: response.RequestID,
	})
	return nil
}

func resumeProductionDerivationVideoDirections(
	artifactRoot string,
	derivationPhase string,
	assets *productionDerivationAssetStore,
	videos *productionDerivationVideoService,
) error {
	raw, err := os.ReadFile( // #nosec G304,G703 -- the operator explicitly selects this live-test checkpoint.
		filepath.Join(artifactRoot, derivationPhase, "task_result.json"),
	)
	if err != nil {
		return err
	}
	var execution ExecutionResult
	if err := json.Unmarshal(raw, &execution); err != nil {
		return err
	}
	var content struct {
		Animations []assetdomain.Animation `json:"animations"`
	}
	if err := json.Unmarshal(execution.Content, &content); err != nil {
		return err
	}
	restored := make([]assetdomain.Animation, 0, 2)
	for _, animation := range content.Animations {
		if animation.Generation == nil {
			continue
		}
		direction := animation.Generation.Direction
		if direction == AnimationDirectionFront || direction == AnimationDirectionBack {
			restored = append(restored, animation)
		}
	}
	if len(restored) != 2 {
		return fmt.Errorf("derived direction checkpoint contains %d restorable video animations; want 2", len(restored))
	}
	if err := assets.applyAnimations(restored); err != nil {
		return err
	}
	for _, direction := range []string{AnimationDirectionFront, AnimationDirectionBack} {
		var result struct {
			VideoRequestID string `json:"videoRequestId"`
		}
		resultRaw, readErr := os.ReadFile( // #nosec G304,G703 -- the operator explicitly selects this checkpoint.
			filepath.Join(
				artifactRoot,
				derivationPhase,
				"formal_animation_service",
				direction,
				"result.json",
			),
		)
		if readErr == nil {
			_ = json.Unmarshal(resultRaw, &result)
		}
		videos.calls = append(videos.calls, productionDerivationVideoCall{
			Phase: derivationPhase, Model: animationDerivationVideoModel,
			ReferenceCount: 2, RequestID: result.VideoRequestID,
		})
	}
	return nil
}

func executeLiveTask(
	t *testing.T,
	ctx context.Context,
	executor Executor,
	taskType TaskType,
	payload any,
	artifactRoot string,
	phase string,
) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode %s payload: %v", taskType, err)
	}
	// #nosec G703 -- the operator explicitly selects this opt-in live-test artifact directory.
	if err := os.MkdirAll(filepath.Join(artifactRoot, phase), 0o750); err != nil {
		t.Fatalf("create %s artifact directory: %v", phase, err)
	}
	if err := writeLiveJSON(filepath.Join(artifactRoot, phase, "task_request.json"), payload); err != nil {
		t.Fatalf("write %s request: %v", taskType, err)
	}
	t.Logf("starting %s (%s)", taskType, phase)
	started := time.Now()
	result, err := executor.Generate(ctx, taskType, encoded)
	if err != nil {
		t.Fatalf("execute %s: %v", taskType, err)
	}
	if err := os.WriteFile( // #nosec G703 -- the operator explicitly selects this live-test artifact directory.
		filepath.Join(artifactRoot, phase, "task_result.json"), result, 0o600,
	); err != nil {
		t.Fatalf("write %s result: %v", taskType, err)
	}
	t.Logf("completed %s (%s) in %s", taskType, phase, time.Since(started).Round(time.Second))
	return result
}

func liveExecutionAnimations(t *testing.T, raw json.RawMessage) []assetdomain.Animation {
	t.Helper()
	var result ExecutionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode execution result: %v", err)
	}
	var content struct {
		Animations []assetdomain.Animation `json:"animations"`
	}
	if err := json.Unmarshal(result.Content, &content); err != nil {
		t.Fatalf("decode animation candidates: %v", err)
	}
	return content.Animations
}

func exportLivePrototype(
	ctx context.Context,
	references ReferenceStore,
	asset assetdomain.Asset,
	dir string,
) error {
	content, err := asset.DecodeContent()
	if err != nil {
		return err
	}
	if content.Prototype == nil || len(*content.Prototype) != 4 {
		return fmt.Errorf("live derivation: expected four prototype directions")
	}
	for index, direction := range animationDirectionLayouts[4] {
		resource := (*content.Prototype)[index]
		if resource.URL == nil {
			return fmt.Errorf("live derivation: %s prototype URL is required", direction)
		}
		if err := exportLiveReference(ctx, references, *resource.URL, filepath.Join(dir, "processed", direction+".png")); err != nil {
			return err
		}
		if err := exportLiveReference(ctx, references, animationUnprocessedImageURL(*resource.URL), filepath.Join(dir, "unprocessed", direction+".png")); err != nil {
			return err
		}
	}
	return nil
}

func exportLiveAnimation(
	ctx context.Context,
	references ReferenceStore,
	animation assetdomain.Animation,
	dir string,
) error {
	if animation.Generation == nil {
		return fmt.Errorf("live derivation: animation generation metadata is required")
	}
	processed := make([]image.Image, len(animation.Frames))
	for index, frame := range animation.Frames {
		if frame.URL == nil {
			return fmt.Errorf("live derivation: frame %d URL is required", index+1)
		}
		processedReference, err := references.ResolveReference(ctx, *frame.URL)
		if err != nil {
			return err
		}
		processedPath := filepath.Join(dir, "processed_frames", fmt.Sprintf("frame_%02d.png", index+1))
		if err := writeLiveDataURL(processedPath, processedReference); err != nil {
			return err
		}
		encoded, err := liveDataURLBase64(processedReference)
		if err != nil {
			return err
		}
		decoded, err := imageprocessor.DecodeBase64Image(encoded)
		if err != nil {
			return err
		}
		processed[index] = decoded

		rawKey := animationUnprocessedImageURL(*frame.URL)
		if rawReference, resolveErr := references.ResolveReference(ctx, rawKey); resolveErr == nil {
			if err := writeLiveDataURL(filepath.Join(dir, "unprocessed_frames", fmt.Sprintf("frame_%02d.png", index+1)), rawReference); err != nil {
				return err
			}
		}
	}
	sheet, err := packTransparentAnimationFrames(processed, animation.Generation.Columns)
	if err != nil {
		return err
	}
	return writeLivePNG(filepath.Join(dir, "spritesheet.png"), sheet)
}

func exportLiveDirectionOverview(
	ctx context.Context,
	references ReferenceStore,
	animations map[string]assetdomain.Animation,
	path string,
) error {
	sheets := make([]image.Image, 0, 4)
	for _, direction := range animationDirectionLayouts[4] {
		animation := animations[direction]
		if animation.Generation == nil {
			return fmt.Errorf("live derivation: %s generation metadata is required", direction)
		}
		frames := make([]image.Image, len(animation.Frames))
		for index, frame := range animation.Frames {
			if frame.URL == nil {
				return fmt.Errorf("live derivation: %s frame %d URL is required", direction, index+1)
			}
			reference, err := references.ResolveReference(ctx, *frame.URL)
			if err != nil {
				return err
			}
			encoded, err := liveDataURLBase64(reference)
			if err != nil {
				return err
			}
			frames[index], err = imageprocessor.DecodeBase64Image(encoded)
			if err != nil {
				return err
			}
		}
		sheet, err := packTransparentAnimationFrames(frames, animation.Generation.Columns)
		if err != nil {
			return err
		}
		sheets = append(sheets, sheet)
	}
	overview, err := packTransparentAnimationFrames(sheets, 2)
	if err != nil {
		return err
	}
	return writeLivePNG(path, overview)
}

func exportLiveReference(ctx context.Context, references ReferenceStore, reference string, path string) error {
	resolved, err := references.ResolveReference(ctx, reference)
	if err != nil {
		return err
	}
	return writeLiveDataURL(path, resolved)
}

func writeLiveJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	// #nosec G703 -- path is constructed beneath the operator-selected live-test artifact directory.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o600) // #nosec G703 -- path is limited to live-test artifacts.
}

func writeLiveDataURL(path string, value string) error {
	encoded, err := liveDataURLBase64(value)
	if err != nil {
		return err
	}
	return writeLiveBase64(path, encoded)
}

func liveDataURLBase64(value string) (string, error) {
	value = strings.TrimSpace(value)
	separator := strings.IndexByte(value, ',')
	if !strings.HasPrefix(value, "data:") || separator < 0 {
		return "", fmt.Errorf("live derivation: expected a data URL")
	}
	return value[separator+1:], nil
}

func writeLiveBase64(path string, value string) error {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	// #nosec G703 -- path is constructed beneath the operator-selected live-test artifact directory.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, decoded, 0o600) // #nosec G703 -- path is limited to live-test artifacts.
}

func writeLivePNG(path string, source image.Image) error {
	// #nosec G703 -- path is constructed beneath the operator-selected live-test artifact directory.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	file, err := os.Create(path) // #nosec G304,G703 -- path is controlled by this opt-in live test.
	if err != nil {
		return err
	}
	if err := png.Encode(file, source); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

var _ AssetWriter = (*productionDerivationAssetStore)(nil)
var _ ReferenceStore = (*productionDerivationReferenceStore)(nil)
var _ imageclient.ImageGenerationService = (*productionDerivationImageService)(nil)
var _ videoclient.VideoGenerationService = (*productionDerivationVideoService)(nil)
var _ AnimationGenerationService = (*productionDerivationAnimationService)(nil)
