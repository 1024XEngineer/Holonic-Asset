package image

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestColorMathAndParsing(t *testing.T) {
	t.Parallel()

	// 1. ParseMatteColor named colors
	namedColors := map[string]MatteColor{
		"black":        {0, 0, 0},
		"white":        {255, 255, 255},
		"green":        {0, 255, 0},
		"chroma-green": {0, 255, 0},
		"magenta":      {255, 0, 255},
		"cyan":         {0, 255, 255},
		"blue":         {0, 0, 255},
	}
	for name, expected := range namedColors {
		parsed, err := ParseMatteColor(name)
		if err != nil || parsed != expected {
			t.Fatalf("ParseMatteColor(%q) = %v, %v; want %v", name, parsed, err, expected)
		}
	}

	// Hex parsing
	hexColor, err := ParseMatteColor("#123456")
	if err != nil || hexColor != (MatteColor{0x12, 0x34, 0x56}) {
		t.Fatalf("ParseMatteColor(#123456) = %v, %v", hexColor, err)
	}

	// Invalid hex parsing
	invalidColors := []string{"", "#123", "#GGFFFF", "random_non_color"}
	for _, inv := range invalidColors {
		if _, err := ParseMatteColor(inv); err == nil {
			t.Fatalf("expected error for invalid color: %q", inv)
		}
	}

	// 2. ParseMatteColorOrAuto
	autoAliases := []string{"auto", "sample", "auto-sample", "auto_sample"}
	for _, alias := range autoAliases {
		_, isAuto, err := ParseMatteColorOrAuto(alias)
		if err != nil || !isAuto {
			t.Fatalf("ParseMatteColorOrAuto(%q) expected isAuto=true, got %v, %v", alias, isAuto, err)
		}
	}
	c, isAuto, err := ParseMatteColorOrAuto("#00ff00")
	if err != nil || isAuto || c != (MatteColor{0, 255, 0}) {
		t.Fatalf("ParseMatteColorOrAuto(#00ff00) = %v, %v, %v", c, isAuto, err)
	}
	if _, _, err := ParseMatteColorOrAuto("invalid-color"); err == nil {
		t.Fatal("expected error on invalid color for ParseMatteColorOrAuto")
	}

	// 3. ColorToHex
	if hex := ColorToHex(MatteColor{10, 20, 30}); hex != "#0a141e" {
		t.Fatalf("ColorToHex = %q, want #0a141e", hex)
	}

	// 4. ColorDistance and EuclideanColorDistance
	d := ColorDistance(MatteColor{0, 0, 0}, MatteColor{3, 4, 0})
	if d != 25.0 {
		t.Fatalf("ColorDistance = %f, want 25.0", d)
	}
	ed := EuclideanColorDistance(MatteColor{0, 0, 0}, MatteColor{3, 4, 0})
	if ed != 5.0 {
		t.Fatalf("EuclideanColorDistance = %f, want 5.0", ed)
	}

	// 5. EstimateMatteColor empty bounds
	emptyImg := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if est := EstimateMatteColor(emptyImg); est != (MatteColor{}) {
		t.Fatalf("expected empty matte for empty image, got %v", est)
	}

	// EstimateMatteColor valid image
	validImg := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillRect(validImg, validImg.Bounds(), color.NRGBA{R: 50, G: 100, B: 150, A: 255})
	if est := EstimateMatteColor(validImg); est != (MatteColor{50, 100, 150}) {
		t.Fatalf("EstimateMatteColor = %v, want (50, 100, 150)", est)
	}

	// 6. median, ratio, clamp, sqrt helper edge cases
	if m := median(nil); m != 0 {
		t.Fatalf("expected median(nil) = 0, got %d", m)
	}
	if r := ratio(5, 0); r != 0 {
		t.Fatalf("expected ratio with 0 total = 0, got %f", r)
	}
	if c := clamp(10, 0, 5); c != 5 {
		t.Fatalf("expected clamp(10, 0, 5) = 5, got %f", c)
	}
	if c := clamp(-10, 0, 5); c != 0 {
		t.Fatalf("expected clamp(-10, 0, 5) = 0, got %f", c)
	}
	if s := sqrt(16); s != 4.0 {
		t.Fatalf("expected sqrt(16) = 4, got %f", s)
	}
}

func TestChromaSettingsForMaterialAndTypes(t *testing.T) {
	t.Parallel()

	materials := []Material{
		MaterialSoft3D,
		MaterialFlatIcon,
		MaterialSticker,
		MaterialGlow,
		MaterialStandard,
		"",
	}
	for _, mat := range materials {
		s := ChromaSettingsForMaterial(mat)
		if s.Threshold <= 0 || s.Softness <= 0 {
			t.Fatalf("invalid settings for material %q: %#v", mat, s)
		}
	}
}

func TestProcessorAPIErrors(t *testing.T) {
	t.Parallel()

	p := NewProcessor()
	ctx := context.Background()
	cancCtx, cancel := context.WithCancel(ctx)
	cancel()

	// 1. FlipHorizontal
	if flipper, ok := p.(HorizontalFlipper); ok {
		if _, err := flipper.FlipHorizontal(cancCtx, &FlipHorizontalRequest{}); err == nil {
			t.Fatal("expected error on canceled context")
		}
		if _, err := flipper.FlipHorizontal(ctx, nil); err == nil {
			t.Fatal("expected error on nil request")
		}
		if _, err := flipper.FlipHorizontal(ctx, &FlipHorizontalRequest{ImageBase64: "invalid"}); err == nil {
			t.Fatal("expected error on invalid base64")
		}
	}

	// 2. NormalizeReference
	if _, err := p.NormalizeReference(cancCtx, &NormalizeReferenceRequest{}); err == nil {
		t.Fatal("expected error on canceled context")
	}
	if _, err := p.NormalizeReference(ctx, nil); err == nil {
		t.Fatal("expected error on nil request")
	}
	if _, err := p.NormalizeReference(ctx, &NormalizeReferenceRequest{ImageBase64: "invalid"}); err == nil {
		t.Fatal("expected error on invalid base64")
	}

	// 3. Resize
	if _, err := p.Resize(cancCtx, &ResizeRequest{}); err == nil {
		t.Fatal("expected error on canceled context")
	}
	if _, err := p.Resize(ctx, nil); err == nil {
		t.Fatal("expected error on nil request")
	}
	if _, err := p.Resize(ctx, &ResizeRequest{ImageBase64: "invalid"}); err == nil {
		t.Fatal("expected error on invalid base64")
	}

	// 4. RemoveBackground
	if _, err := p.RemoveBackground(cancCtx, &RemoveBackgroundRequest{}); err == nil {
		t.Fatal("expected error on canceled context")
	}
	if _, err := p.RemoveBackground(ctx, nil); err == nil {
		t.Fatal("expected error on nil request")
	}
	if _, err := p.RemoveBackground(ctx, &RemoveBackgroundRequest{ImageBase64: "invalid"}); err == nil {
		t.Fatal("expected error on invalid base64")
	}

	// 5. decodeBase64Payload branches
	if _, err := decodeBase64Payload(""); err == nil {
		t.Fatal("expected error for empty base64")
	}
	if _, err := decodeBase64Payload("data:image/png;base64"); err == nil {
		t.Fatal("expected error for data URL without comma")
	}
	if _, err := decodeBase64Payload("data:image/png,abc"); err == nil {
		t.Fatal("expected error for data URL without ;base64")
	}

	// 6. pngColorType and paletteHasAlpha
	if _, _, err := pngColorType([]byte("not png")); err == nil {
		t.Fatal("expected error for non-png in pngColorType")
	}
	// Synthetic PNG header with 26 bytes
	fakePNG := make([]byte, 30)
	copy(fakePNG[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	copy(fakePNG[12:16], []byte("XXXX")) // not IHDR
	if _, _, err := pngColorType(fakePNG); err == nil {
		t.Fatal("expected error for missing IHDR chunk")
	}

	copy(fakePNG[12:16], []byte("IHDR"))
	for _, ct := range []struct {
		code uint8
		name string
	}{
		{0, "grayscale"},
		{2, "rgb"},
		{3, "indexed"},
		{4, "grayscale-alpha"},
		{6, "rgba"},
		{99, "unknown(99)"},
	} {
		fakePNG[25] = ct.code
		name, _, err := pngColorType(fakePNG)
		if err != nil || name != ct.name {
			t.Fatalf("pngColorType code %d: got %q, %v; want %q", ct.code, name, err, ct.name)
		}
	}

	// 7. hasAlphaFromImage Paletted with transparent and opaque
	paletteWithAlpha := color.Palette{color.NRGBA{R: 0, A: 100}, color.NRGBA{R: 255, A: 255}}
	palImgAlpha := image.NewPaletted(image.Rect(0, 0, 4, 4), paletteWithAlpha)
	if !hasAlphaFromImage(palImgAlpha) {
		t.Fatal("expected true for paletted image with alpha")
	}

	paletteOpaque := color.Palette{color.NRGBA{R: 0, A: 255}, color.NRGBA{R: 255, A: 255}}
	palImgOpaque := image.NewPaletted(image.Rect(0, 0, 4, 4), paletteOpaque)
	if hasAlphaFromImage(palImgOpaque) {
		t.Fatal("expected false for paletted image without alpha")
	}

	// 8. ResizeImage validation error branches
	testImg := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	if _, _, err := ResizeImage(nil, ResizeOptions{}); err == nil {
		t.Fatal("expected error on nil input")
	}
	if _, _, err := ResizeImage(testImg, ResizeOptions{Width: 0, Height: 16}); err == nil {
		t.Fatal("expected error on non-positive width")
	}
	if _, _, err := ResizeImage(testImg, ResizeOptions{Width: 16, Height: 16, PaletteSize: -1}); err == nil {
		t.Fatal("expected error on negative palette size")
	}
	if _, _, err := ResizeImage(testImg, ResizeOptions{Width: 16, Height: 16, Mode: "invalid_mode"}); err == nil {
		t.Fatal("expected error on invalid mode")
	}
	if _, _, err := ResizeImage(testImg, ResizeOptions{Width: 16, Height: 16, Margin: 10}); err == nil {
		t.Fatal("expected error on margin >= half dimension")
	}
	if _, _, err := ResizeImage(testImg, ResizeOptions{Width: 16, Height: 16, Margin: 2, CoverCanvas: true}); err == nil {
		t.Fatal("expected error on CoverCanvas with non-zero margin")
	}
	if _, _, err := ResizeImage(image.NewNRGBA(image.Rect(0, 0, 0, 0)), ResizeOptions{Width: 16, Height: 16, Margin: 0}); err == nil {
		t.Fatal("expected error on empty input image")
	}
}
