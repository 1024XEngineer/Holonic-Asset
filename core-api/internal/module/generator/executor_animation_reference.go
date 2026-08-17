package generator

import (
<<<<<<< HEAD
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
=======
	"fmt"
	"net/url"
	"strings"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func animationReference(asset assetdomain.Asset, direction string) (string, bool, error) {
	if asset.Type != assetdomain.AssetTypeCharacter && asset.Type != assetdomain.AssetTypeObject {
		return "", false, fmt.Errorf("generator: asset type %q does not support animation generation", asset.Type)
	}
	// Select the requested direction and resolve its -unprocessed image-hosting
	// reference.
	content, err := asset.DecodeContent()
	if err != nil {
		return "", false, fmt.Errorf("generator: decode animation asset %d content: %w", asset.ID, err)
	}
	prototypeIndex, err := animationDirectionIndex(direction, content.DirectionCount)
	if err != nil {
		return "", false, err
	}

	if content.Prototype == nil || prototypeIndex >= len(*content.Prototype) {
		return "", false, fmt.Errorf("generator: animation asset %d has no prototype for direction %q", asset.ID, direction)
	}
	prototype := (*content.Prototype)[prototypeIndex]
	if prototype.URL == nil || strings.TrimSpace(*prototype.URL) == "" {
		return "", false, fmt.Errorf("generator: animation asset %d prototype direction %q has no image URL", asset.ID, direction)
	}
	unprocessedURL := animationUnprocessedImageURL(strings.TrimSpace(*prototype.URL))
	return unprocessedURL, false, nil
}

func animationUnprocessedImageURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "data:") {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return addObjectKeySuffix(value, "-unprocessed")
	}
	parsed.Path = addObjectKeySuffix(parsed.Path, "-unprocessed")
	return parsed.String()
>>>>>>> 45e65c3 (fix(generator):change task result to patch and return url)
}
