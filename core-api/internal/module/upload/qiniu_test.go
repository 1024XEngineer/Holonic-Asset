package upload_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

func TestQiniuStorageReturnsPlaceholderValues(t *testing.T) {
	store, err := upload.NewQiniuStorage(config.QiniuConfig{})
	if err != nil {
		t.Fatalf("create object store: %v", err)
	}

	target, err := store.CreateUploadTarget(context.Background(), upload.UploadRequest{
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

	if err := store.DeleteObject(context.Background(), "users/7/project-previews/uuid"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
}
