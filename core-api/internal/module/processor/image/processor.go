package image

import (
	"context"
	"fmt"
	"image"
	"strings"
)

// Processor contains only deterministic local image processing. Image
// generation and provider calls belong to the generator module.
type Processor interface {
	RemoveBackground(context.Context, *RemoveBackgroundRequest) (*RemoveBackgroundResult, error)
	NormalizeReference(context.Context, *NormalizeReferenceRequest) (*NormalizeReferenceResult, error)
	Resize(context.Context, *ResizeRequest) (*ResizeResult, error)
	Verify(context.Context, *VerifyRequest) (*VerificationReport, error)
	SplitImage(context.Context, *SplitImageRequest) (*SplitImageResult, error)
}

// NormalizeReference enlarges small reference images by an integer scale
// using exact nearest-neighbour pixel replication. It is intentionally
// separate from Resize, whose filtering and framing rules target final assets.
func (p *processor) NormalizeReference(
	ctx context.Context,
	request *NormalizeReferenceRequest,
) (*NormalizeReferenceResult, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("normalize reference request is required")
	}
	if request.MaxEdge < 0 {
		return nil, fmt.Errorf("reference max edge cannot be negative")
	}

	input, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("decode reference image: %w", err)
	}
	maxEdge := request.MaxEdge
	if maxEdge == 0 {
		maxEdge = DefaultReferenceMaxEdge
	}
	width, height := input.image.Bounds().Dx(), input.image.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("reference image must not be empty")
	}

	scale := max(1, maxEdge/max(width, height))
	if scale == 1 {
		return &NormalizeReferenceResult{
			ImageBase64: request.ImageBase64,
			MIMEType:    "image/" + input.format,
			Report: ReferenceNormalizationReport{
				InputWidth: width, InputHeight: height,
				OutputWidth: width, OutputHeight: height,
				Scale: 1,
			},
		}, nil
	}
	normalized := integerNearestNeighborScale(input.image, scale)
	encoded, err := EncodePNGBase64(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode normalized reference image: %w", err)
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return &NormalizeReferenceResult{
		ImageBase64: encoded,
		MIMEType:    pngMIMEType,
		Report: ReferenceNormalizationReport{
			InputWidth: width, InputHeight: height,
			OutputWidth: normalized.Bounds().Dx(), OutputHeight: normalized.Bounds().Dy(),
			Scale: scale, Upscaled: scale > 1,
		},
	}, nil
}

type processor struct{}

// NewProcessor creates a stateless local image processor.
func NewProcessor() Processor {
	return &processor{}
}

// RemoveBackground extracts alpha from a controlled single-colour background.
// A supplied matte remains authoritative unless sampled fallback is explicitly
// enabled; applied fallback is reported through ExtractionReport.
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
	source := ToRGBA(input.image)
	output, report := ExtractChromaWithReport(source, matte, settings)
	if matte != nil && request.AllowSampledMatteFallback && !hasUsableTransparentSubject(output) {
		fallback, fallbackReport := ExtractChromaWithReport(source, nil, settings)
		if hasUsableTransparentSubject(fallback) {
			output = fallback
			report = fallbackReport
			report.FallbackApplied = true
		}
	}
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

func hasUsableTransparentSubject(img image.Image) bool {
	if img == nil {
		return false
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return false
	}
	var total, transparent, nontransparent uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			total++
			_, _, _, alpha := img.At(x, y).RGBA()
			if colorChannel8(alpha) <= TransparentAlphaMax {
				transparent++
			} else {
				nontransparent++
			}
		}
	}
	return nontransparent > 0 && ratio(transparent, total) >= MinTransparentRatio
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
		ToRGBA(input.image),
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
