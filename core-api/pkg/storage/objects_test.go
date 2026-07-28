package storage_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/pkg/storage"
)

func TestNewQiniuStorageReturnsPlaceholderStore(t *testing.T) {
	store, err := storage.NewQiniuStorage(storage.QiniuConfig{})
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	if store == nil {
		t.Fatal("expected a non-nil placeholder store")
	}
}

func TestCreateUploadTargetReturnsPlaceholderTarget(t *testing.T) {
	store := newTestStorage(t)
	target, err := store.CreateUploadTarget(context.Background(), storage.UploadRequest{
		ObjectKey:     "users/7/project-previews/uuid",
		ContentType:   "image/png",
		ContentLength: 8,
	})
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}
	if target == nil {
		t.Fatal("expected a non-nil placeholder target")
	}
	if target.ObjectKey != "" || target.ObjectURL != "" || target.UploadURL != "" || target.UploadToken != "" {
		t.Fatalf("expected an empty placeholder target, got %+v", target)
	}
}

func TestGetObjectMetadataReturnsPlaceholderMetadata(t *testing.T) {
	store := newTestStorage(t)
	metadata, err := store.GetObjectMetadata(context.Background(), "users/7/project-previews/uuid")
	if err != nil {
		t.Fatalf("get object metadata: %v", err)
	}
	if metadata == nil {
		t.Fatal("expected non-nil placeholder metadata")
	}
	if metadata.ObjectKey != "" || metadata.ObjectURL != "" || metadata.ContentType != "" || metadata.ContentLength != 0 {
		t.Fatalf("expected empty placeholder metadata, got %+v", metadata)
	}
}

func TestDeleteObjectReturnsPlaceholderResult(t *testing.T) {
	store := newTestStorage(t)
	if err := store.DeleteObject(context.Background(), "users/7/project-previews/uuid"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
}

func newTestStorage(t *testing.T) *storage.QiniuStorage {
	t.Helper()
	store, err := storage.NewQiniuStorage(storage.QiniuConfig{})
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}
	return store
}
