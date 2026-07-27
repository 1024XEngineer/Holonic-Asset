package domain

type StepExecutor string
type RequestKind string
type RunLifecycle string

const (
	RunLifecycleAccepted            RunLifecycle = "accepted"
	RunLifecyclePlanning            RunLifecycle = "planning"
	RunLifecyclePlanned             RunLifecycle = "planned"
	RunLifecycleGenerating          RunLifecycle = "generating"
	RunLifecyclePostProcessing      RunLifecycle = "post_processing"
	RunLifecycleWaitingConfirmation RunLifecycle = "waiting_confirmation"
	RunLifecycleCompleted           RunLifecycle = "completed"
	RunLifecycleFailed              RunLifecycle = "failed"
	RunLifecycleCancelled           RunLifecycle = "cancelled"
)

const (
	StepExecutorAI          StepExecutor = "ai"
	StepExecutorAssetWorker StepExecutor = "asset_worker"
)

const (
	RequestKindGenerateCharacter      RequestKind = "generate_character"
	RequestKindGenerateProjectPreview RequestKind = "generate_project_preview"
	RequestKindGenerateTileSetItem    RequestKind = "generate_tileset_item"
	RequestKindEditTileSetItem        RequestKind = "edit_tileset_item"
	RequestKindGenerateObject         RequestKind = "generate_object"
	RequestKindGenerateSceneryLayer   RequestKind = "generate_scenery_layer"
	RequestKindEditSceneryLayer       RequestKind = "edit_scenery_layer"
	RequestKindGenerateAnimation      RequestKind = "generate_animation"
	RequestKindEditFrame              RequestKind = "edit_frame"
	RequestKindGenerateUI             RequestKind = "generate_ui"
	RequestKindEditUIComponent        RequestKind = "edit_ui_component"
	RequestKindGenerateAudio          RequestKind = "generate_audio"
	RequestKindEditAudio              RequestKind = "edit_audio"
)
