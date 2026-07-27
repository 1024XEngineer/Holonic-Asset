package ai

import "encoding/json"

// MediaID is an opaque reference owned by the media module.
type MediaID string

type MediaRef struct {
	ID MediaID
}

type Size struct {
	Width  uint
	Height uint
}

// Usage is the provider-neutral usage collected for one AI provider call.
type Usage struct {
	InputTokens         uint
	OutputTokens        uint
	GeneratedImages     uint
	AudioDurationMillis uint64
}

type ProjectContext struct {
	ProjectID      uint
	Summary        string
	Attributes     json.RawMessage
	ReferenceMedia []MediaRef
}

type PromptRequest struct {
	ProjectID   uint
	Operation   string
	Instruction string
	Attributes  json.RawMessage
}

type Prompt struct {
	System          string
	User            string
	TemplateVersion string
}

type StepType string
type RequestKind string

type PlanConstraints struct {
	AllowedStepTypes []StepType
	AllowedMediaIDs  []MediaID
	MaxSteps         uint
	MaxRetries       uint
}

type PlanRequest struct {
	RequestID              string
	ProjectID              uint
	AssetID                uint
	RequestKind            RequestKind
	Prompt                 string
	ReferenceMediaIDs      []MediaID
	TargetAssetResourceIDs []uint
	Parameters             json.RawMessage
	Constraints            PlanConstraints
}

type ProposedStep struct {
	Key         string
	Type        StepType
	DependsOn   []string
	Parameters  json.RawMessage
	MaxAttempts uint
}

// PlanProposal has no lifecycle of its own. The generation module validates
// and converts it into its authoritative Plan and Step state.
type PlanProposal struct {
	Steps []ProposedStep
	Usage Usage
}

type GenerateImageRequest struct {
	RequestID  string
	ProjectID  uint
	Prompt     PromptRequest
	References []MediaRef
	Size       Size
}

type EditImageRequest struct {
	RequestID string
	ProjectID uint
	Prompt    PromptRequest
	Targets   []MediaRef
	Mask      *MediaRef
}

type ImageResult struct {
	Outputs []MediaRef
	Usage   Usage
}

type GenerateAudioRequest struct {
	RequestID      string
	ProjectID      uint
	Prompt         PromptRequest
	References     []MediaRef
	DurationMillis uint64
}

type EditAudioRequest struct {
	RequestID string
	ProjectID uint
	Prompt    PromptRequest
	Targets   []MediaRef
}

type AudioResult struct {
	Outputs []MediaRef
	Usage   Usage
}

type ModelCapability string

const (
	ModelCapabilityPlanning        ModelCapability = "planning"
	ModelCapabilityImageGeneration ModelCapability = "image_generation"
	ModelCapabilityImageEditing    ModelCapability = "image_editing"
	ModelCapabilityAudioGeneration ModelCapability = "audio_generation"
	ModelCapabilityAudioEditing    ModelCapability = "audio_editing"
)

type ModelSelection struct {
	Provider string
	Model    string
}
