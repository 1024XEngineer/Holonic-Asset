package generator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

type mockTileSetReferenceStore struct {
	mu                  sync.Mutex
	newKeyFunc          func(mimeType string) (string, error)
	persistAtFunc       func(ctx context.Context, key, dataURL string) error
	deleteFunc          func(ctx context.Context, keys []string) error
	deletedKeys         []string
	persistAtCallsCount int
}

func (m *mockTileSetReferenceStore) ResolveReference(_ context.Context, ref string) (string, error) {
	return ref, nil
}

func (m *mockTileSetReferenceStore) PersistReference(_ context.Context, ref string) (string, error) {
	return ref, nil
}

func (m *mockTileSetReferenceStore) NewObjectKey(mimeType string) (string, error) {
	if m.newKeyFunc != nil {
		return m.newKeyFunc(mimeType)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistAtCallsCount++
	return fmt.Sprintf("obj-key-%d", m.persistAtCallsCount), nil
}

func (m *mockTileSetReferenceStore) PersistReferenceAt(ctx context.Context, key, dataURL string) error {
	if m.persistAtFunc != nil {
		return m.persistAtFunc(ctx, key, dataURL)
	}
	return nil
}

func (m *mockTileSetReferenceStore) DeleteObjects(_ context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedKeys = append(m.deletedKeys, keys...)
	if m.deleteFunc != nil {
		return m.deleteFunc(context.Background(), keys)
	}
	return nil
}

var _ ReferenceStore = (*mockTileSetReferenceStore)(nil)

func TestBuildTileSetContentItems(t *testing.T) {
	items := []processedTileSetItem{
		{Name: "Item 0"},
		{Name: "Item 1"},
	}
	uploads := []tileSetTileUpload{
		{itemIndex: 0, tileIndex: 0, position: TileSetCoordinate{0, 0}, objectKey: "k0"},
		{itemIndex: 0, tileIndex: 1, position: TileSetCoordinate{0, 1}, objectKey: "k1"},
		{itemIndex: 1, tileIndex: 0, position: TileSetCoordinate{1, 0}, objectKey: "k2"},
	}

	contentItems := buildTileSetContentItems(items, uploads)
	if len(contentItems) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(contentItems))
	}
	if contentItems[0].Name != "Item 0" || len(contentItems[0].Tiles) != 2 {
		t.Fatalf("unexpected content item 0: %+v", contentItems[0])
	}
	if contentItems[1].Name != "Item 1" || len(contentItems[1].Tiles) != 1 {
		t.Fatalf("unexpected content item 1: %+v", contentItems[1])
	}
}

func TestBuildTileSetUploads(t *testing.T) {
	items := []processedTileSetItem{
		{
			Name: "Wall",
			Tiles: []imageprocessor.ImageRegion{
				{ImageBase64: "tile0", MIMEType: "image/png"},
			},
			RawImageBase64: "rawB64",
			RawMediaType:   "image/png",
		},
	}
	layout := []tileSetPlacement{
		{ItemIndex: 0, Positions: []TileSetCoordinate{{0, 0}}},
	}

	t.Run("mismatched layout count", func(t *testing.T) {
		store := &mockTileSetReferenceStore{}
		_, err := buildTileSetUploads(store, items, nil)
		if err == nil {
			t.Fatal("expected error for mismatched layout count")
		}
	})

	t.Run("mismatched item index in layout", func(t *testing.T) {
		store := &mockTileSetReferenceStore{}
		badLayout := []tileSetPlacement{
			{ItemIndex: 1, Positions: []TileSetCoordinate{{0, 0}}},
		}
		_, err := buildTileSetUploads(store, items, badLayout)
		if err == nil {
			t.Fatal("expected error for bad item index")
		}
	})

	t.Run("new object key failure", func(t *testing.T) {
		store := &mockTileSetReferenceStore{
			newKeyFunc: func(_ string) (string, error) {
				return "", errors.New("key error")
			},
		}
		_, err := buildTileSetUploads(store, items, layout)
		if err == nil {
			t.Fatal("expected error for new object key failure")
		}
	})

	t.Run("duplicate object key error", func(t *testing.T) {
		store := &mockTileSetReferenceStore{
			newKeyFunc: func(_ string) (string, error) {
				return "dup-key", nil
			},
		}
		multiItems := []processedTileSetItem{
			{
				Name: "Wall",
				Tiles: []imageprocessor.ImageRegion{
					{ImageBase64: "tile0", MIMEType: "image/png"},
					{ImageBase64: "tile1", MIMEType: "image/png"},
				},
			},
		}
		multiLayout := []tileSetPlacement{
			{ItemIndex: 0, Positions: []TileSetCoordinate{{0, 0}, {0, 1}}},
		}
		_, err := buildTileSetUploads(store, multiItems, multiLayout)
		if err == nil {
			t.Fatal("expected error for duplicate key")
		}
	})

	t.Run("success", func(t *testing.T) {
		store := &mockTileSetReferenceStore{}
		uploads, err := buildTileSetUploads(store, items, layout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(uploads) != 1 {
			t.Fatalf("expected 1 upload, got %d", len(uploads))
		}
		if uploads[0].rawImageBase64 != "rawB64" {
			t.Fatalf("expected raw image b64 preserved: %+v", uploads[0])
		}
	})

	t.Run("defaults raw media type to image/png when empty", func(t *testing.T) {
		store := &mockTileSetReferenceStore{}
		emptyMIMEItems := []processedTileSetItem{
			{
				Name: "Wall",
				Tiles: []imageprocessor.ImageRegion{
					{ImageBase64: "tile0", MIMEType: "image/png"},
				},
				RawImageBase64: "rawB64",
				RawMediaType:   "",
			},
		}
		uploads, err := buildTileSetUploads(store, emptyMIMEItems, layout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(uploads) != 1 || uploads[0].rawMediaType != "image/png" {
			t.Fatalf("expected default rawMediaType image/png, got %+v", uploads)
		}
	})
}

func TestPersistTileSetUploads(t *testing.T) {
	uploads := []tileSetTileUpload{
		{
			itemIndex:      0,
			tileIndex:      0,
			position:       TileSetCoordinate{0, 0},
			region:         imageprocessor.ImageRegion{ImageBase64: "img0", MIMEType: "image/png"},
			objectKey:      "key-0.png",
			rawImageBase64: "raw-0",
			rawMediaType:   "image/png",
		},
	}

	t.Run("success", func(t *testing.T) {
		store := &mockTileSetReferenceStore{}
		exec := &executor{references: store}
		keys, err := exec.persistTileSetUploads(context.Background(), uploads)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 2 { // key-0.png and key-0-unprocessed.png
			t.Fatalf("expected 2 keys, got %d (%v)", len(keys), keys)
		}
	})

	t.Run("defaults region and raw mime types to image/png when empty", func(t *testing.T) {
		var persistedDataURLs []string
		store := &mockTileSetReferenceStore{
			persistAtFunc: func(_ context.Context, key, dataURL string) error {
				persistedDataURLs = append(persistedDataURLs, dataURL)
				return nil
			},
		}
		emptyMIMEUploads := []tileSetTileUpload{
			{
				itemIndex:      0,
				tileIndex:      0,
				position:       TileSetCoordinate{0, 0},
				region:         imageprocessor.ImageRegion{ImageBase64: "img0", MIMEType: ""},
				objectKey:      "key-0.png",
				rawImageBase64: "raw-0",
				rawMediaType:   "",
			},
		}
		exec := &executor{references: store}
		keys, err := exec.persistTileSetUploads(context.Background(), emptyMIMEUploads)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(keys))
		}
		if len(persistedDataURLs) != 2 ||
			persistedDataURLs[0] != "data:image/png;base64,img0" ||
			persistedDataURLs[1] != "data:image/png;base64,raw-0" {
			t.Fatalf("unexpected data URLs: %+v", persistedDataURLs)
		}
	})

	t.Run("upload tile failure triggers cleanup", func(t *testing.T) {
		store := &mockTileSetReferenceStore{
			persistAtFunc: func(_ context.Context, key, _ string) error {
				return errors.New("persist failed")
			},
		}
		exec := &executor{references: store}
		_, err := exec.persistTileSetUploads(context.Background(), uploads)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("upload unprocessed failure triggers cleanup", func(t *testing.T) {
		multiUploads := []tileSetTileUpload{
			{
				itemIndex: 0,
				tileIndex: 0,
				position:  TileSetCoordinate{0, 0},
				region:    imageprocessor.ImageRegion{ImageBase64: "img0", MIMEType: "image/png"},
				objectKey: "key-0.png",
			},
			{
				itemIndex:      1,
				tileIndex:      0,
				position:       TileSetCoordinate{1, 0},
				region:         imageprocessor.ImageRegion{ImageBase64: "img1", MIMEType: "image/png"},
				objectKey:      "key-1.png",
				rawImageBase64: "raw-1",
				rawMediaType:   "image/png",
			},
		}
		store := &mockTileSetReferenceStore{
			persistAtFunc: func(_ context.Context, key, _ string) error {
				if key == "key-1-unprocessed.png" {
					return errors.New("unprocessed upload failed")
				}
				return nil
			},
		}
		exec := &executor{references: store}
		_, err := exec.persistTileSetUploads(context.Background(), multiUploads)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		store := &mockTileSetReferenceStore{}
		exec := &executor{references: store}
		_, err := exec.persistTileSetUploads(ctx, uploads)
		if err == nil {
			t.Fatal("expected error for canceled context")
		}
	})
}
