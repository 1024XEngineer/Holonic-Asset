package generator

import taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"

// Engine coordinates Generator runs with the generic Task module.
type Engine struct {
	reader   *RunReader
	tasks    taskdomain.Manager
	executor Executor
}

// NewEngine constructs Generator and binds its handlers to the injected task manager.
// A nil manager is accepted while the application composition root is incomplete.
func NewEngine(tasks taskdomain.Manager, executor Executor) *Engine {
	engine := &Engine{
		reader:   NewRunReader(tasks),
		tasks:    tasks,
		executor: executor,
	}
	if tasks != nil {
		engine.registerTaskHandlers(tasks)
	}
	return engine
}

var _ RunManager = (*Engine)(nil)
