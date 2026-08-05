package image

import (
	"image"
	"image/color"
	"testing"
)

func TestNormalizeAnimationImageStabilizesFramesWithSharedCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	// The same body is displaced differently in each generated cell.
	fillRect(src, image.Rect(7, 8, 19, 34), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	fillRect(src, image.Rect(57, 4, 69, 30), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	// A moving arm changes the silhouette but must not become a per-frame crop.
	fillRect(src, image.Rect(19, 14, 28, 19), color.NRGBA{R: 220, G: 70, B: 40, A: 255})
	fillRect(src, image.Rect(48, 11, 57, 16), color.NRGBA{R: 220, G: 70, B: 40, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 48, FrameHeight: 48, Margin: 3,
	})
	if err != nil {
		t.Fatalf("normalize animation: %v", err)
	}
	if len(result.Frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(result.Frames))
	}
	if result.Report.GridPolicy != "proportional_fixed_cells" {
		t.Fatalf("grid policy = %q", result.Report.GridPolicy)
	}
	if result.Report.OutputAnchorRange.X > 2 || result.Report.OutputAnchorRange.Y > 2 {
		t.Fatalf("output anchors still drift: %+v", result.Report.OutputAnchorRange)
	}
	if result.Frames[0].Translation == (AnimationOffset{}) || result.Frames[1].Translation == (AnimationOffset{}) {
		t.Fatalf("expected generated displacement to be corrected: %+v %+v", result.Frames[0].Translation, result.Frames[1].Translation)
	}
	for i, frame := range result.Frames {
		decoded, err := DecodeBase64Image(frame.ImageBase64)
		if err != nil {
			t.Fatalf("decode frame %d: %v", i, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(48, 48) {
			t.Fatalf("frame %d size = %v, want 48x48", i, got)
		}
	}
	if result.Report.SharedCrop.Width <= 0 || result.Report.SharedCrop.Height <= 0 {
		t.Fatalf("invalid shared crop: %+v", result.Report.SharedCrop)
	}
}

func TestNormalizeAnimationImagePreservesRequestedVerticalMotion(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 60, 40))
	fillRect(src, image.Rect(9, 14, 21, 36), color.NRGBA{B: 255, A: 255})
	fillRect(src, image.Rect(39, 4, 51, 26), color.NRGBA{B: 255, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 40, FrameHeight: 40, PreserveVerticalMotion: true,
	})
	if err != nil {
		t.Fatalf("normalize animation: %v", err)
	}
	if result.Frames[0].Translation.Y != 0 || result.Frames[1].Translation.Y != 0 {
		t.Fatalf("vertical translations should remain disabled: %+v %+v", result.Frames[0].Translation, result.Frames[1].Translation)
	}
	if result.Report.OutputAnchorRange.Y <= 1 {
		t.Fatalf("expected vertical motion to remain, range=%+v", result.Report.OutputAnchorRange)
	}
}

func TestNormalizeAnimationImageCanRemoveGeneratedFlatBackground(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	fillRect(src, src.Bounds(), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(5, 3, 15, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})
	fillRect(src, image.Rect(25, 3, 35, 18), color.NRGBA{R: 230, G: 40, B: 30, A: 255})

	result, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
		FrameWidth: 24, FrameHeight: 24,
		Background: &AnimationBackgroundOptions{MatteColor: "#00ff00", Material: MaterialFlatIcon},
	})
	if err != nil {
		t.Fatalf("normalize green-screen animation: %v", err)
	}
	if result.Report.BackgroundRemovalReport == nil {
		t.Fatal("expected background removal report")
	}
	for i, frame := range result.Frames {
		if frame.ContentBounds == nil {
			t.Fatalf("frame %d has no visible content", i)
		}
	}
}

func TestNormalizeAnimationImageRejectsOpaqueSourceWithoutBackgroundOptions(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	fillRect(src, src.Bounds(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	_, err := normalizeAnimationImage(src, normalizeAnimationRequest{
		Columns: 2, Rows: 1,
	})
	if err == nil {
		t.Fatal("expected opaque-source error")
	}
}
