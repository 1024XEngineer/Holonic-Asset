package video

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// FrameObservation contains media measurements for one sampled source frame.
type FrameObservation struct {
	Safe           bool
	CentroidX      float64
	CentroidY      float64
	Width          float64
	Height         float64
	ForegroundArea int
}

// FrameSequenceAnalysis exposes media measurements without prescribing how a
// caller should choose an interval or score candidates.
type FrameSequenceAnalysis struct {
	FPS             int
	Frames          []FrameObservation
	PairwiseMSE     [][]float64
	ForegroundRatio float64
}

// FramePairDifference measures how much a candidate frame differs from the
// corresponding original frame while ignoring the configured chroma-key
// background.
type FramePairDifference struct {
	AppearanceMSE            float64
	ForegroundMaskDifference float64
}

type frameDescriptor struct {
	mask       [analysisSize * analysisSize]bool
	gray       [analysisSize * analysisSize]float64
	cx         float64
	cy         float64
	width      float64
	height     float64
	foreground int
}

type frameAnalysis struct {
	descriptor frameDescriptor
	safe       bool
}

// AnalyzeFrameSequence measures already-decoded video frames with the same
// foreground model used by extraction and interval selection. This lets
// callers validate a recomposed sequence without encoding it back into a
// temporary video first.
func AnalyzeFrameSequence(frames []image.Image, fps int, chromaKey ChromaKey) (FrameSequenceAnalysis, error) {
	if len(frames) == 0 {
		return FrameSequenceAnalysis{}, fmt.Errorf("video: frame sequence is required")
	}
	if fps < 0 {
		return FrameSequenceAnalysis{}, fmt.Errorf("video: frame sequence FPS must not be negative")
	}
	if !chromaKey.valid() {
		return FrameSequenceAnalysis{}, fmt.Errorf("video: valid chroma key settings are required")
	}
	analyses := make([]frameAnalysis, len(frames))
	for index, frame := range frames {
		if frame == nil || frame.Bounds().Empty() {
			return FrameSequenceAnalysis{}, fmt.Errorf("video: frame sequence image %d is empty", index)
		}
		analyses[index] = frameAnalysis{
			descriptor: describeFrame(frame, chromaKey),
			safe:       frameInsideSafetyBand(frame, chromaKey),
		}
	}
	return buildFrameSequenceAnalysis(analyses, fps), nil
}

// AnalyzeFramePairs compares corresponding original and candidate frames with
// the same foreground descriptor used by sequence analysis.
func AnalyzeFramePairs(
	original, candidate []image.Image,
	chromaKey ChromaKey,
) ([]FramePairDifference, error) {
	if len(original) == 0 {
		return nil, fmt.Errorf("video: original frame sequence is required")
	}
	if len(original) != len(candidate) {
		return nil, fmt.Errorf("video: frame pair sequences contain %d and %d frames", len(original), len(candidate))
	}
	if !chromaKey.valid() {
		return nil, fmt.Errorf("video: valid chroma key settings are required")
	}
	differences := make([]FramePairDifference, len(original))
	for index := range original {
		if original[index] == nil || original[index].Bounds().Empty() {
			return nil, fmt.Errorf("video: original frame sequence image %d is empty", index)
		}
		if candidate[index] == nil || candidate[index].Bounds().Empty() {
			return nil, fmt.Errorf("video: candidate frame sequence image %d is empty", index)
		}
		left := describeFrame(original[index], chromaKey)
		right := describeFrame(candidate[index], chromaKey)
		var union [analysisSize * analysisSize]bool
		unionArea := 0
		for pixel := range union {
			union[pixel] = left.mask[pixel] || right.mask[pixel]
			if union[pixel] {
				unionArea++
			}
		}
		differences[index] = FramePairDifference{
			AppearanceMSE:            foregroundMSE(left, right, &union, unionArea),
			ForegroundMaskDifference: foregroundMaskDifference(left, right, unionArea),
		}
	}
	return differences, nil
}

func buildFrameSequenceAnalysis(analyses []frameAnalysis, fps int) FrameSequenceAnalysis {
	descriptors := make([]frameDescriptor, len(analyses))
	observations := make([]FrameObservation, len(analyses))
	var union [analysisSize * analysisSize]bool
	for index, analysis := range analyses {
		descriptor := analysis.descriptor
		descriptors[index] = descriptor
		observations[index] = FrameObservation{
			Safe: analysis.safe, CentroidX: descriptor.cx, CentroidY: descriptor.cy,
			Width: descriptor.width, Height: descriptor.height, ForegroundArea: descriptor.foreground,
		}
		for pixel, visible := range descriptor.mask {
			union[pixel] = union[pixel] || visible
		}
	}
	unionArea := 0
	for _, visible := range union {
		if visible {
			unionArea++
		}
	}
	pairwiseMSE := make([][]float64, len(descriptors))
	for left := range pairwiseMSE {
		pairwiseMSE[left] = make([]float64, len(descriptors))
		for right := range left {
			pairwiseMSE[left][right] = foregroundMSE(descriptors[left], descriptors[right], &union, unionArea)
			pairwiseMSE[right][left] = pairwiseMSE[left][right]
		}
	}
	return FrameSequenceAnalysis{
		FPS: fps, Frames: observations, PairwiseMSE: pairwiseMSE,
		ForegroundRatio: float64(unionArea) / float64(len(union)),
	}
}

func describeFrame(source image.Image, chromaKey ChromaKey) frameDescriptor {
	bounds := source.Bounds()
	var descriptor frameDescriptor
	var sumX, sumY float64
	minX, maxX := analysisSize, -1
	minY, maxY := analysisSize, -1
	for y := range analysisSize {
		for x := range analysisSize {
			sourceX := bounds.Min.X + minInt(bounds.Dx()-1, int((float64(x)+.5)*float64(bounds.Dx())/analysisSize))
			sourceY := bounds.Min.Y + minInt(bounds.Dy()-1, int((float64(y)+.5)*float64(bounds.Dy())/analysisSize))
			value := color.NRGBAModel.Convert(source.At(sourceX, sourceY)).(color.NRGBA)
			if chromaKey.matches(value) {
				continue
			}
			pixel := y*analysisSize + x
			descriptor.mask[pixel] = true
			descriptor.gray[pixel] = (.299*float64(value.R) + .587*float64(value.G) + .114*float64(value.B)) / 255
			descriptor.foreground++
			sumX += float64(x)
			sumY += float64(y)
			minX, maxX = minInt(minX, x), maxInt(maxX, x)
			minY, maxY = minInt(minY, y), maxInt(maxY, y)
		}
	}
	if descriptor.foreground > 0 {
		descriptor.cx = sumX / float64(descriptor.foreground)
		descriptor.cy = sumY / float64(descriptor.foreground)
		descriptor.width = float64(maxX - minX + 1)
		descriptor.height = float64(maxY - minY + 1)
	} else {
		descriptor.cx, descriptor.cy = math.NaN(), math.NaN()
	}
	return descriptor
}

func foregroundMaskDifference(a, b frameDescriptor, unionArea int) float64 {
	if unionArea == 0 {
		return 0
	}
	different := 0
	for index := range a.mask {
		if a.mask[index] != b.mask[index] {
			different++
		}
	}
	return float64(different) / float64(unionArea)
}

func foregroundMSE(a, b frameDescriptor, union *[analysisSize * analysisSize]bool, unionArea int) float64 {
	if unionArea == 0 {
		return 1
	}
	var sum float64
	for index, include := range union {
		if include {
			delta := a.gray[index] - b.gray[index]
			sum += delta * delta
		}
	}
	return sum / float64(unionArea)
}
