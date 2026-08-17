package generator

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"strings"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

// prepareAnimationReference keeps an explicitly prepared green-screen reference
// pixel-identical. Other callers retain the legacy transparent-prototype
// normalization path.
func (s *animationGenerationService) prepareAnimationReference(
	ctx context.Context,
	reference string,
	prepared bool,
) (string, error) {
	reference, err := s.loadAnimationReference(ctx, reference)
	if err != nil {
		return "", err
	}
	if !prepared {
		return s.prepareGreenReference(ctx, reference)
	}
	referenceImage, err := imageprocessor.DecodeBase64Image(reference)
	if err != nil {
		return "", fmt.Errorf("generator: decode prepared animation reference: %w", err)
	}
	encoded, err := imageprocessor.EncodePNGBase64(referenceImage)
	if err != nil {
		return "", fmt.Errorf("generator: encode prepared animation reference: %w", err)
	}
	return encoded, nil
}

func (s *animationGenerationService) loadAnimationReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("generator: animation reference image is required")
	}

	if s.referenceResolver != nil {
		resolved, err := s.referenceResolver.ResolveReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("generator: resolve animation reference: %w", err)
		}
		reference = strings.TrimSpace(resolved)
		if reference == "" {
			return "", fmt.Errorf("generator: resolve animation reference: empty result")
		}
	}

	if !strings.HasPrefix(reference, "http://") && !strings.HasPrefix(reference, "https://") {
		return reference, nil
	}

	client := s.referenceHTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference, nil)
	if err != nil {
		return "", fmt.Errorf("generator: create animation reference download request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("generator: download animation reference: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return "", fmt.Errorf("generator: download animation reference: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnimationReferenceBytes+1))
	if err != nil {
		return "", fmt.Errorf("generator: read animation reference: %w", err)
	}
	if len(body) > maxAnimationReferenceBytes {
		return "", fmt.Errorf("generator: animation reference exceeds %d bytes", maxAnimationReferenceBytes)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("generator: download animation reference: empty response")
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	decoded, err := imageprocessor.DecodeBase64Image(encoded)
	if err != nil {
		return "", fmt.Errorf("generator: decode downloaded animation reference: %w", err)
	}
	canonical, err := imageprocessor.EncodePNGBase64(decoded)
	if err != nil {
		return "", fmt.Errorf("generator: encode downloaded animation reference: %w", err)
	}
	return canonical, nil
}

func (s *animationGenerationService) prepareGreenReference(
	ctx context.Context,
	referenceBase64 string,
) (string, error) {
	foregroundBase64 := referenceBase64
	reference, err := imageprocessor.DecodeBase64Image(referenceBase64)
	if err != nil {
		return "", fmt.Errorf("generator: decode animation reference: %w", err)
	}
	if !animationImageHasTransparency(reference) {
		removed, removeErr := s.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
			ImageBase64: referenceBase64,
			MatteColor:  "auto",
		})
		if removeErr != nil {
			return "", fmt.Errorf("generator: remove animation reference background: %w", removeErr)
		}
		if removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
			return "", fmt.Errorf("generator: remove animation reference background: empty result")
		}
		foregroundBase64 = removed.ImageBase64
	}

	// Use the same padded canonical-frame layout as prototype sprites. The
	// animation reference must not be made smaller by an extra safety margin;
	// the margin is part of the shared prototype/animation canvas contract.
	resizeOptions := imageprocessor.AnimationFrameResizeOptions(animationReferenceSize, animationReferenceSize)
	resized, err := s.processor.Resize(ctx, &imageprocessor.ResizeRequest{
		ImageBase64: foregroundBase64,
		Options:     resizeOptions,
	})
	if err != nil {
		return "", fmt.Errorf("generator: normalize animation reference: %w", err)
	}
	if resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
		return "", fmt.Errorf("generator: normalize animation reference: empty result")
	}
	foreground, err := imageprocessor.DecodeBase64Image(resized.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("generator: decode normalized animation reference: %w", err)
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, foreground.Bounds().Dx(), foreground.Bounds().Dy()))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), foreground, foreground.Bounds().Min, draw.Over)
	encoded, err := imageprocessor.EncodePNGBase64(canvas)
	if err != nil {
		return "", fmt.Errorf("generator: encode green animation reference: %w", err)
	}
	return encoded, nil
}

func animationImageHasTransparency(source image.Image) bool {
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			_, _, _, alpha := source.At(x, y).RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}
