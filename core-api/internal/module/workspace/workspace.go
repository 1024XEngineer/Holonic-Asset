// Package workspace groups project, asset, and tag capabilities.
package workspace

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/tag"
)

// Workspace exposes the project, asset, and tag capabilities as one business module.
type Workspace struct {
	Projects project.Manager
	Assets   asset.Manager
	Tags     tag.Manager
}

func New(
	projectStore project.Store,
	assetStore asset.Store,
	tagStore tag.Store,
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
	if tagStore != nil {
		workspace.Tags = tag.NewManager(tagStore)
	}
	return workspace
}
