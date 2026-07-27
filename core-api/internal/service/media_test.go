package service_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/service"
)

func TestCreateProjectPreviewUploadTargetReturnsPlaceholderTarget(t *testing.T) {
	mediaService := service.NewMediaService()
	request := &service.CreateProjectPreviewUploadRequest{ContentType: "image/png"}

	target, err := mediaService.CreateProjectPreviewUploadTarget(context.Background(), request)
	if err != nil {
		t.Fatalf("create project preview upload target: %v", err)
	}
	if target == nil {
		t.Fatal("expected a non-nil placeholder target")
	}
	if target.ObjectKey != "" || target.UploadURL != "" {
		t.Fatalf("expected an empty placeholder target, got %+v", target)
	}
}
