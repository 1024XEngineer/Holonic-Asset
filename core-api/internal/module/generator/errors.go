package generator

import "errors"

var (
	ErrInvalidRunListStatus        = errors.New("generator: invalid run list status")
	ErrInvalidRunListCursor        = errors.New("generator: invalid run list cursor")
	ErrRunNotFailed                = errors.New("generator: run is not failed")
	ErrInvalidTaskPayload          = errors.New("generator: invalid task payload")
	ErrTaskManagerRequired         = errors.New("generator: task manager is required")
	ErrTaskRequired                = errors.New("generator: task is required")
	ErrUnsupportedTaskType         = errors.New("generator: unsupported task type")
	ErrExecutorRequired            = errors.New("generator: executor is required")
	ErrImageServiceRequired        = errors.New("generator: image service is required")
	ErrVideoServiceRequired        = errors.New("generator: video service is required")
	ErrVideoFrameExtractorRequired = errors.New(
		"generator: video frame extractor is required",
	)
	ErrAnimationServiceRequired        = errors.New("generator: animation service is required")
	ErrAnimationReferenceStoreRequired = errors.New(
		"generator: animation reference store is required",
	)
	ErrImageProcessorRequired = errors.New("generator: image processor is required")
	ErrAssetWriterRequired    = errors.New("generator: asset writer is required")
	ErrAssetReaderRequired    = errors.New("generator: asset reader is required for Nexus References")
	ErrImageResultRequired    = errors.New("generator: image result is required")
	ErrProjectReaderRequired  = errors.New("generator: project reader is required")
	ErrReferenceStoreRequired = errors.New("generator: reference store is required")
	ErrLLMServiceRequired     = errors.New("generator: LLM service is required")
	ErrInvalidSceneryPayload  = errors.New("generator: invalid scenery payload")
	ErrInvalidSceneryPlan     = errors.New("generator: invalid scenery plan")
	ErrInvalidSceneryLayout   = errors.New("generator: invalid scenery layout")
	ErrResourceStoreRequired  = errors.New("generator: resource store is required")
)
