// Package workspace groups project and asset capabilities.
package workspace

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

// Workspace exposes the project and asset capabilities as one business module.
type Workspace struct {
	Projects project.Manager
	Assets   asset.Manager
}

func New(
	projectStore project.Store,
	assetStore asset.Store,
	imageService imageclient.ImageGenerationService,
	references ...project.ReferenceStore,
) *Workspace {
	workspace := &Workspace{}
	if projectStore != nil {
		workspace.Projects = project.NewManager(projectStore, imageService, references...)
	}
	if assetStore != nil {
		workspace.Assets = asset.NewManager(assetStore)
	}
	return workspace
}
