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

// prepareAdaptiveAnimationReference removes any existing matte, selects a
// subject-safe saturated colour, and composes the provider reference with that
// colour. The selected matte is returned so prompt, video analysis, and frame
// extraction use one contract.
func (s *animationGenerationService) prepareAdaptiveAnimationReference(
	ctx context.Context,
	reference string,
	prepared bool,
	prototypeWidth, prototypeHeight int,
	frameWidth, frameHeight int,
	matteOverride *imageprocessor.MatteColor,
) (string, imageprocessor.MatteColor, error) {
	reference, err := s.loadAnimationReference(ctx, reference)
	if err != nil {
		return "", imageprocessor.MatteColor{}, err
	}
	referenceImage, err := imageprocessor.DecodeBase64Image(reference)
	if err != nil {
		return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: decode animation reference: %w", err)
	}

	var foreground image.Image
	if prepared {
		foreground = imageprocessor.PrepareAnimationForeground(referenceImage)
	} else {
		foregroundBase64 := reference
		if !animationImageHasTransparency(referenceImage) {
			removed, removeErr := s.processor.RemoveBackground(ctx, &imageprocessor.RemoveBackgroundRequest{
				ImageBase64: reference,
				MatteColor:  "auto",
			})
			if removeErr != nil {
				return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: remove animation reference background: %w", removeErr)
			}
			if removed == nil || strings.TrimSpace(removed.ImageBase64) == "" {
				return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: remove animation reference background: empty result")
			}
			foregroundBase64 = removed.ImageBase64
		}
		foreground, err = imageprocessor.DecodeBase64Image(foregroundBase64)
		if err != nil {
			return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: decode normalized animation foreground: %w", err)
		}
	}
	if foreground == nil || foreground.Bounds().Empty() {
		return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: animation reference foreground is empty")
	}
	matte := imageprocessor.SelectAnimationMatteColor(foreground)
	if matteOverride != nil {
		matte = *matteOverride
	}

	canvasSize := animationReferenceCanvasSize()
	if prepared {
		// Prepared references have already been composed at the intended source
		// scale. Preserve that geometry exactly (the historic prepared-reference
		// contract) while replacing their old key colour with the selected matte.
		canvasSize = image.Pt(max(foreground.Bounds().Dx(), foreground.Bounds().Dy()), max(foreground.Bounds().Dx(), foreground.Bounds().Dy()))
	} else {
		referenceSize := animationReferencePrototypeCanvasSize(
			canvasSize, prototypeWidth, prototypeHeight, frameWidth, frameHeight,
		)
		foregroundBase64, encodeErr := imageprocessor.EncodePNGBase64(foreground)
		if encodeErr != nil {
			return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: encode animation foreground: %w", encodeErr)
		}
		resized, resizeErr := s.processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: foregroundBase64,
			Options: imageprocessor.ResizeOptions{
				Width: referenceSize.X, Height: referenceSize.Y,
				Margin: 0, CropContent: false, PreserveCanvasGeometry: true,
			},
		})
		if resizeErr != nil {
			return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: normalize animation reference: %w", resizeErr)
		}
		if resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
			return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: normalize animation reference: empty result")
		}
		foreground, err = imageprocessor.DecodeBase64Image(resized.ImageBase64)
		if err != nil {
			return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: decode normalized animation reference: %w", err)
		}
	}
	canvas := imageprocessor.CompositeAnimationMatte(foreground, matte, canvasSize)
	encoded, err := imageprocessor.EncodePNGBase64(canvas)
	if err != nil {
		return "", imageprocessor.MatteColor{}, fmt.Errorf("generator: encode adaptive animation reference: %w", err)
	}
	return encoded, matte, nil
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
	// The source already carries the selected provider matte. Preserve that
	// colour when adding retry room instead of reintroducing the historical
	// green screen around an adaptive reference.
	matte := color.NRGBAModel.Convert(source.At(source.Bounds().Min.X, source.Bounds().Min.Y)).(color.NRGBA)
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: matte}, image.Point{}, draw.Src)
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
