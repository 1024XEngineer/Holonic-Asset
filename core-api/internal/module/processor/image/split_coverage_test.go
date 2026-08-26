package image

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestSplitImageValidationErrors(t *testing.T) {
	t.Parallel()

	p := NewProcessor()
	ctx := context.Background()

	// 1. Canceled context
	cancCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err := p.SplitImage(cancCtx, &SplitImageRequest{})
	if err == nil {
		t.Fatal("expected error on canceled context")
	}

	// 2. Nil request
	_, err = p.SplitImage(ctx, nil)
	if err == nil {
		t.Fatal("expected error on nil request")
	}

	// 3. Invalid Base64 image
	_, err = p.SplitImage(ctx, &SplitImageRequest{ImageBase64: "not-valid-base64!"})
	if err == nil {
		t.Fatal("expected error on invalid base64")
	}

	// 4. splitImage with nil src
	_, err = splitImage(nil, SplitImageRequest{})
	if err == nil {
		t.Fatal("expected error on nil source image")
	}

	validImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(validImg, validImg.Bounds(), color.NRGBA{R: 200, G: 100, B: 50, A: 255})

	// 5. Unsupported mode
	_, err = splitImage(validImg, SplitImageRequest{Mode: "unsupported-mode"})
	if err == nil {
		t.Fatal("expected error on unsupported mode")
	}

	// 6. Negative validation options
	negativeCases := []SplitImageRequest{
		{MinComponentPixels: -1},
		{MinBandSize: -1},
		{ProjectionMergeGap: -1},
		{GridBoundaryMargin: -1},
	}
	for i, req := range negativeCases {
		_, err := splitImage(validImg, req)
		if err == nil {
			t.Fatalf("case %d: expected error on negative constraint: %#v", i, req)
		}
	}

	// 7. Default mode inference: Columns > 0 -> Animation
	animImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(animImg, animImg.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(animImg, image.Rect(4, 4, 12, 12), color.NRGBA{R: 255, A: 255})
	fillRect(animImg, image.Rect(20, 4, 28, 12), color.NRGBA{R: 255, A: 255})
	fillRect(animImg, image.Rect(4, 20, 12, 28), color.NRGBA{R: 255, A: 255})
	fillRect(animImg, image.Rect(20, 20, 28, 28), color.NRGBA{R: 255, A: 255})

	resAnim, err := splitImage(animImg, SplitImageRequest{Columns: 2, Rows: 2})
	if err != nil {
		t.Fatalf("splitImage with default columns: %v", err)
	}
	if len(resAnim.Regions) != 4 {
		t.Fatalf("expected 4 regions for 2x2 animation split, got %d", len(resAnim.Regions))
	}
}

func TestDetectGridBoundsAndBoundariesFromRuns(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	// Draw 2 separate rectangles along x axis
	fillRect(img, image.Rect(4, 4, 28, 60), color.NRGBA{R: 255, A: 255})
	fillRect(img, image.Rect(36, 4, 60, 60), color.NRGBA{R: 255, A: 255})

	// forceProportional = true
	xB, yB := detectGridBoundsNRGBA(img, 2, 1, true, 10)
	if len(xB) != 3 || len(yB) != 2 {
		t.Fatalf("unexpected proportional bounds lengths: %v, %v", xB, yB)
	}

	// detectGridBoundsNRGBA with forceProportional = false
	xB2, yB2 := detectGridBoundsNRGBA(img, 2, 1, false, 10)
	if len(xB2) != 3 || len(yB2) != 2 {
		t.Fatalf("unexpected detected bounds lengths: %v, %v", xB2, yB2)
	}

	// gridRegions with detectBounds = true
	regions := gridRegions(img, 2, 1, true, 10)
	if len(regions) != 2 {
		t.Fatalf("expected 2 grid regions, got %d", len(regions))
	}

	// boundariesFromRuns edge cases
	// 1. len(runs) != expected
	if _, ok := boundariesFromRuns([]imageRun{{0, 10}}, 2, 64); ok {
		t.Fatal("expected false when runs length != expected")
	}
	// 2. expected == 1
	if bounds, ok := boundariesFromRuns([]imageRun{{0, 10}}, 1, 64); !ok || len(bounds) != 2 || bounds[0] != 0 || bounds[1] != 64 {
		t.Fatalf("expected [0, 64], got %v", bounds)
	}
	// 3. runs resulting in invalid median <= 0 (e.g. same center)
	sameCenterRuns := []imageRun{{10, 20}, {10, 20}}
	if _, ok := boundariesFromRuns(sameCenterRuns, 2, 64); ok {
		t.Fatal("expected false for runs with median distance <= 0")
	}
}

func TestProjectionRegionsAndRunsCoverage(t *testing.T) {
	t.Parallel()

	// 1. projectionRuns branches
	counts := []int{0, 0, 5, 5, 5, 0, 0, 0, 5, 5, 5, 0}
	runs := projectionRuns(counts, 2, 1, 2)
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d: %v", len(runs), runs)
	}

	// counts with trailing active run
	trailingCounts := []int{0, 5, 5}
	trailingRuns := projectionRuns(trailingCounts, 2, 1, 1)
	if len(trailingRuns) != 1 || trailingRuns[0].End != 3 {
		t.Fatalf("expected trailing run ending at 3, got %v", trailingRuns)
	}

	// 2. medianFloat64 odd and even
	if m := medianFloat64([]float64{1.0, 3.0, 2.0}); m != 2.0 {
		t.Fatalf("odd median = %f, want 2.0", m)
	}
	if m := medianFloat64([]float64{1.0, 2.0, 4.0, 3.0}); m != 2.5 {
		t.Fatalf("even median = %f, want 2.5", m)
	}

	// 3. projectionRegions with < 2 components
	singleImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(singleImg, image.Rect(8, 8, 24, 24), color.NRGBA{R: 255, A: 255})
	pSingle := projectionRegions(singleImg, 10, 2, 0)
	if len(pSingle) != 1 {
		t.Fatalf("expected 1 region for single component, got %d", len(pSingle))
	}

	// 4. projectionRegions with 2 nearby components that merge with default gap
	mergeImg := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	// Two components separated by a 2-pixel transparent gap (e.g. x: 10..18 and x: 20..28)
	fillRect(mergeImg, image.Rect(10, 10, 18, 20), color.NRGBA{R: 255, A: 255})
	fillRect(mergeImg, image.Rect(20, 10, 28, 20), color.NRGBA{R: 255, A: 255})
	// A far component at (45, 45) to (55, 55)
	fillRect(mergeImg, image.Rect(45, 45, 55, 55), color.NRGBA{R: 255, A: 255})

	pMerged := projectionRegions(mergeImg, 10, 1, 0)
	if len(pMerged) != 2 {
		t.Fatalf("expected 2 merged projection regions, got %d", len(pMerged))
	}
}
