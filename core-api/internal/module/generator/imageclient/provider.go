package imageclient

import "context"

// ProviderRequest is the request passed from the call layer to a model provider.
type ProviderRequest struct {
	Prompt          string
	ReferenceImages []string
	Model           string
	Size            string
	Params          Params
}

// ProviderResult is the normalized response returned by a model provider.
type ProviderResult struct {
	Images       []string
	OutputFormat string
	Size         string
	CreatedAt    int64
	Usage        Usage
}

// ImageProvider encapsulates the protocol details of an upstream model vendor.
type ImageProvider interface {
	Generate(context.Context, *ProviderRequest) (*ProviderResult, error)
	Edit(context.Context, *ProviderRequest) (*ProviderResult, error)
}
