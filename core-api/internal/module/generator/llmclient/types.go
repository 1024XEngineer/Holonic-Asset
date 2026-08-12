package llmclient

import "encoding/json"

// ImageInput identifies one image that the model must inspect. URL accepts an
// accessible HTTP(S) URL or an image data URI.
type ImageInput struct {
	URL string
}

// JSONSchema describes the structured response required from the model.
type JSONSchema struct {
	Name   string
	Schema json.RawMessage
}

// CompletionRequest describes one multimodal structured completion.
type CompletionRequest struct {
	Prompt         string
	Images         []ImageInput
	Model          string
	ResponseSchema JSONSchema
}

// Usage contains normalized token counts reported by the provider.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// CompletionResult is the provider-independent structured completion result.
type CompletionResult struct {
	ID    string
	Model string
	JSON  json.RawMessage
	Usage Usage
}
