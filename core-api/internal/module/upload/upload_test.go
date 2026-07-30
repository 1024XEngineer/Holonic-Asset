package upload_test

import (
	"context"
	"errors"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

type storageStub struct {
	request upload.UploadRequest
	target  *upload.UploadTarget
	err     error
}

func (s *storageStub) CreateUploadTarget(_ context.Context, request upload.UploadRequest) (*upload.UploadTarget, error) {
	s.request = request
	return s.target, s.err
}

func (*storageStub) GetObjectMetadata(context.Context, string) (*upload.ObjectMetadata, error) {
	return nil, errors.New("unexpected object metadata lookup")
}

func (*storageStub) DeleteObject(context.Context, string) error {
	return nil
}

func TestUploaderReturnsPlaceholderTargetWithoutStorage(t *testing.T) {
	uploader := upload.NewUploader(nil)
	target, err := uploader.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{
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

func TestUploaderDelegatesTargetCreationToStorage(t *testing.T) {
	wantTarget := &upload.UploadTarget{ObjectURL: "https://files.example.com/object"}
	store := &storageStub{target: wantTarget}
	uploader := upload.NewUploader(store)

	target, err := uploader.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{
		ContentType:   "image/png",
		ContentLength: 8,
	})
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}
	if target != wantTarget {
		t.Fatalf("expected storage target %p, got %p", wantTarget, target)
	}
	if store.request.ContentType != "image/png" || store.request.ContentLength != 8 {
		t.Fatalf("unexpected storage request: %+v", store.request)
	}
}

func TestUploaderPropagatesStorageError(t *testing.T) {
	wantErr := errors.New("create upload target failed")
	store := &storageStub{err: wantErr}
	uploader := upload.NewUploader(store)

	target, err := uploader.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if target != nil {
		t.Fatalf("expected nil target, got %+v", target)
	}
}

func TestQiniuStorageReturnsPlaceholderValues(t *testing.T) {
	store, err := upload.NewQiniuStorage(upload.QiniuConfig{})
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
