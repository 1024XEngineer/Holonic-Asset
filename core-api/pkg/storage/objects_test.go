package storage_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/pkg/storage"
)

func TestUploadFromURLReturnsPlaceholderObject(t *testing.T) {
	object, err := storage.UploadFromURL(context.Background(), storage.UploadFromURLRequest{
		SourceURL:   "https://ai-provider.example/generated/image.png",
		ObjectKey:   "projects/42/previews/uuid",
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("upload from URL: %v", err)
	}
	if object == nil {
		t.Fatal("expected a non-nil placeholder object")
	}
	if object.ObjectKey != "" || object.ObjectURL != "" {
		t.Fatalf("expected an empty placeholder object, got %+v", object)
	}
}

func TestCreatePresignedUploadReturnsPlaceholderTarget(t *testing.T) {
	target, err := storage.CreatePresignedUpload(context.Background(), storage.PresignedUploadRequest{
		ObjectKey:   "users/7/project-previews/uuid",
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("create presigned upload: %v", err)
	}
	if target == nil {
		t.Fatal("expected a non-nil placeholder target")
	}
	if target.ObjectKey != "" || target.UploadURL != "" {
		t.Fatalf("expected an empty placeholder target, got %+v", target)
	}
}

func TestHeadObjectReturnsPlaceholderMetadata(t *testing.T) {
	metadata, err := storage.HeadObject(context.Background(), "users/7/project-previews/uuid")
	if err != nil {
		t.Fatalf("head object: %v", err)
	}
	if metadata == nil {
		t.Fatal("expected non-nil placeholder metadata")
	}
	if metadata.ObjectKey != "" || metadata.ContentType != "" {
		t.Fatalf("expected empty placeholder metadata, got %+v", metadata)
	}
}
