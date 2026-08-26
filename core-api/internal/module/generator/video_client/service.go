package videoclient

import (
	"context"
	"strings"
)

// VideoGenerationService completes provider-independent image-to-video calls.
// It does not build business prompts or process generated video frames.
type VideoGenerationService interface {
	Generate(context.Context, *GenerateRequest) (*GenerateResult, error)
	Download(context.Context, string) ([]byte, error)
}

type videoGenerationService struct {
	provider VideoProvider
}

// NewVideoGenerationService creates the call layer backed by provider.
func NewVideoGenerationService(provider VideoProvider) VideoGenerationService {
	return &videoGenerationService{provider: provider}
}

func (s *videoGenerationService) Generate(
	ctx context.Context,
	request *GenerateRequest,
) (*GenerateResult, error) {
	if request == nil {
		return nil, invalidRequestError("video generation request is nil")
	}
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return nil, invalidRequestError("video prompt is required")
	}
	var referenceImageURLs []string
	if len(request.ReferenceImages) > 0 {
		referenceImageURLs = make([]string, 0, len(request.ReferenceImages))
		for _, ref := range request.ReferenceImages {
			url, err := referenceDataURL(ref)
			if err != nil {
				return nil, err
			}
			referenceImageURLs = append(referenceImageURLs, url)
		}
	}
	var startImageURL string
	if request.StartImage.Base64 != "" {
		var err error
		startImageURL, err = referenceDataURL(request.StartImage)
		if err != nil {
			return nil, err
		}
	} else if len(referenceImageURLs) > 0 {
		startImageURL = referenceImageURLs[0]
	} else {
		return nil, invalidRequestError("start image or reference images are required")
	}
	endImageURL := ""
	if request.EndImage != nil {
		var err error
		endImageURL, err = referenceDataURL(*request.EndImage)
		if err != nil {
			return nil, err
		}
	}
	providerResult, err := s.provider.Generate(ctx, &ProviderRequest{
		Prompt:             prompt,
		Model:              strings.TrimSpace(request.Model),
		StartImageURL:      startImageURL,
		ReferenceImageURLs: referenceImageURLs,
		EndImageURL:        endImageURL,
		Resolution:         request.Resolution,
		Duration:           request.Duration,
		AspectRatio:        request.AspectRatio,
		GenerateAudio:      request.GenerateAudio,
	})
	if err != nil {
		return nil, err
	}
	if providerResult == nil || strings.TrimSpace(providerResult.VideoURL) == "" {
		return nil, &ProviderError{
			Kind:      ErrorKindInvalidResponse,
			Transient: true,
			Message:   "provider returned no video URL",
		}
	}

	return &GenerateResult{
		RequestID: providerResult.RequestID,
		VideoURL:  providerResult.VideoURL,
	}, nil
}

func referenceDataURL(image ReferenceImage) (string, error) {
	imageBase64 := strings.TrimSpace(image.Base64)
	if imageBase64 == "" {
		return "", invalidRequestError("reference image is required")
	}
	mediaType := strings.TrimSpace(image.MediaType)
	if mediaType == "" {
		mediaType = "image/png"
	}
	return "data:" + mediaType + ";base64," + imageBase64, nil
}

func (s *videoGenerationService) Download(ctx context.Context, videoURL string) ([]byte, error) {
	if strings.TrimSpace(videoURL) == "" {
		return nil, invalidRequestError("video URL is required")
	}
	return s.provider.Download(ctx, videoURL)
}

func invalidRequestError(message string) *ProviderError {
	return &ProviderError{
		Kind:      ErrorKindInvalidRequest,
		Transient: false,
		Message:   message,
	}
}
