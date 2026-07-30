// Package workspace groups project and asset capabilities.
package workspace

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

// Workspace exposes the project and asset capabilities as one business module.
type Workspace struct {
	Projects project.Manager
	Assets   asset.Manager
}

// New constructs the available Workspace capabilities from their stores.
func New(projectStore project.Store, assetStore asset.Store) *Workspace {
	workspace := &Workspace{}
	if projectStore != nil {
		workspace.Projects = project.NewManager(projectStore)
	}
	if assetStore != nil {
		workspace.Assets = asset.NewManager(assetStore)
	}
	return workspace
}
