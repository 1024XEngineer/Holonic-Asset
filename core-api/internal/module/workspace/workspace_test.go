package workspace_test

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

type projectStoreStub struct {
	project.Store
}

type assetStoreStub struct {
	asset.Store
}

func TestNewGroupsProjectAndAssetManagers(t *testing.T) {
	module := workspace.New(&projectStoreStub{}, &assetStoreStub{}, nil)
	if module.Projects == nil {
		t.Fatal("expected project manager")
	}
	if module.Assets == nil {
		t.Fatal("expected asset manager")
	}
}

type imageServiceStub struct{}

func (*imageServiceStub) Generate(context.Context, *imageclient.GenerateRequest) (*imageclient.GenerateResult, error) {
	return &imageclient.GenerateResult{
		Images: []imageclient.GeneratedImage{{Base64: "generated-image"}},
	}, nil
}

func TestNewInjectsImageServiceIntoProjectManager(t *testing.T) {
	module := workspace.New(&projectStoreStub{}, nil, &imageServiceStub{})
	result, err := module.Projects.GenerateVisualImage(context.Background(), &project.Project{
		UserID:         7,
		Name:           "Prototype",
		GameType:       project.GameTypeRPG,
		ViewType:       project.ViewTypeTopDown,
		TargetPlatform: project.PlatformTypePC,
	})
	if err != nil {
		t.Fatalf("generate visual image through workspace: %v", err)
	}
	if result.Reference != "generated-image" {
		t.Fatalf("expected injected image service result, got %q", result.Reference)
	}
}

func TestNewLeavesUnavailableCapabilitiesNil(t *testing.T) {
	module := workspace.New(nil, nil, nil)
	if module.Projects != nil || module.Assets != nil {
		t.Fatalf("expected unavailable capabilities to remain nil: %+v", module)
	}
}
