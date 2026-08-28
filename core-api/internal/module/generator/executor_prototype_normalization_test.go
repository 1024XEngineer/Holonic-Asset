package generator

import (
	"context"
	"errors"
	"strings"
	"testing"

	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

var errPrototypeNormalizerNotImplemented = errors.New("prototype normalizer stub method was called")

type prototypeNormalizerProcessor struct {
	flipResult      *imageprocessor.FlipHorizontalResult
	flipErr         error
	normalizeResult *imageprocessor.NormalizeReferenceResult
	normalizeErr    error
}

func (p *prototypeNormalizerProcessor) RemoveBackground(context.Context, *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (p *prototypeNormalizerProcessor) NormalizeReference(context.Context, *imageprocessor.NormalizeReferenceRequest) (*imageprocessor.NormalizeReferenceResult, error) {
	if p.normalizeResult != nil || p.normalizeErr != nil {
		return p.normalizeResult, p.normalizeErr
	}
	return nil, errPrototypeNormalizerNotImplemented
}

func (p *prototypeNormalizerProcessor) Resize(context.Context, *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (p *prototypeNormalizerProcessor) Verify(context.Context, *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (p *prototypeNormalizerProcessor) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (p *prototypeNormalizerProcessor) FlipHorizontal(context.Context, *imageprocessor.FlipHorizontalRequest) (*imageprocessor.FlipHorizontalResult, error) {
	return p.flipResult, p.flipErr
}

type prototypeProcessorWithoutFlip struct{}

func (prototypeProcessorWithoutFlip) RemoveBackground(context.Context, *imageprocessor.RemoveBackgroundRequest) (*imageprocessor.RemoveBackgroundResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (prototypeProcessorWithoutFlip) NormalizeReference(context.Context, *imageprocessor.NormalizeReferenceRequest) (*imageprocessor.NormalizeReferenceResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (prototypeProcessorWithoutFlip) Resize(context.Context, *imageprocessor.ResizeRequest) (*imageprocessor.ResizeResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (prototypeProcessorWithoutFlip) Verify(context.Context, *imageprocessor.VerifyRequest) (*imageprocessor.VerificationReport, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func (prototypeProcessorWithoutFlip) SplitImage(context.Context, *imageprocessor.SplitImageRequest) (*imageprocessor.SplitImageResult, error) {
	return nil, errPrototypeNormalizerNotImplemented
}

func TestNormalizePrototypeReferencePreservesExistingDataURL(t *testing.T) {
	t.Parallel()

	const reference = "data:image/png;base64,cG5n"
	executor := &executor{processor: &prototypeNormalizerProcessor{
		normalizeResult: &imageprocessor.NormalizeReferenceResult{
			ImageBase64: reference,
			MIMEType:    "image/png",
			Report:      imageprocessor.ReferenceNormalizationReport{Scale: 1},
		},
	}}

	got, err := executor.normalizePrototypeReference(context.Background(), reference)
	if err != nil {
		t.Fatalf("normalize existing data URL: %v", err)
	}
	if got != reference {
		t.Fatalf("normalized reference = %q, want %q", got, reference)
	}
}

func TestNormalizeSideOnRegions(t *testing.T) {
	t.Parallel()

	baseRegions := func() []imageprocessor.ImageRegion {
		return []imageprocessor.ImageRegion{
			{Index: 7, ImageBase64: "source-left", MIMEType: "image/original"},
			{Index: 8, ImageBase64: "source-right", MIMEType: "image/png"},
		}
	}

	tests := []struct {
		name        string
		regions     []imageprocessor.ImageRegion
		perspective assetdomain.Perspective
		processor   imageprocessor.Processor
		wantText    string
		wantLeft    *imageprocessor.ImageRegion
	}{
		{
			name:        "non side-on is unchanged",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveTopDown,
			processor:   &prototypeNormalizerProcessor{},
			wantLeft:    &imageprocessor.ImageRegion{Index: 7, ImageBase64: "source-left", MIMEType: "image/original"},
		},
		{
			name:        "mirrors canonical right direction",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor: &prototypeNormalizerProcessor{flipResult: &imageprocessor.FlipHorizontalResult{
				ImageBase64: "mirrored-right",
				MIMEType:    "image/png",
			}},
			wantLeft: &imageprocessor.ImageRegion{Index: 0, ImageBase64: "mirrored-right", MIMEType: "image/png"},
		},
		{
			name:        "preserves original MIME type when mirror omits it",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor: &prototypeNormalizerProcessor{flipResult: &imageprocessor.FlipHorizontalResult{
				ImageBase64: "mirrored-right",
			}},
			wantLeft: &imageprocessor.ImageRegion{Index: 0, ImageBase64: "mirrored-right", MIMEType: "image/original"},
		},
		{
			name:        "rejects missing horizontal flipper",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor:   prototypeProcessorWithoutFlip{},
			wantText:    "horizontal flip is unavailable",
		},
		{
			name:        "rejects wrong region count",
			regions:     baseRegions()[:1],
			perspective: assetdomain.PerspectiveSideOn,
			processor:   &prototypeNormalizerProcessor{},
			wantText:    "got 1 regions, want 2",
		},
		{
			name: "rejects empty canonical right direction",
			regions: []imageprocessor.ImageRegion{
				{Index: 0, ImageBase64: "source-left"},
				{Index: 1, MIMEType: "image/png"},
			},
			perspective: assetdomain.PerspectiveSideOn,
			processor:   &prototypeNormalizerProcessor{},
			wantText:    "right direction is empty",
		},
		{
			name:        "wraps mirror failure",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor: &prototypeNormalizerProcessor{
				flipErr: errors.New("mirror failed"),
			},
			wantText: "mirror right direction: mirror failed",
		},
		{
			name:        "rejects nil mirror result",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor:   &prototypeNormalizerProcessor{},
			wantText:    "empty mirrored left direction",
		},
		{
			name:        "rejects empty mirror result",
			regions:     baseRegions(),
			perspective: assetdomain.PerspectiveSideOn,
			processor: &prototypeNormalizerProcessor{flipResult: &imageprocessor.FlipHorizontalResult{
				MIMEType: "image/png",
			}},
			wantText: "empty mirrored left direction",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &executor{processor: test.processor}
			got, err := executor.normalizeSideOnRegions(context.Background(), test.regions, test.perspective, GenerateCharacterProtoType)
			if test.wantText != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantText) {
					t.Fatalf("error = %v, want text %q", err, test.wantText)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize regions: %v", err)
			}
			if got[0] != *test.wantLeft {
				t.Fatalf("left region = %+v, want %+v", got[0], *test.wantLeft)
			}
			if test.perspective == assetdomain.PerspectiveSideOn && (got[1].Index != 1 || got[1].ImageBase64 != "source-right") {
				t.Fatalf("right region changed unexpectedly: %+v", got[1])
			}
		})
	}
}
