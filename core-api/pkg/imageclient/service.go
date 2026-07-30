package imageclient

import (
	"context"
	"strings"
)

// ImageGenerationService completes one provider-independent image generation
// call. It does not build business prompts or process generated assets.
type ImageGenerationService interface {
	Generate(context.Context, *GenerateRequest) (*GenerateResult, error)
}

type imageGenerationService struct {
	provider ImageProvider
}

// NewImageGenerationService creates the call layer backed by provider.
func NewImageGenerationService(provider ImageProvider) ImageGenerationService {
	return &imageGenerationService{provider: provider}
}

func (s *imageGenerationService) Generate(
	ctx context.Context,
	request *GenerateRequest,
) (*GenerateResult, error) {
	providerRequest := &ProviderRequest{
		Prompt:          request.Prompt,
		ReferenceImages: append([]string(nil), request.ReferenceImages...),
		Model:           request.Model,
		Size:            request.Size,
		Params:          copyParams(request.Params),
	}

	var (
		providerResult *ProviderResult
		err            error
	)
	if len(providerRequest.ReferenceImages) == 0 {
		providerResult, err = s.provider.Generate(ctx, providerRequest)
	} else {
		providerResult, err = s.provider.Edit(ctx, providerRequest)
	}
	if err != nil {
		return nil, err
	}
	if providerResult == nil || len(providerResult.Images) == 0 {
		return nil, &ProviderError{
			Kind:      ErrorKindInvalidResponse,
			Transient: true,
			Message:   "provider returned no images",
		}
	}

	mediaType := mediaTypeForFormat(providerResult.OutputFormat)
	images := make([]GeneratedImage, 0, len(providerResult.Images))
	for _, imageBase64 := range providerResult.Images {
		if imageBase64 == "" {
			return nil, &ProviderError{
				Kind:      ErrorKindInvalidResponse,
				Transient: true,
				Message:   "provider returned an empty image",
			}
		}
		images = append(images, GeneratedImage{
			Base64:    imageBase64,
			MediaType: mediaType,
		})
	}

	return &GenerateResult{
		Images:    images,
		Model:     providerResult.Model,
		Size:      providerResult.Size,
		CreatedAt: providerResult.CreatedAt,
		Usage:     providerResult.Usage,
	}, nil
}

func copyParams(params Params) Params {
	if params == nil {
		return nil
	}
	copied := make(Params, len(params))
	for key, value := range params {
		copied[key] = value
	}
	return copied
}

func mediaTypeForFormat(format string) string {
	switch strings.ToLower(format) {
	case "", "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/" + strings.ToLower(format)
	}
}
