package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/draw"
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

func TestProcessorSplitImageGridReassemblesWithoutBoundaryLoss(t *testing.T) {
	const tileSize = 16
	src := image.NewNRGBA(image.Rect(0, 0, 3*tileSize, 3*tileSize))
	for y := range src.Bounds().Dy() {
		for x := range src.Bounds().Dx() {
			if (x+y)%13 == 0 {
				continue
			}
			src.SetNRGBA(x, y, color.NRGBA{
				R: uint8(40 + x*3),
				G: uint8(30 + y*4),
				B: uint8(20 + (x+y)*2),
				A: 255,
			})
		}
	}

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src),
		Mode:        ImageSplitModeGrid,
		Columns:     3,
		Rows:        3,
	})
	if err != nil {
		t.Fatalf("split grid: %v", err)
	}
	if len(result.Regions) != 9 {
		t.Fatalf("got %d regions, want 9", len(result.Regions))
	}

	reassembled := image.NewNRGBA(src.Bounds())
	for index, region := range result.Regions {
		decoded, decodeErr := DecodeBase64Image(region.ImageBase64)
		if decodeErr != nil {
			t.Fatalf("decode region %d: %v", index, decodeErr)
		}
		if decoded.Bounds().Size() != image.Pt(tileSize, tileSize) {
			t.Fatalf("region %d size = %v, want %dx%d", index, decoded.Bounds().Size(), tileSize, tileSize)
		}
		destination := image.Rect(
			region.SourceBounds.X,
			region.SourceBounds.Y,
			region.SourceBounds.X+region.SourceBounds.Width,
			region.SourceBounds.Y+region.SourceBounds.Height,
		)
		draw.Draw(reassembled, destination, decoded, decoded.Bounds().Min, draw.Src)
	}
	for y := range src.Bounds().Dy() {
		for x := range src.Bounds().Dx() {
			if got, want := reassembled.NRGBAAt(x, y), src.NRGBAAt(x, y); got != want {
				t.Fatalf("reassembled pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
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

func TestProcessorSplitImageAnimationSupportsSupersampledRenderFrames(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	fillRect(src, image.Rect(5, 5, 17, 35), color.NRGBA{R: 220, G: 70, B: 45, A: 255})
	fillRect(src, image.Rect(45, 5, 77, 35), color.NRGBA{R: 220, G: 70, B: 45, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64: encodeImageForTest(t, src),
		Mode:        ImageSplitModeAnimation,
		Columns:     2,
		Rows:        1,
		FrameWidth:  32,
		FrameHeight: 32,
		RenderScale: 4,
		Margin:      AnimationFrameMargin(32, 32),
		Anchor:      AnimationAnchorCenter,
	})
	if err != nil {
		t.Fatalf("split supersampled animation: %v", err)
	}
	if result.FrameWidth != 128 || result.FrameHeight != 128 || result.OutputWidth != 256 || result.OutputHeight != 128 {
		t.Fatalf("supersampled output = frame %dx%d sheet %dx%d, want frame 128x128 sheet 256x128", result.FrameWidth, result.FrameHeight, result.OutputWidth, result.OutputHeight)
	}
	for index, region := range result.Regions {
		decoded, err := DecodeBase64Image(region.ImageBase64)
		if err != nil {
			t.Fatalf("decode supersampled region %d: %v", index, err)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(128, 128) {
			t.Errorf("supersampled region %d size = %v, want 128x128", index, got)
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

func TestProcessorSplitImageAnimationNormalizesPrototypeScaleAndCenter(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 40))
	// Two views of the same static object are intentionally rendered at
	// different scales and at different positions inside their fixed cells.
	fillRect(src, image.Rect(14, 15, 24, 25), color.NRGBA{R: 220, G: 80, B: 40, A: 255})
	fillRect(src, image.Rect(50, 10, 70, 30), color.NRGBA{R: 220, G: 80, B: 40, A: 255})

	result, err := NewProcessor().SplitImage(context.Background(), &SplitImageRequest{
		ImageBase64:           encodeImageForTest(t, src),
		Mode:                  ImageSplitModeAnimation,
		Columns:               2,
		Rows:                  1,
		ForceProportionalGrid: true,
		FrameWidth:            64,
		FrameHeight:           64,
		Anchor:                AnimationAnchorCenter,
		NormalizeContentScale: true,
	})
	if err != nil {
		t.Fatalf("split normalized prototype: %v", err)
	}
	if len(result.Regions) != 2 {
		t.Fatalf("got %d regions, want 2", len(result.Regions))
	}

	var bounds [2]image.Rectangle
	for index, region := range result.Regions {
		decoded, err := DecodeBase64Image(region.ImageBase64)
		if err != nil {
			t.Fatalf("decode region %d: %v", index, err)
		}
		var ok bool
		bounds[index], ok = alphaBounds(decoded, defaultImageSplitAlphaThreshold)
		if !ok {
			t.Fatalf("region %d has no visible content", index)
		}
		if got := decoded.Bounds().Size(); got != image.Pt(64, 64) {
			t.Fatalf("region %d size = %v, want 64x64", index, got)
		}
	}
	if bounds[0].Dx() != bounds[1].Dx() || bounds[0].Dy() != bounds[1].Dy() {
		t.Fatalf("normalized content sizes differ: %v and %v", bounds[0].Size(), bounds[1].Size())
	}
	// Registration is driven by the configured anchor metadata rather than by
	// recentering the final alpha bounding box. This keeps irregular silhouettes
	// and asymmetric details intact.
}
