package image

import (
	"context"
	"fmt"
	"strings"
)

// Processor contains only deterministic local image processing. Image
// generation and provider calls belong to the generator module.
type Processor interface {
	RemoveBackground(context.Context, *RemoveBackgroundRequest) (*RemoveBackgroundResult, error)
	Resize(context.Context, *ResizeRequest) (*ResizeResult, error)
	Verify(context.Context, *VerifyRequest) (*VerificationReport, error)
}

type processor struct{}

// NewProcessor creates a stateless local image processor.
func NewProcessor() Processor {
	return &processor{}
}

// RemoveBackground extracts alpha from a controlled single-colour background.
func (p *processor) RemoveBackground(
	ctx context.Context,
	request *RemoveBackgroundRequest,
) (*RemoveBackgroundResult, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("remove background request is required")
	}

	input, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode remove-background image: %w", err)
	}
	matteValue := strings.TrimSpace(request.MatteColor)
	if matteValue == "" {
		matteValue = DefaultMatteColor
	}
	matteColor, autoMatte, err := ParseMatteColorOrAuto(matteValue)
	if err != nil {
		return nil, fmt.Errorf("parse matte color: %w", err)
	}
	var matte *MatteColor
	if !autoMatte {
		matte = &matteColor
	}

	settings := ResolveChromaSettings(
		request.Material,
		request.Threshold,
		request.Softness,
		request.SpillSuppression,
	)
	output, report := ExtractChromaWithReport(input.image, matte, settings)
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	encoded, err := EncodePNGBase64(output)
	if err != nil {
		return nil, fmt.Errorf("encode background-removed image: %w", err)
	}
	return &RemoveBackgroundResult{
		ImageBase64: encoded,
		MIMEType:    pngMIMEType,
		Report:      report,
	}, nil
}

// Resize converts an image to a deterministic final game-asset canvas.
func (p *processor) Resize(
	ctx context.Context,
	request *ResizeRequest,
) (*ResizeResult, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("resize request is required")
	}

	input, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode resize image: %w", err)
	}
	output, report, err := ResizeImage(input.image, request.Options)
	if err != nil {
		return nil, fmt.Errorf("resize image: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	encoded, err := EncodePNGBase64(output)
	if err != nil {
		return nil, fmt.Errorf("encode resized image: %w", err)
	}
	return &ResizeResult{
		ImageBase64: encoded,
		MIMEType:    pngMIMEType,
		Report:      report,
	}, nil
}

// Verify evaluates transparent-image quality without modifying the input.
func (p *processor) Verify(
	ctx context.Context,
	request *VerifyRequest,
) (*VerificationReport, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("verify request is required")
	}

	input, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode verification image: %w", err)
	}
	var expectedMatte *MatteColor
	if strings.TrimSpace(request.ExpectedMatteColor) != "" {
		matte, parseErr := ParseMatteColor(request.ExpectedMatteColor)
		if parseErr != nil {
			return nil, fmt.Errorf("parse expected matte color: %w", parseErr)
		}
		expectedMatte = &matte
	}
	report := verifyImage(
		input.image,
		input.format == "png",
		input.colorType,
		input.hasAlpha,
		VerificationOptions{
			Profile:            request.Profile,
			ExpectedMatteColor: expectedMatte,
		},
	)
	return &report, nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
