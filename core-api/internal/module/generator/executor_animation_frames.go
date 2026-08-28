package generator

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	videoprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/video"
)

func animationFrameSelectionOptions(frameCount int) videoprocessor.FrameIntervalSelectionOptions {
	return videoprocessor.FrameIntervalSelectionOptions{
		SampleCount:              frameCount,
		MinimumSpanFrames:        animationMinLoopSpanFrames,
		MinimumSpanRatio:         animationMinLoopSpanRatio,
		MinimumStartWindowFrames: animationMinStartWindow,
		StartWindowRatio:         animationInitialWindowRatio,
		PreferFirstFrame:         true,
		MinimumForegroundRatio:   animationMinForegroundRatio,
		EndpointMSEQuantile:      animationEndpointQuantile,
		ChangeScaleQuantile:      animationRichnessQuantile,
		ChangeBaselineQuantile:   animationMotionQuantile,
		Weights: videoprocessor.FrameIntervalSelectionWeights{
			EndpointSimilarity:    animationEndpointWeight,
			MeanAdjacentMSE:       animationRichnessWeight,
			CentroidStability:     animationCentroidStabilityWeight,
			LinearCentroidMotion:  animationTranslationWeight,
			FirstFrameSimilarity:  animationInitialFrameWeight,
			Compactness:           animationLoopCompactnessWeight,
			GeometryCoverage:      animationPoseCoverageWeight,
			ChangeCoverage:        animationMotionCoverageWeight,
			PostIntervalStability: animationRecoveryWeight,
		},
	}
}

func scaleAnimationDerivationReference(source image.Image, minimumEdge int) image.Image {
	if source == nil || minimumEdge <= 0 {
		return source
	}
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	shortEdge := min(width, height)
	if shortEdge <= 0 || shortEdge >= minimumEdge {
		return source
	}
	scale := (minimumEdge + shortEdge - 1) / shortEdge
	scaled := image.NewNRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := range height * scale {
		for x := range width * scale {
			scaled.Set(x, y, source.At(source.Bounds().Min.X+x/scale, source.Bounds().Min.Y+y/scale))
		}
	}
	return scaled
}

func (s *animationGenerationService) pixelProcessAnimationFrames(
	ctx context.Context,
	regions []imageprocessor.ImageRegion,
	columns, frameWidth, frameHeight int,
) ([]imageprocessor.ImageRegion, string, error) {
	return pixelProcessAnimationFrames(ctx, s.processor, regions, columns, frameWidth, frameHeight)
}

func pixelProcessAnimationFrames(
	ctx context.Context,
	processor imageprocessor.Processor,
	regions []imageprocessor.ImageRegion,
	columns, frameWidth, frameHeight int,
) ([]imageprocessor.ImageRegion, string, error) {
	if processor == nil {
		return nil, "", ErrImageProcessorRequired
	}
	options := AnimationPixelResizeOptions(frameWidth, frameHeight)
	processedRegions := make([]imageprocessor.ImageRegion, 0, len(regions))
	processedImages := make([]image.Image, 0, len(regions))

	// Quantize the complete sequence with one palette before resizing. Running
	// the quantizer independently for each frame lets tiny generation changes
	// move palette centroids, which makes materials flicker or appear to lose
	// colour during animation.
	sources := make([]image.Image, len(regions))
	for index, region := range regions {
		if strings.TrimSpace(region.ImageBase64) == "" {
			return nil, "", fmt.Errorf("generator: pixel-process animation frame %d: empty input", index+1)
		}
		decoded, err := imageprocessor.DecodeBase64Image(region.ImageBase64)
		if err != nil {
			return nil, "", fmt.Errorf("generator: decode animation frame %d for shared palette: %w", index+1, err)
		}
		sources[index] = decoded
	}
	quantized, err := imageprocessor.QuantizePixelArtSources(sources, options.PaletteSize)
	if err != nil {
		return nil, "", fmt.Errorf("generator: quantize animation frames with shared palette: %w", err)
	}
	quantizedImages := make([]image.Image, len(quantized))
	for index, frame := range quantized {
		quantizedImages[index] = frame
	}
	for index, region := range regions {
		quantizedBase64, err := imageprocessor.EncodePNGBase64(quantized[index])
		if err != nil {
			return nil, "", fmt.Errorf("generator: encode shared-palette animation frame %d: %w", index+1, err)
		}
		resized, err := processor.Resize(ctx, &imageprocessor.ResizeRequest{
			ImageBase64: quantizedBase64,
			Options:     options,
		})
		if err != nil {
			return nil, "", fmt.Errorf("generator: pixel-process animation frame %d: %w", index+1, err)
		}
		if resized == nil || strings.TrimSpace(resized.ImageBase64) == "" {
			return nil, "", fmt.Errorf("generator: pixel-process animation frame %d: empty result", index+1)
		}
		decoded, err := imageprocessor.DecodeBase64Image(resized.ImageBase64)
		if err != nil {
			return nil, "", fmt.Errorf("generator: decode pixel-processed animation frame %d: %w", index+1, err)
		}
		if decoded.Bounds().Dx() != frameWidth || decoded.Bounds().Dy() != frameHeight {
			return nil, "", fmt.Errorf(
				"generator: pixel-processed animation frame %d has dimensions %dx%d; want %dx%d",
				index+1,
				decoded.Bounds().Dx(),
				decoded.Bounds().Dy(),
				frameWidth,
				frameHeight,
			)
		}

		processedRegion := region
		processedRegion.ImageBase64 = resized.ImageBase64
		processedRegion.MIMEType = resized.MIMEType
		if processedRegion.MIMEType == "" {
			processedRegion.MIMEType = "image/png"
		}
		processedRegions = append(processedRegions, processedRegion)
		processedImages = append(processedImages, decoded)
	}

	sheet, err := packTransparentAnimationFrames(processedImages, columns)
	if err != nil {
		return nil, "", fmt.Errorf("generator: pack pixel-processed animation frames: %w", err)
	}
	encoded, err := imageprocessor.EncodePNGBase64(sheet)
	if err != nil {
		return nil, "", fmt.Errorf("generator: encode pixel-processed animation sheet: %w", err)
	}
	return processedRegions, encoded, nil
}

func packAnimationVideoFrames(frames []image.Image, columns int) (*image.NRGBA, error) {
	return packAnimationFrames(frames, columns, color.NRGBA{G: 255, A: 255})
}

func packTransparentAnimationFrames(frames []image.Image, columns int) (*image.NRGBA, error) {
	return packAnimationFrames(frames, columns, color.NRGBA{})
}

func packAnimationFrames(frames []image.Image, columns int, background color.NRGBA) (*image.NRGBA, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("generator: sampled animation frames are required")
	}
	if columns <= 0 {
		return nil, fmt.Errorf("generator: animation columns must be positive")
	}
	bounds := frames[0].Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("generator: sampled animation frame dimensions must be positive")
	}
	rows := animationRows(len(frames), columns)
	sheet := image.NewNRGBA(image.Rect(0, 0, width*columns, height*rows))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	for index, frame := range frames {
		if frame.Bounds().Dx() != width || frame.Bounds().Dy() != height {
			return nil, fmt.Errorf("generator: sampled animation frame %d dimensions differ", index+1)
		}
		column, row := index%columns, index/columns
		destination := image.Rect(column*width, row*height, (column+1)*width, (row+1)*height)
		draw.Draw(sheet, destination, frame, frame.Bounds().Min, draw.Src)
	}
	return sheet, nil
}

func animationRows(frameCount, columns int) int {
	return (frameCount + columns - 1) / columns
}
