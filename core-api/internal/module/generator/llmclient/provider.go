package llmclient

import (
	"context"
	"encoding/json"
)

// ProviderRequest is the normalized multimodal request passed to a provider.
type ProviderRequest struct {
	Prompt         string
	ImageURLs      []string
	Model          string
	ResponseSchema JSONSchema
}

// ProviderResult is the normalized structured response returned by a provider.
type ProviderResult struct {
	ID    string
	Model string
	JSON  json.RawMessage
	Usage Usage
}

// LLMProvider isolates the protocol details of an upstream language-model API.
type LLMProvider interface {
	Complete(context.Context, *ProviderRequest) (*ProviderResult, error)
}
