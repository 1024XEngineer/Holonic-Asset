package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestProcessorSplitImageGrid(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	fillRect(src, image.Rect(6, 2, 14, 8), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(26, 2, 34, 8), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(6, 12, 14, 18), color.NRGBA{B: 255, A: 255})
	fillRect(src, image.Rect(26, 12, 34, 18), color.NRGBA{R: 255, G: 255, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src), Mode: ImageSplitModeGrid, Columns: 2, Rows: 2,
	})
	if err != nil {
		t.Fatalf("split grid: %v", err)
	}
	if result.Mode != ImageSplitModeGrid || len(result.Regions) != 4 {
		t.Fatalf("unexpected result: mode=%q regions=%d", result.Mode, len(result.Regions))
	}
	for index, region := range result.Regions {
		decoded, err := DecodeBase64Image(region.ImageBase64)
		if err != nil {
			t.Fatalf("decode region %d: %v", index, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(20, 10) {
			t.Errorf("region %d size = %v, want 20x10", index, got)
		}
		if region.MIMEType != pngMIMEType {
			t.Errorf("region %d MIME type = %q", index, region.MIMEType)
		}
	}
	if got, want := result.Regions[0].SourceBounds, (AlphaBoundingBox{Width: 20, Height: 10}); got != want {
		t.Errorf("first source bounds = %+v, want %+v", got, want)
	}
	if got := result.Regions[0].ContentBounds; got == nil || got.Width != 8 || got.Height != 6 {
		t.Errorf("first content bounds = %+v, want 8x6", got)
	}
}

func TestProcessorSplitImageComponentsCropsIndependentImages(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 60, 30))
	fillRect(src, image.Rect(4, 5, 14, 17), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(39, 9, 54, 24), color.NRGBA{G: 255, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src), Mode: ImageSplitModeComponents,
	})
	if err != nil {
		t.Fatalf("split components: %v", err)
	}
	if len(result.Regions) != 2 {
		t.Fatalf("got %d components, want 2", len(result.Regions))
	}
	wantBounds := []AlphaBoundingBox{{X: 4, Y: 5, Width: 10, Height: 12}, {X: 39, Y: 9, Width: 15, Height: 15}}
	for i, region := range result.Regions {
		if region.SourceBounds != wantBounds[i] {
			t.Errorf("region %d source bounds = %+v, want %+v", i, region.SourceBounds, wantBounds[i])
		}
		decoded, err := DecodeBase64Image(region.ImageBase64)
		if err != nil {
			t.Fatalf("decode component %d: %v", i, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(wantBounds[i].Width, wantBounds[i].Height) {
			t.Errorf("component %d size = %v, want %dx%d", i, got, wantBounds[i].Width, wantBounds[i].Height)
		}
	}
}

func TestProcessorSplitImageProjectionGroupsDisconnectedParts(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 60, 40))
	// Each pose has two disconnected pieces, but the pieces share one x/y band.
	fillRect(src, image.Rect(4, 4, 12, 12), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(14, 14, 20, 20), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(34, 4, 42, 12), color.NRGBA{G: 255, A: 255})
	fillRect(src, image.Rect(44, 14, 50, 20), color.NRGBA{G: 255, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src), Mode: ImageSplitModeProjection,
	})
	if err != nil {
		t.Fatalf("split projection: %v", err)
	}
	if len(result.Regions) != 2 {
		t.Fatalf("got %d projection regions, want 2", len(result.Regions))
	}
	if got := result.Regions[0].SourceBounds; got != (AlphaBoundingBox{X: 4, Y: 4, Width: 16, Height: 16}) {
		t.Errorf("first projection bounds = %+v", got)
	}
	if got := result.Regions[1].SourceBounds; got != (AlphaBoundingBox{X: 34, Y: 4, Width: 16, Height: 16}) {
		t.Errorf("second projection bounds = %+v", got)
	}
}

func TestProcessorSplitImageRejectsEmptyGridRegion(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	fillRect(src, image.Rect(2, 2, 8, 8), color.NRGBA{A: 255})
	request := &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src), Mode: ImageSplitModeGrid,
		Columns: 2, Rows: 1, ForceProportionalGrid: true,
	}
	if _, err := NewProcessor().SplitImage(context.Background(), request); err == nil || !strings.Contains(err.Error(), "region 1 is empty") {
		t.Fatalf("expected empty-region error, got %v", err)
	}
	request.AllowEmptyRegions = true
	result, err := NewProcessor().SplitImage(context.Background(), request)
	if err != nil {
		t.Fatalf("allow empty region: %v", err)
	}
	if len(result.Regions) != 2 {
		t.Fatalf("got %d regions, want 2", len(result.Regions))
	}
	if result.Regions[1].ContentBounds != nil {
		t.Errorf("empty region content bounds = %+v, want nil", result.Regions[1].ContentBounds)
	}
}

func TestProcessorSplitImageHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewProcessor().SplitImage(ctx, &SplitImageRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func encodeImageForTest(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestProcessorSplitImageGridUsesFixedProportionalCellsByDefault(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 20))
	// Deliberately place the subjects off-centre. Content projections would put
	// the boundary at x=40 and make the two returned frames use different source
	// coordinates; a known animation grid must stay at x=50.
	fillRect(src, image.Rect(4, 3, 16, 17), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(64, 3, 76, 17), color.NRGBA{G: 255, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src), Mode: ImageSplitModeGrid, Columns: 2, Rows: 1,
	})
	if err != nil {
		t.Fatalf("split proportional grid: %v", err)
	}
	if got := result.Regions[0].SourceBounds; got != (AlphaBoundingBox{Width: 50, Height: 20}) {
		t.Fatalf("first source bounds = %+v, want fixed 50x20 cell", got)
	}
	if got := result.Regions[1].SourceBounds; got != (AlphaBoundingBox{X: 50, Width: 50, Height: 20}) {
		t.Fatalf("second source bounds = %+v, want fixed cell starting at x=50", got)
	}
}

func TestProcessorSplitImageAnimationReturnsStabilizedFrames(t *testing.T) {
	matte := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	src := image.NewNRGBA(image.Rect(0, 0, 80, 50))
	fillRect(src, src.Bounds(), matte)
	// The same subject is deliberately displaced between two fixed grid cells.
	fillRect(src, image.Rect(5, 8, 17, 42), color.NRGBA{R: 210, G: 55, B: 45, A: 255})
	fillRect(src, image.Rect(60, 3, 72, 37), color.NRGBA{R: 210, G: 55, B: 45, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src),
		Mode:        ImageSplitModeAnimation,
		Columns:     2,
		Rows:        1,
		FrameWidth:  64,
		FrameHeight: 64,
		Anchor:      AnimationAnchorFeet,
	})
	if err != nil {
		t.Fatalf("split animation: %v", err)
	}
	if result.Mode != ImageSplitModeAnimation || len(result.Regions) != 2 {
		t.Fatalf("unexpected result: mode=%q regions=%d", result.Mode, len(result.Regions))
	}
	if result.AnimationReport == nil {
		t.Fatal("animation report is nil")
	}
	if result.AnimationReport.BackgroundRemovalReport == nil {
		t.Fatal("opaque animation input should use automatic background removal")
	}
	if result.AnimationReport.SourceAnchorRange.X < 10 || result.AnimationReport.SourceAnchorRange.Y < 4 {
		t.Fatalf("source anchor range = %+v, want deliberately displaced source", result.AnimationReport.SourceAnchorRange)
	}
	if result.AnimationReport.OutputAnchorRange.X > 1 || result.AnimationReport.OutputAnchorRange.Y > 1 {
		t.Fatalf("output anchor range = %+v, want at most one pixel", result.AnimationReport.OutputAnchorRange)
	}
	if result.ImageBase64 == "" || result.MIMEType != pngMIMEType {
		t.Fatal("normalized spritesheet was not returned")
	}
	if result.OutputWidth != 128 || result.OutputHeight != 64 {
		t.Fatalf("output sheet = %dx%d, want 128x64", result.OutputWidth, result.OutputHeight)
	}
	for index, region := range result.Regions {
		decoded, err := DecodeBase64Image(region.ImageBase64)
		if err != nil {
			t.Fatalf("decode animation region %d: %v", index, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(64, 64) {
			t.Errorf("animation region %d size = %v, want 64x64", index, got)
		}
		if region.OutputAnchor == nil || region.Translation == nil {
			t.Errorf("animation region %d is missing stabilization metadata", index)
		}
	}
}

func TestProcessorSplitImageKnownGridDefaultsToAnimation(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	fillRect(src, image.Rect(5, 5, 17, 35), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(60, 3, 72, 33), color.NRGBA{G: 255, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src),
		Columns:     2,
		Rows:        1,
	})
	if err != nil {
		t.Fatalf("split default animation: %v", err)
	}
	if result.Mode != ImageSplitModeAnimation || result.AnimationReport == nil {
		t.Fatalf("known grid default mode = %q report=%v, want animation", result.Mode, result.AnimationReport)
	}
	if result.AnimationReport.OutputAnchorRange.X != 0 || result.AnimationReport.OutputAnchorRange.Y != 0 {
		t.Fatalf("default animation output anchor range = %+v, want zero", result.AnimationReport.OutputAnchorRange)
	}
}

func TestProcessorSplitImageAnimationRejectsIndependentContentCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	fillRect(src, image.Rect(4, 3, 16, 17), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(24, 3, 36, 17), color.NRGBA{G: 255, A: 255})

	_, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64:   encodeImageForTest(t, src),
		Mode:          ImageSplitModeAnimation,
		Columns:       2,
		Rows:          1,
		CropToContent: true,
	})
	if err == nil || !strings.Contains(err.Error(), "shared crop") {
		t.Fatalf("expected shared-crop validation error, got %v", err)
	}
}

func TestProcessorSplitImageProjectionUsesConfiguredMergeGap(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 200, 80))
	// Two visual groups. Each group has a body and a detached nearby part.
	fillRect(src, image.Rect(10, 10, 30, 65), color.NRGBA{R: 255, A: 255})
	fillRect(src, image.Rect(35, 25, 43, 52), color.NRGBA{R: 255, G: 180, A: 255})
	fillRect(src, image.Rect(105, 10, 125, 65), color.NRGBA{B: 255, A: 255})
	fillRect(src, image.Rect(130, 25, 138, 52), color.NRGBA{G: 180, B: 255, A: 255})

	request := &SplitImageRequest{
		ImageBase64:        encodeImageForTest(t, src),
		Mode:               ImageSplitModeProjection,
		MinBandSize:        2,
		ProjectionMergeGap: 10,
	}
	result, err := NewProcessor().SplitImage(context.Background(), request)
	if err != nil {
		t.Fatalf("split projection with narrow gap: %v", err)
	}
	if len(result.Regions) != 2 {
		t.Fatalf("narrow merge gap regions = %d, want 2 groups", len(result.Regions))
	}

	request.ProjectionMergeGap = 40
	result, err = NewProcessor().SplitImage(context.Background(), request)
	if err != nil {
		t.Fatalf("split projection with wide gap: %v", err)
	}
	if len(result.Regions) != 1 {
		t.Fatalf("wide merge gap regions = %d, want 1 merged group", len(result.Regions))
	}
}
