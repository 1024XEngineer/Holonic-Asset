package generator

import (
	"context"

	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

// Engine coordinates Generator runs with the generic Task module.
type Engine struct {
	reader     RunReader
	tasks      taskdomain.Manager
	executor   Executor
	projects   ProjectReader
	references ReferenceStore
}

// ProjectReader supplies the canonical persisted reference when a generation
// request does not include an explicit override.
type ProjectReader interface {
	GetDetail(context.Context, uint) (*projectdomain.Project, error)
}

// EngineDependencies keeps storage and project lookup optional for lightweight
// callers and existing tests.
type EngineDependencies struct {
	Projects   ProjectReader
	References ReferenceStore
}

// NewEngine constructs Generator and binds its handlers to the injected task manager.
// A nil manager is accepted while the application composition root is incomplete.
func NewEngine(
	tasks taskdomain.Manager,
	reader RunReader,
	executor Executor,
	dependencies ...EngineDependencies,
) *Engine {
	engine := &Engine{
		reader:   reader,
		tasks:    tasks,
		executor: executor,
	}
	if len(dependencies) > 0 {
		engine.projects = dependencies[0].Projects
		engine.references = dependencies[0].References
	}
	if tasks != nil {
		engine.registerTaskHandlers(tasks)
	}
	return engine
}

var _ RunManager = (*Engine)(nil)
