package videoclient

// GenerateRequest describes one image-to-video generation call.
//
// ReferenceImageBase64 accepts raw base64 image data. ReferenceImageMediaType
// defaults to image/png when omitted.
type GenerateRequest struct {
	Prompt                  string
	ReferenceImageBase64    string
	ReferenceImageMediaType string
	Resolution              string
	Duration                int
	AspectRatio             string
	GenerateAudio           bool
}

// GenerateResult is the provider-independent result of one video generation
// call. VideoURL points to the generated video and may be downloaded through
// VideoGenerationService.Download.
type GenerateResult struct {
	RequestID string
	VideoURL  string
}
