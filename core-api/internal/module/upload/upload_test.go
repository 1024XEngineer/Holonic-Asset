package upload_test

import (
	"context"
	"errors"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

type storeStub struct {
	request upload.UploadRequest
	target  *upload.UploadTarget
	err     error
}

func (s *storeStub) CreateUploadTarget(_ context.Context, request upload.UploadRequest) (*upload.UploadTarget, error) {
	s.request = request
	return s.target, s.err
}

func (*storeStub) GetObjectMetadata(context.Context, string) (*upload.ObjectMetadata, error) {
	return nil, errors.New("unexpected object metadata lookup")
}

func TestManagerReturnsPlaceholderTargetWithoutStore(t *testing.T) {
	manager := upload.NewManager(nil)
	target, err := manager.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{
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

func TestManagerDelegatesTargetCreationToStore(t *testing.T) {
	wantTarget := &upload.UploadTarget{ObjectURL: "https://files.example.com/object"}
	store := &storeStub{target: wantTarget}
	manager := upload.NewManager(store)

	target, err := manager.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{
		ContentType:   "image/png",
		ContentLength: 8,
	})
	if err != nil {
		t.Fatalf("create upload target: %v", err)
	}
	if target != wantTarget {
		t.Fatalf("expected store target %p, got %p", wantTarget, target)
	}
	if store.request.ContentType != "image/png" || store.request.ContentLength != 8 {
		t.Fatalf("unexpected storage request: %+v", store.request)
	}
}

func TestManagerPropagatesStoreError(t *testing.T) {
	wantErr := errors.New("create upload target failed")
	store := &storeStub{err: wantErr}
	manager := upload.NewManager(store)

	target, err := manager.CreateUploadTarget(context.Background(), &upload.CreateUploadTargetRequest{
		ContentType:   "image/png",
		ContentLength: 8,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if target != nil {
		t.Fatalf("expected nil target, got %+v", target)
	}
}

func TestManagerRejectsInvalidUploadRequest(t *testing.T) {
	store := &storeStub{}
	manager := upload.NewManager(store)

	tests := []struct {
		name    string
		request *upload.CreateUploadTargetRequest
	}{
		{name: "nil request"},
		{name: "empty content type", request: &upload.CreateUploadTargetRequest{ContentLength: 8}},
		{name: "blank content type", request: &upload.CreateUploadTargetRequest{ContentType: "   ", ContentLength: 8}},
		{name: "zero content length", request: &upload.CreateUploadTargetRequest{ContentType: "image/png"}},
		{name: "negative content length", request: &upload.CreateUploadTargetRequest{ContentType: "image/png", ContentLength: -1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, err := manager.CreateUploadTarget(context.Background(), test.request)
			if !errors.Is(err, upload.ErrInvalidUploadRequest) {
				t.Fatalf("expected invalid upload request error, got %v", err)
			}
			if target != nil {
				t.Fatalf("expected nil target, got %+v", target)
			}
		})
	}
}
