package workspace_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
)

type projectStoreStub struct {
	project.Store
}

type assetStoreStub struct {
	asset.Store
}

type tagStoreStub struct {
	tag.Store
}

func TestNewGroupsProjectAssetAndTagManagers(t *testing.T) {
	module := workspace.New(&projectStoreStub{}, &assetStoreStub{}, &tagStoreStub{}, nil)
	if module.Projects == nil {
		t.Fatal("expected project manager")
	}
	if module.Assets == nil {
		t.Fatal("expected asset manager")
	}
	if module.Tags == nil {
		t.Fatal("expected tag manager")
	}
}

type imageServiceStub struct{}

func (*imageServiceStub) Generate(context.Context, *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	return &imageclient.GenerateResult{
		Images: []imageclient.GeneratedImage{{Base64: "generated-image", MediaType: "image/png"}},
	}, nil
}

func TestNewInjectsImageServiceIntoProjectManager(t *testing.T) {
	module := workspace.New(&projectStoreStub{}, nil, nil, &imageServiceStub{})
	result, err := module.Projects.GenerateReference(context.Background(), &project.Project{
		UserID:         7,
		Name:           "Prototype",
		GameType:       "RPG",
		Perspective:    project.PerspectiveTopDown,
		TargetPlatform: project.PlatformTypePC,
	})
	if err != nil {
		t.Fatalf("generate reference through workspace: %v", err)
	}
	if result != "data:image/png;base64,generated-image" {
		t.Fatalf("expected injected image service result, got %q", result)
	}
}

func TestNewLeavesUnavailableCapabilitiesNil(t *testing.T) {
	module := workspace.New(nil, nil, nil, nil)
	if module.Projects != nil || module.Assets != nil || module.Tags != nil {
		t.Fatalf("expected unavailable capabilities to remain nil: %+v", module)
	}
}
