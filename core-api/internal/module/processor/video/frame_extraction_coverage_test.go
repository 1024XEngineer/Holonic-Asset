package video

import (
	"image"
	"image/color"
	"os"
	"testing"
)

func TestResolveAutoDetectedChromaKeyEdgeCases(t *testing.T) {
	t.Parallel()

	baseKey := ChromaKey{
		AutoDetect:          true,
		HighSaturationMin:   50,
		HighValueMin:        50,
		BrightSaturationMin: 30,
		BrightValueMin:      150,
	}

	// 1. AutoDetect is false
	nonAuto := baseKey
	nonAuto.AutoDetect = false
	if res := resolveAutoDetectedChromaKey(nil, nonAuto); res.AutoDetect {
		t.Fatal("expected AutoDetect=false to return key unchanged")
	}

	// 2. Empty frames slice
	if res := resolveAutoDetectedChromaKey(nil, baseKey); !res.AutoDetect {
		t.Fatal("expected empty frames to return key unchanged")
	}

	// 3. Nil frame or empty bounds
	if res := resolveAutoDetectedChromaKey([]image.Image{nil}, baseKey); !res.AutoDetect {
		t.Fatal("expected nil frame to return key unchanged")
	}
	emptyImg := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if res := resolveAutoDetectedChromaKey([]image.Image{emptyImg}, baseKey); !res.AutoDetect {
		t.Fatal("expected empty frame to return key unchanged")
	}

	// 4. All transparent corner pixels (count == 0)
	transImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	if res := resolveAutoDetectedChromaKey([]image.Image{transImg}, baseKey); !res.AutoDetect {
		t.Fatal("expected all transparent corners to return key unchanged")
	}

	// 5. Red corners (hue ~0 <= 18 -> HueMin = 0)
	redImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			redImg.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	redKey := resolveAutoDetectedChromaKey([]image.Image{redImg}, baseKey)
	if redKey.AutoDetect || redKey.HueMin != 0 {
		t.Fatalf("red corners HueMin = %d, want 0", redKey.HueMin)
	}

	// 6. Green corners (hue ~60 > 18 -> HueMin > 0)
	greenImg := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			greenImg.SetNRGBA(x, y, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	greenKey := resolveAutoDetectedChromaKey([]image.Image{greenImg}, baseKey)
	if greenKey.AutoDetect || greenKey.HueMin == 0 {
		t.Fatalf("green corners HueMin = %d, want > 0", greenKey.HueMin)
	}
}

func TestAverageColorChannelAndClampOpenCVHue(t *testing.T) {
	t.Parallel()

	// 1. averageColorChannel
	if avg := averageColorChannel(100, 0); avg != 0 {
		t.Fatalf("expected 0 for count=0, got %d", avg)
	}
	if avg := averageColorChannel(1000, 2); avg != 255 {
		t.Fatalf("expected 255 for overflow, got %d", avg)
	}
	if avg := averageColorChannel(200, 2); avg != 100 {
		t.Fatalf("expected 100, got %d", avg)
	}

	// 2. clampOpenCVHue
	if h := clampOpenCVHue(-5); h != 0 {
		t.Fatalf("expected 0 for negative hue, got %d", h)
	}
	if h := clampOpenCVHue(200); h != 179 {
		t.Fatalf("expected 179 for overflow hue, got %d", h)
	}
	if h := clampOpenCVHue(50); h != 50 {
		t.Fatalf("expected 50, got %d", h)
	}
}

func TestValidateSelectedFrameIndicesAndExpression(t *testing.T) {
	t.Parallel()

	// 1. Empty indices
	if err := validateSelectedFrameIndices(nil, 10); err == nil {
		t.Fatal("expected error on empty indices")
	}

	// 2. Out of range (negative or >= candidateCount)
	if err := validateSelectedFrameIndices([]int{-1}, 10); err == nil {
		t.Fatal("expected error on negative index")
	}
	if err := validateSelectedFrameIndices([]int{10}, 10); err == nil {
		t.Fatal("expected error on index >= candidateCount")
	}

	// 3. Not strictly increasing (equal or decreasing)
	if err := validateSelectedFrameIndices([]int{1, 1}, 10); err == nil {
		t.Fatal("expected error on duplicate index")
	}
	if err := validateSelectedFrameIndices([]int{3, 2}, 10); err == nil {
		t.Fatal("expected error on decreasing index")
	}

	// 4. Valid indices
	if err := validateSelectedFrameIndices([]int{0, 2, 5}, 10); err != nil {
		t.Fatalf("expected valid indices to pass: %v", err)
	}

	// 5. ffmpegSelectExpression
	expr := ffmpegSelectExpression([]int{0, 2, 5})
	if expr != "eq(n\\,0)+eq(n\\,2)+eq(n\\,5)" {
		t.Fatalf("unexpected select expression: %q", expr)
	}
}

func TestValidateExtractedFrameConfigsAndResolveFFmpeg(t *testing.T) {
	t.Parallel()

	// 1. validateExtractedFrameCount
	if err := validateExtractedFrameCount(maxExtractedFrames + 1); err == nil {
		t.Fatal("expected error when count exceeds limit")
	}
	if err := validateExtractedFrameCount(10); err != nil {
		t.Fatalf("expected valid count to pass: %v", err)
	}

	// 2. validateExtractedFrameConfigs
	if err := validateExtractedFrameConfigs([]image.Config{{Width: 0, Height: 10}}); err == nil {
		t.Fatal("expected error for non-positive width")
	}
	if err := validateExtractedFrameConfigs([]image.Config{{Width: maxFrameDimension + 1, Height: 10}}); err == nil {
		t.Fatal("expected error for exceeding max dimension")
	}
	hugeConfigs := []image.Config{
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
		{Width: 3000, Height: 3000},
	}
	if err := validateExtractedFrameConfigs(hugeConfigs); err == nil {
		t.Fatal("expected error for memory budget overflow")
	}

	// 3. resolveFFmpeg with directory or non-existent path
	tempDir := t.TempDir()
	if _, err := resolveFFmpeg(tempDir); err == nil {
		t.Fatal("expected error when ffmpeg path is a directory")
	}
	if _, err := resolveFFmpeg(tempDir + "/non_existent_ffmpeg_binary"); err == nil {
		t.Fatal("expected error when ffmpeg path does not exist")
	}

	// 4. decodeFrames and decodeFrameConfig error branches
	if _, err := decodeFrames([]string{tempDir + "/missing.png"}, ""); err == nil {
		t.Fatal("expected error when decoding non-existent file")
	}
	if _, err := decodeFrameConfig(tempDir + "/missing.png"); err == nil {
		t.Fatal("expected error when decoding config for non-existent file")
	}

	// Corrupt file
	corruptPath := tempDir + "/corrupt.png"
	_ = os.WriteFile(corruptPath, []byte("not image data"), 0o600)
	if _, err := decodeFrames([]string{corruptPath}, ""); err == nil {
		t.Fatal("expected error when decoding corrupt image file")
	}
	if _, err := decodeFrameConfig(corruptPath); err == nil {
		t.Fatal("expected error when decoding config for corrupt image file")
	}
}
