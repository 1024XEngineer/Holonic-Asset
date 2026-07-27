// Package provider defines provider-neutral ports for external AI models.
package provider

import (
	"context"
	"encoding/json"
)

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole
	Content string
}

type Usage struct {
	InputTokens         uint
	OutputTokens        uint
	GeneratedImages     uint
	AudioDurationMillis uint64
}

type StructuredTextRequest struct {
	RequestID      string
	Model          string
	Messages       []Message
	ResponseSchema json.RawMessage
}

type StructuredTextResult struct {
	ProviderRequestID string
	Model             string
	Output            json.RawMessage
	Usage             Usage
}

type Size struct {
	Width  uint
	Height uint
}

type ImageGenerationRequest struct {
	RequestID     string
	Model         string
	Prompt        string
	ReferenceURLs []string
	Size          Size
}

type ImageEditRequest struct {
	RequestID  string
	Model      string
	Prompt     string
	TargetURLs []string
	MaskURL    string
}

type AudioGenerationRequest struct {
	RequestID      string
	Model          string
	Prompt         string
	ReferenceURLs  []string
	DurationMillis uint64
}

type AudioEditRequest struct {
	RequestID  string
	Model      string
	Prompt     string
	TargetURLs []string
}

type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusSucceeded OperationStatus = "succeeded"
	OperationStatusFailed    OperationStatus = "failed"
	OperationStatusCancelled OperationStatus = "cancelled"
)

type Output struct {
	URL       string
	MediaType string
}

type OperationResult struct {
	OperationID       string
	ProviderRequestID string
	Model             string
	Status            OperationStatus
	Outputs           []Output
	Usage             Usage
	ErrorMessage      string
}

type TextProvider interface {
	CompleteStructured(ctx context.Context, request *StructuredTextRequest) (*StructuredTextResult, error)
}

type ImageProvider interface {
	Generate(ctx context.Context, request *ImageGenerationRequest) (*OperationResult, error)
	Edit(ctx context.Context, request *ImageEditRequest) (*OperationResult, error)
	GetOperation(ctx context.Context, operationID string) (*OperationResult, error)
	CancelOperation(ctx context.Context, operationID string) error
}

type AudioProvider interface {
	Generate(ctx context.Context, request *AudioGenerationRequest) (*OperationResult, error)
	Edit(ctx context.Context, request *AudioEditRequest) (*OperationResult, error)
	GetOperation(ctx context.Context, operationID string) (*OperationResult, error)
	CancelOperation(ctx context.Context, operationID string) error
}
