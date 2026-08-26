package generator

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	projectdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/project"
)

func createTestPNG(w, h int, drawFunc func(img *image.RGBA)) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if drawFunc != nil {
		drawFunc(img)
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestVerifyTileSetNoGuideLeak(t *testing.T) {
	t.Run("invalid base64 decode error", func(t *testing.T) {
		err := verifyTileSetNoGuideLeak("not-valid-base64")
		if err == nil {
			t.Fatal("expected decode error")
		}
	})

	t.Run("valid clean transparent image", func(t *testing.T) {
		b64 := createTestPNG(32, 32, nil)
		if err := verifyTileSetNoGuideLeak(b64); err != nil {
			t.Fatalf("unexpected error for transparent image: %v", err)
		}
	})

	t.Run("valid colored sprite image without guide leak", func(t *testing.T) {
		b64 := createTestPNG(32, 32, func(img *image.RGBA) {
			// Center red square
			for y := 10; y < 22; y++ {
				for x := 10; x < 22; x++ {
					img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
				}
			}
		})
		if err := verifyTileSetNoGuideLeak(b64); err != nil {
			t.Fatalf("unexpected error for clean sprite: %v", err)
		}
	})

	t.Run("near-black guide leak detected", func(t *testing.T) {
		b64 := createTestPNG(32, 32, func(img *image.RGBA) {
			// Fill entire cell with near-black (<=20) opaque pixels
			for y := range 32 {
				for x := range 32 {
					img.Set(x, y, color.RGBA{R: 5, G: 5, B: 5, A: 255})
				}
			}
		})
		err := verifyTileSetNoGuideLeak(b64)
		if err == nil {
			t.Fatal("expected near-black guide leak error")
		}
		if !strings.Contains(err.Error(), "near-black occupancy-guide component matches the Tile bounds") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestFormatTileSetProjectContext(t *testing.T) {
	proj := &projectdomain.Project{
		Name:           "Cyberpunk City",
		GameType:       "RPG",
		Description:    "Futuristic dystopian game",
		Style:          "Pixel Art 16-bit",
		TargetPlatform: "PC",
		Perspective:    "Top-Down",
	}
	formatted := formatTileSetProjectContext(proj)
	for _, expected := range []string{
		"Cyberpunk City",
		"RPG",
		"Futuristic dystopian game",
		"Pixel Art 16-bit",
		"PC",
		"Top-Down",
	} {
		if !strings.Contains(formatted, expected) {
			t.Fatalf("expected context to contain %q: %s", expected, formatted)
		}
	}
}

func TestTileSetItemBounds(t *testing.T) {
	shape := []TileSetCoordinate{
		{2, 3},
		{1, 5},
		{4, 2},
		{3, 4},
	}
	minX, minY, maxX, maxY := tileSetItemBounds(shape)
	if minX != 1 || minY != 2 || maxX != 4 || maxY != 5 {
		t.Fatalf("got bounds (%d,%d,%d,%d), want (1,2,4,5)", minX, minY, maxX, maxY)
	}
}
