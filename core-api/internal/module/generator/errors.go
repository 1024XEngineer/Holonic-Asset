package generator

import "errors"

var (
	ErrInvalidRunListStatus = errors.New("generator: invalid run list status")
	ErrTaskManagerRequired  = errors.New("generator: task manager is required")
	ErrTaskRequired         = errors.New("generator: task is required")
	ErrUnsupportedTaskType  = errors.New("generator: unsupported task type")
	ErrExecutorRequired     = errors.New("generator: executor is required")
	ErrImageServiceRequired = errors.New("generator: image service is required")
	ErrAssetWriterRequired  = errors.New("generator: asset writer is required")
	ErrImageResultRequired  = errors.New("generator: image result is required")
)
