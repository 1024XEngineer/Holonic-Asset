package videoclient

// ReferenceImage is one image input for a video generation call.
// Base64 accepts raw base64 image data. MediaType defaults to image/png.
type ReferenceImage struct {
	Base64    string
	MediaType string
}

// GenerateRequest describes one image-to-video call. EndImage optionally pins
// the final frame for first/last-frame video generation.
type GenerateRequest struct {
	Prompt        string
	Model         string
	StartImage    ReferenceImage
	EndImage      *ReferenceImage
	Resolution    string
	Duration      int
	AspectRatio   string
	GenerateAudio bool
}

// GenerateResult is the provider-independent result of one video generation
// call. VideoURL points to the generated video and may be downloaded through
// VideoGenerationService.Download.
type GenerateResult struct {
	RequestID string
	VideoURL  string
}
