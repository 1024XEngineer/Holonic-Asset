package workspace_test

import (
	"testing"

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
	module := workspace.New(&projectStoreStub{}, &assetStoreStub{})
	if module.Projects == nil {
		t.Fatal("expected project manager")
	}
	if module.Assets == nil {
		t.Fatal("expected asset manager")
	}
}

func TestNewLeavesUnavailableCapabilitiesNil(t *testing.T) {
	module := workspace.New(nil, nil)
	if module.Projects != nil || module.Assets != nil {
		t.Fatalf("expected unavailable capabilities to remain nil: %+v", module)
	}
}
