package imageclient

// Params carries provider-specific request parameters that the generic layer
// does not interpret. Each provider reads only the keys it understands; for
// example, the QNA provider reads "quality". Callers must consult a provider's
// documentation for supported keys.
type Params map[string]string

// GenerateRequest describes one image generation call.
//
// When ReferenceImages is empty, the service performs text-to-image generation.
// Otherwise, it performs image-to-image generation. Each reference may be an
// accessible URL or a base64 data URI supported by the provider.
type GenerateRequest struct {
	Prompt          string
	ReferenceImages []string
	MaskImage       string
	Model           string
	Size            string
	Params          Params
}

// GeneratedImage is one normalized image returned by a model provider.
type GeneratedImage struct {
	Base64    string
	MediaType string
}

// Usage contains the provider usage values relevant to callers.
type Usage struct {
	TotalTokens       int
	InputTokens       int
	OutputTokens      int
	TextToImageCount  int
	ImageToImageCount int
	RequestCount      int
}

// GenerateResult is the provider-independent result of one generation call.
type GenerateResult struct {
	Images    []GeneratedImage
	Model     string
	Size      string
	CreatedAt int64
	Usage     Usage
}
