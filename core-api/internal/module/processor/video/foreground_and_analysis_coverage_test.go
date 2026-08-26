package video

import (
	"errors"
	"image"
	"testing"
)

func TestValidateSelectedFrameBounds(t *testing.T) {
	t.Parallel()

	valid := testGreenFrame(96, 96)
	drawSubject(valid, image.Rect(30, 20, 66, 88))
	empty := testGreenFrame(96, 96)
	clipped := testGreenFrame(96, 96)
	drawSubject(clipped, image.Rect(0, 20, 36, 88))

	// 1. len(frames) != len(sourceIndices)
	if err := validateSelectedFrameBounds([]image.Image{valid}, []int{0, 1}, testGreenChromaKey); err == nil {
		t.Fatal("expected error when frames and indices length mismatch")
	}

	// 2. Missing foreground
	errEmpty := validateSelectedFrameBounds([]image.Image{empty}, []int{0}, testGreenChromaKey)
	var qErrEmpty *QualityError
	if !errors.As(errEmpty, &qErrEmpty) || qErrEmpty.Kind != "foreground" {
		t.Fatalf("expected foreground quality error, got %v", errEmpty)
	}

	// 3. Clipped foreground enters safety band
	errClipped := validateSelectedFrameBounds([]image.Image{clipped}, []int{0}, testGreenChromaKey)
	var qErrClipped *QualityError
	if !errors.As(errClipped, &qErrClipped) || qErrClipped.Kind != "framing" {
		t.Fatalf("expected framing quality error, got %v", errClipped)
	}

	// 4. Valid frames
	if err := validateSelectedFrameBounds([]image.Image{valid}, []int{0}, testGreenChromaKey); err != nil {
		t.Fatalf("expected valid frames to pass: %v", err)
	}
}

func TestRGBToOpenCVHSVBranches(t *testing.T) {
	t.Parallel()

	// 1. Black (maximum == 0)
	h, s, v := rgbToOpenCVHSV(0, 0, 0)
	if h != 0 || s != 0 || v != 0 {
		t.Fatalf("black HSV = (%d,%d,%d), want (0,0,0)", h, s, v)
	}

	// 2. Blue dominant (maximum == blue)
	hB, sB, vB := rgbToOpenCVHSV(0, 0, 255)
	if hB < 110 || hB > 130 || sB != 255 || vB != 255 {
		t.Fatalf("pure blue HSV = (%d,%d,%d)", hB, sB, vB)
	}

	// 3. Magenta (red dominant with blue > green -> negative hue offset before wrap)
	hM, _, _ := rgbToOpenCVHSV(255, 0, 200)
	if hM < 140 {
		t.Fatalf("magenta hue = %d", hM)
	}
}

func TestAnalyzeFrameSequenceValidationAndAutoDetect(t *testing.T) {
	t.Parallel()

	valid := testGreenFrame(96, 96)
	drawSubject(valid, image.Rect(30, 20, 66, 88))

	// 1. Empty frames
	if _, err := AnalyzeFrameSequence(nil, 12, testGreenChromaKey); err == nil {
		t.Fatal("expected error on empty frames")
	}

	// 2. Negative FPS
	if _, err := AnalyzeFrameSequence([]image.Image{valid}, -1, testGreenChromaKey); err == nil {
		t.Fatal("expected error on negative FPS")
	}

	// 3. Invalid chroma key
	invalidKey := ChromaKey{HueMin: 100, HueMax: 50}
	if _, err := AnalyzeFrameSequence([]image.Image{valid}, 12, invalidKey); err == nil {
		t.Fatal("expected error on invalid chroma key")
	}

	// 4. Frame is nil or empty bounds
	if _, err := AnalyzeFrameSequence([]image.Image{nil}, 12, testGreenChromaKey); err == nil {
		t.Fatal("expected error on nil frame in sequence")
	}
	emptyBoundsImg := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if _, err := AnalyzeFrameSequence([]image.Image{emptyBoundsImg}, 12, testGreenChromaKey); err == nil {
		t.Fatal("expected error on empty bounds frame in sequence")
	}

	// 5. AutoDetect ChromaKey
	autoKey := testGreenChromaKey
	autoKey.AutoDetect = true
	analysis, err := AnalyzeFrameSequence([]image.Image{valid, valid}, 12, autoKey)
	if err != nil {
		t.Fatalf("AnalyzeFrameSequence with auto-detect: %v", err)
	}
	if analysis.FPS != 12 || len(analysis.Frames) != 2 {
		t.Fatalf("unexpected sequence analysis: %#v", analysis)
	}
}

func TestAnalyzeFramePairsValidationAndAutoDetect(t *testing.T) {
	t.Parallel()

	valid := testGreenFrame(96, 96)
	drawSubject(valid, image.Rect(30, 20, 66, 88))
	other := testGreenFrame(96, 96)
	drawSubject(other, image.Rect(32, 22, 68, 90))

	// 1. Empty original
	if _, err := AnalyzeFramePairs(nil, []image.Image{valid}, testGreenChromaKey); err == nil {
		t.Fatal("expected error on empty original")
	}

	// 2. Length mismatch
	if _, err := AnalyzeFramePairs([]image.Image{valid}, []image.Image{valid, other}, testGreenChromaKey); err == nil {
		t.Fatal("expected error on length mismatch")
	}

	// 3. Invalid chroma key
	invalidKey := ChromaKey{HueMin: 100, HueMax: 50}
	if _, err := AnalyzeFramePairs([]image.Image{valid}, []image.Image{valid}, invalidKey); err == nil {
		t.Fatal("expected error on invalid chroma key")
	}

	// 4. Nil or empty images
	if _, err := AnalyzeFramePairs([]image.Image{nil}, []image.Image{valid}, testGreenChromaKey); err == nil {
		t.Fatal("expected error on nil original frame")
	}
	if _, err := AnalyzeFramePairs([]image.Image{valid}, []image.Image{nil}, testGreenChromaKey); err == nil {
		t.Fatal("expected error on nil candidate frame")
	}

	// 5. AutoDetect mode
	autoKey := testGreenChromaKey
	autoKey.AutoDetect = true
	diffs, err := AnalyzeFramePairs([]image.Image{valid}, []image.Image{other}, autoKey)
	if err != nil || len(diffs) != 1 {
		t.Fatalf("AnalyzeFramePairs auto-detect: %v, diffs: %#v", err, diffs)
	}

	// 6. Zero foreground for both frames (unionArea == 0)
	empty1 := testGreenFrame(96, 96)
	empty2 := testGreenFrame(96, 96)
	diffsZero, err := AnalyzeFramePairs([]image.Image{empty1}, []image.Image{empty2}, testGreenChromaKey)
	if err != nil || len(diffsZero) != 1 {
		t.Fatalf("AnalyzeFramePairs zero union: %v", err)
	}
	if diffsZero[0].ForegroundMaskDifference != 0 || diffsZero[0].AppearanceMSE != 1 {
		t.Fatalf("unexpected zero-union differences: %#v", diffsZero[0])
	}
}
