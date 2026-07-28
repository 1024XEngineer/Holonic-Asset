package service_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func TestCreateUploadTargetReturnsPlaceholderTarget(t *testing.T) {
	uploadService := service.NewUploadService(nil)
	request := &service.CreateUploadTargetRequest{
		ContentType:   "image/png",
		ContentLength: 8,
	}

	target, err := uploadService.CreateUploadTarget(context.Background(), request)
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
