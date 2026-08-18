package videoclient

import "context"

// ProviderRequest is the request passed from the call layer to a video provider.
type ProviderRequest struct {
	Prompt        string
	StartImageURL string
	EndImageURL   string
	Resolution    string
	Duration      int
	AspectRatio   string
	GenerateAudio bool
}

// ProviderResult is the normalized response returned by a video provider.
type ProviderResult struct {
	RequestID string
	VideoURL  string
}

// VideoProvider encapsulates the protocol details of an upstream video vendor.
type VideoProvider interface {
	Generate(context.Context, *ProviderRequest) (*ProviderResult, error)
	Download(context.Context, string) ([]byte, error)
}
