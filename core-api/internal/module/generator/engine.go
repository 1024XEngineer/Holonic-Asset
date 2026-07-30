package generator

import taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"

// Engine coordinates Generator runs with the generic Task module.
type Engine struct {
	reader   RunReader
	tasks    taskdomain.TaskManager
	executor Executor
}

// NewEngine constructs Generator and binds its handlers to the injected queue.
// A nil queue is accepted while the application composition root is incomplete.
func NewEngine(
	queue taskdomain.Queue,
	tasks taskdomain.TaskManager,
	reader RunReader,
	executor Executor,
) *Engine {
	engine := &Engine{
		reader:   reader,
		tasks:    tasks,
		executor: executor,
	}
	if queue != nil {
		engine.registerTaskHandlers(queue)
	}
	return engine
}

var _ RunManager = (*Engine)(nil)
