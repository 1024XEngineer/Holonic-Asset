package generator

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

// prepareAnimationReference preserves prepared green-screen pixels while padding
// their canvas when needed. Other callers use transparent-prototype normalization.
func (s *animationGenerationService) prepareAnimationReference(
	ctx context.Context,
	reference string,
	prepared bool,
	prototypeWidth, prototypeHeight int,
	frameWidth, frameHeight int,
) (string, error) {
	reference, err := s.loadAnimationReference(ctx, reference)
	if err != nil {
		return "", err
	}
	if !prepared {
		return s.prepareGreenReference(
			ctx, reference, prototypeWidth, prototypeHeight, frameWidth, frameHeight,
		)
	}
	referenceImage, err := imageprocessor.DecodeBase64Image(reference)
	if err != nil {
		return "", fmt.Errorf("generator: decode prepared animation reference: %w", err)
	}
	preparedImage := padPreparedAnimationReference(referenceImage)
	encoded, err := imageprocessor.EncodePNGBase64(preparedImage)
	if err != nil {
		return "", fmt.Errorf("generator: encode prepared animation reference: %w", err)
	}
	return encoded, nil
}

func (s *animationGenerationService) prepareGreenReference(
	ctx context.Context,
	referenceBase64 string,
	prototypeWidth, prototypeHeight int,
	frameWidth, frameHeight int,
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

	// Preserve the prototype canvas with one uniform scale. The generated video is
	// later contained inside the requested frame, so reference preparation must
	// use the inverse of that same contain scale. Scaling width and height
	// independently distorts the prototype and makes the final animation subject
	// smaller whenever the provider and target ratios are not identical.
	canvasSize := animationReferenceCanvasSize()
	referenceSize := animationReferencePrototypeCanvasSize(
		canvasSize,
		prototypeWidth,
		prototypeHeight,
		frameWidth,
		frameHeight,
	)
	referenceWidth, referenceHeight := referenceSize.X, referenceSize.Y
	resizeOptions := imageprocessor.DefaultResizeOptions(referenceWidth, referenceHeight)
	resizeOptions.Margin = 0
	resizeOptions.CropContent = false
	resizeOptions.PreserveCanvasGeometry = true
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
	canvas := image.NewNRGBA(image.Rect(0, 0, canvasSize.X, canvasSize.Y))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	placement := image.Pt(
		(canvas.Bounds().Dx()-foreground.Bounds().Dx())/2,
		(canvas.Bounds().Dy()-foreground.Bounds().Dy())/2,
	)
	draw.Draw(canvas, foreground.Bounds().Add(placement), foreground, foreground.Bounds().Min, draw.Over)
	encoded, err := imageprocessor.EncodePNGBase64(canvas)
	if err != nil {
		return "", fmt.Errorf("generator: encode green animation reference: %w", err)
	}
	return encoded, nil
}

func animationReferenceLongEdge(referenceBase64 string) (int, error) {
	reference, err := imageprocessor.DecodeBase64Image(referenceBase64)
	if err != nil {
		return 0, fmt.Errorf("decode animation reference dimensions: %w", err)
	}
	return max(reference.Bounds().Dx(), reference.Bounds().Dy()), nil
}

func expandAnimationReferenceCanvas(referenceBase64 string, longEdge int) (string, error) {
	reference, err := imageprocessor.DecodeBase64Image(referenceBase64)
	if err != nil {
		return "", fmt.Errorf("decode animation reference for expanded canvas: %w", err)
	}
	expanded := padAnimationReference(reference, longEdge)
	encoded, err := imageprocessor.EncodePNGBase64(expanded)
	if err != nil {
		return "", fmt.Errorf("encode animation reference with expanded canvas: %w", err)
	}
	return encoded, nil
}

func padPreparedAnimationReference(source image.Image) image.Image {
	return padAnimationReference(source, 0)
}

// padAnimationReference only expands the matte canvas. It deliberately keeps
// the existing reference pixels at their current size so a framing retry shows
// the video model a smaller subject with more movement room.
func padAnimationReference(source image.Image, minimumSize int) image.Image {
	sourceWidth := source.Bounds().Dx()
	sourceHeight := source.Bounds().Dy()
	canvasSide := max(sourceWidth, sourceHeight, minimumSize)
	if canvasSide <= 0 {
		canvasSide = 1
	}
	canvasSize := image.Pt(canvasSide, canvasSide)
	if sourceWidth == canvasSide && sourceHeight == canvasSide && source.Bounds().Min == (image.Point{}) {
		return source
	}
	canvas := image.NewNRGBA(image.Rectangle{Max: canvasSize})
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	placement := image.Pt(
		(canvasSide-sourceWidth)/2,
		(canvasSide-sourceHeight)/2,
	)
	draw.Draw(canvas, source.Bounds().Add(placement.Sub(source.Bounds().Min)), source, source.Bounds().Min, draw.Src)
	return canvas
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

func (s *animationGenerationService) loadAnimationContextFrames(
	ctx context.Context,
	references []string,
) ([]image.Image, error) {
	frames := make([]image.Image, len(references))
	for index, reference := range references {
		loaded, err := s.loadAnimationReference(ctx, reference)
		if err != nil {
			return nil, fmt.Errorf("generator: load animation edit context frame %d: %w", index+1, err)
		}
		frame, err := imageprocessor.DecodeBase64Image(loaded)
		if err != nil {
			return nil, fmt.Errorf("generator: decode animation edit context frame %d: %w", index+1, err)
		}
		frames[index] = frame
	}
	return frames, nil
}
