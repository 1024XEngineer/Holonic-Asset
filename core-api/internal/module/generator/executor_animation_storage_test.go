package generator

import (
	"context"
	"errors"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

type animationFrameStorageStub struct {
	persisted   []string
	persistedAt []struct {
		key       string
		reference string
	}
	persistErr   error
	persistAtErr error
	keys         []string
}

func (s *animationFrameStorageStub) ResolveReference(context.Context, string) (string, error) {
	return "", errors.New("unexpected resolve")
}

func (s *animationFrameStorageStub) PersistReference(_ context.Context, reference string) (string, error) {
	s.persisted = append(s.persisted, reference)
	if s.persistErr != nil {
		return "", s.persistErr
	}
	if len(s.keys) == 0 {
		return "uploads/frame.png", nil
	}
	key := s.keys[0]
	s.keys = s.keys[1:]
	return key, nil
}

func (s *animationFrameStorageStub) NewObjectKey(string) (string, error) {
	return "", errors.New("unexpected key allocation")
}

func (s *animationFrameStorageStub) DeleteObjects(context.Context, []string) error {
	return nil
}

func (s *animationFrameStorageStub) PersistReferenceAt(_ context.Context, key, reference string) error {
	s.persistedAt = append(s.persistedAt, struct {
		key       string
		reference string
	}{key: key, reference: reference})
	return s.persistAtErr
}

func TestPersistAnimationFramesValidatesInputs(t *testing.T) {
	result := &AnimationGenerationResult{
		Frames:    []imageprocessor.ImageRegion{{ImageBase64: "processed"}},
		RawFrames: []imageprocessor.ImageRegion{{ImageBase64: "raw"}},
	}
	if _, err := (&executor{}).persistAnimationFrames(context.Background(), result); !errors.Is(err, ErrAnimationReferenceStoreRequired) {
		t.Fatalf("expected reference store error, got %v", err)
	}
	store := &animationFrameStorageStub{}
	executor := &executor{references: store}
	if _, err := executor.persistAnimationFrames(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "result is required") {
		t.Fatalf("expected nil result error, got %v", err)
	}
	if _, err := executor.persistAnimationFrames(context.Background(), &AnimationGenerationResult{
		Frames:    []imageprocessor.ImageRegion{{ImageBase64: "processed"}},
		RawFrames: []imageprocessor.ImageRegion{{ImageBase64: "raw"}, {ImageBase64: "raw-2"}},
	}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected raw count mismatch, got %v", err)
	}
}

func TestPersistAnimationFramesStoresProcessedAndRawFrames(t *testing.T) {
	store := &animationFrameStorageStub{keys: []string{"uploads/one.png", "uploads/two.png"}}
	executor := &executor{references: store}
	frames, err := executor.persistAnimationFrames(context.Background(), &AnimationGenerationResult{
		Frames: []imageprocessor.ImageRegion{
			{ImageBase64: "processed-1", MIMEType: ""},
			{ImageBase64: "processed-2", MIMEType: "image/webp"},
		},
		RawFrames: []imageprocessor.ImageRegion{
			{ImageBase64: "raw-1", MIMEType: ""},
			{ImageBase64: "raw-2", MIMEType: "image/jpeg"},
		},
		FrameDurationMS: 83,
	})
	if err != nil {
		t.Fatalf("persist animation frames: %v", err)
	}
	if len(frames) != 2 || frames[0].ID != 1 || frames[1].ID != 2 || frames[0].Duration != 83 || frames[1].Duration != 83 {
		t.Fatalf("unexpected persisted frames: %+v", frames)
	}
	if frames[0].URL == nil || *frames[0].URL != "uploads/one.png" || frames[1].URL == nil || *frames[1].URL != "uploads/two.png" {
		t.Fatalf("unexpected frame URLs: %+v", frames)
	}
	if len(store.persisted) != 2 || store.persisted[0] != "data:image/png;base64,processed-1" || store.persisted[1] != "data:image/webp;base64,processed-2" {
		t.Fatalf("unexpected processed references: %v", store.persisted)
	}
	if len(store.persistedAt) != 2 || store.persistedAt[0].key != "uploads/one-unprocessed.png" || store.persistedAt[0].reference != "data:image/png;base64,raw-1" || store.persistedAt[1].key != "uploads/two-unprocessed.png" || store.persistedAt[1].reference != "data:image/jpeg;base64,raw-2" {
		t.Fatalf("unexpected raw references: %+v", store.persistedAt)
	}
}

func TestPersistAnimationFramesRejectsStorageFailures(t *testing.T) {
	tests := []struct {
		name   string
		store  *animationFrameStorageStub
		result *AnimationGenerationResult
		want   string
	}{
		{name: "persist processed", store: &animationFrameStorageStub{persistErr: errors.New("upload failed")}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}, want: "persist animation frame 1"},
		{name: "empty key", store: &animationFrameStorageStub{keys: []string{""}}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}, want: "empty object key"},
		{name: "data key", store: &animationFrameStorageStub{keys: []string{"data:image/png;base64,frame"}}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}, want: "non-object-key"},
		{name: "http key", store: &animationFrameStorageStub{keys: []string{"https://cdn.example/frame.png"}}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}}, want: "non-object-key"},
		{name: "empty raw", store: &animationFrameStorageStub{keys: []string{"uploads/frame.png"}}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}, RawFrames: []imageprocessor.ImageRegion{{}}}, want: "raw animation frame 1 is empty"},
		{name: "persist raw", store: &animationFrameStorageStub{keys: []string{"uploads/frame.png"}, persistAtErr: errors.New("raw upload failed")}, result: &AnimationGenerationResult{Frames: []imageprocessor.ImageRegion{{ImageBase64: "frame"}}, RawFrames: []imageprocessor.ImageRegion{{ImageBase64: "raw"}}}, want: "persist raw animation frame 1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&executor{references: test.store}).persistAnimationFrames(context.Background(), test.result)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestAnimationReferenceErrors(t *testing.T) {
	t.Run("unsupported asset type", func(t *testing.T) {
		asset := assetdomain.Asset{Type: assetdomain.AssetTypeScenery}
		_, _, err := animationReference(asset, "front")
		if err == nil {
			t.Fatal("expected error for scenery asset")
		}
	})

	t.Run("corrupt content json", func(t *testing.T) {
		asset := assetdomain.Asset{
			Type:    assetdomain.AssetTypeCharacter,
			Content: []byte(`{invalid`),
		}
		_, _, err := animationReference(asset, "front")
		if err == nil {
			t.Fatal("expected error for corrupt content")
		}
	})

	t.Run("missing prototype url", func(t *testing.T) {
		asset := assetdomain.Asset{
			Type: assetdomain.AssetTypeCharacter,
			Content: []byte(`{
				"direction_count": 4,
				"prototype": [{"url": ""}]
			}`),
		}
		_, _, err := animationReference(asset, "front")
		if err == nil {
			t.Fatal("expected error for empty prototype url")
		}
	})

	t.Run("invalid url path fallback", func(t *testing.T) {
		got := animationUnprocessedImageURL("http://::invalid-url")
		if !strings.HasSuffix(got, "-unprocessed") {
			t.Fatalf("expected -unprocessed suffix, got %q", got)
		}
	})
}

