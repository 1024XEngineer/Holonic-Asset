package generator

import "errors"

var (
	ErrInvalidRunListStatus = errors.New("generator: invalid run list status")
	ErrTaskManagerRequired  = errors.New("generator: task manager is required")
	ErrTaskRequired         = errors.New("generator: task is required")
	ErrUnsupportedTaskType  = errors.New("generator: unsupported task type")
)
