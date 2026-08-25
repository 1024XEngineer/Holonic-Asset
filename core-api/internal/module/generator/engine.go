package generator

import (
	"context"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

// Engine coordinates Generator runs with the generic Task module.
type Engine struct {
	reader     *RunReader
	tasks      taskdomain.Manager
	executor   Executor
	projects   ProjectReader
	assets     AssetReader
	references ReferenceStore
}

// ProjectReader supplies the canonical persisted reference when a generation
// request does not include an explicit override.
type ProjectReader interface {
	GetDetail(context.Context, uint) (*projectdomain.Project, error)
}

// AssetReader supplies the project-scoped list projection used for tag
// reference selection as well as detail lookup for asset validation.
type AssetReader interface {
	GetDetail(context.Context, uint) (assetdomain.Asset, error)
	GetAssets(context.Context, uint, assetdomain.AssetListFilter) ([]assetdomain.Asset, error)
}

// EngineDependencies keeps storage and project lookup optional for lightweight
// callers and existing tests.
type EngineDependencies struct {
	Projects   ProjectReader
	Assets     AssetReader
	References ReferenceStore
}

// NewEngine constructs Generator from the generic Task module and binds its handlers.
// A nil manager is accepted while the application composition root is incomplete.
func NewEngine(
	tasks taskdomain.Manager,
	executor Executor,
	dependencies ...EngineDependencies,
) *Engine {
	engine := &Engine{
		reader:   NewRunReader(tasks),
		tasks:    tasks,
		executor: executor,
	}
	if len(dependencies) > 0 {
		engine.projects = dependencies[0].Projects
		engine.assets = dependencies[0].Assets
		engine.references = dependencies[0].References
	}
	if tasks != nil {
		engine.registerTaskHandlers(tasks)
	}
	return engine
}

var _ RunManager = (*Engine)(nil)
