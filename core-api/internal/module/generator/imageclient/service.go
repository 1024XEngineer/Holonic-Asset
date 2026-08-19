package imageclient

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"
)

var (
	defaultMaxAttempts = 1
	baseRetryBackoff   = time.Second
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
		MaskImage:       request.MaskImage,
		N:               request.N,
		Model:           request.Model,
		Size:            request.Size,
		Params:          copyParams(request.Params),
	}

	maxAttempts := request.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	var (
		providerResult *ProviderResult
		err            error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if len(providerRequest.ReferenceImages) == 0 {
			providerResult, err = s.provider.Generate(ctx, providerRequest)
		} else {
			providerResult, err = s.provider.Edit(ctx, providerRequest)
		}
		if err == nil {
			err = validateProviderResult(providerResult)
		}
		if err == nil {
			break
		}
		if !IsTransient(err) || errors.Is(ctx.Err(), context.Canceled) || attempt >= maxAttempts {
			return nil, err
		}
		if waitErr := backoffSleep(ctx, attempt); waitErr != nil {
			return nil, waitErr
		}
	}
	mediaType := mediaTypeForFormat(providerResult.OutputFormat)
	images := make([]GeneratedImage, 0, len(providerResult.Images))
	for _, imageBase64 := range providerResult.Images {
		images = append(images, GeneratedImage{
			Base64:    imageBase64,
			MediaType: mediaType,
		})
	}

	return &GenerateResult{
		Images:    images,
		Model:     request.Model,
		Size:      providerResult.Size,
		CreatedAt: providerResult.CreatedAt,
		Usage:     providerResult.Usage,
	}, nil
}

func validateProviderResult(result *ProviderResult) error {
	if result == nil || len(result.Images) == 0 {
		return &ProviderError{
			Kind:      ErrorKindInvalidResponse,
			Transient: true,
			Message:   "provider returned no images",
		}
	}
	if slices.Contains(result.Images, "") {
		return &ProviderError{
			Kind:      ErrorKindInvalidResponse,
			Transient: true,
			Message:   "provider returned an empty image",
		}
	}
	return nil
}

func copyParams(params Params) Params {
	if params == nil {
		return nil
	}
	copied := make(Params, len(params))
	maps.Copy(copied, params)
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

func backoffSleep(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt) * baseRetryBackoff
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
