package video

import (
	"image"
	"strings"
	"testing"
)

func TestValidateExtractedAnimationFrameCountRejectsExcessFrames(t *testing.T) {
	err := validateExtractedAnimationFrameCount(maxAnimationExtractedFrames + 1)
	if err == nil {
		t.Fatal("expected excessive animation frame count to be rejected")
	}
	if !strings.Contains(err.Error(), "limit is 100") {
		t.Fatalf("expected frame limit error, got %v", err)
	}
}

func TestValidateExtractedAnimationFrameConfigsAcceptsSelectedFrameBudget(t *testing.T) {
	configs := make([]image.Config, 32)
	for index := range configs {
		configs[index] = image.Config{Width: 1024, Height: 1024}
	}

	if err := validateExtractedAnimationFrameConfigs(configs); err != nil {
		t.Fatalf("expected maximum selected frame set to fit memory budget: %v", err)
	}
}

func TestValidateExtractedAnimationFrameConfigsRejectsDecodedMemorySpike(t *testing.T) {
	configs := make([]image.Config, 33)
	for index := range configs {
		configs[index] = image.Config{Width: 1024, Height: 1024}
	}

	err := validateExtractedAnimationFrameConfigs(configs)
	if err == nil {
		t.Fatal("expected decoded animation frame memory spike to be rejected")
	}
	if !strings.Contains(err.Error(), "exceed 128 MiB memory budget") {
		t.Fatalf("expected decoded memory budget error, got %v", err)
	}
}

func TestValidateExtractedAnimationFrameConfigsRejectsOversizedFrame(t *testing.T) {
	err := validateExtractedAnimationFrameConfigs([]image.Config{{
		Width:  maxAnimationFrameDimension + 1,
		Height: 1,
	}})
	if err == nil {
		t.Fatal("expected oversized animation frame to be rejected")
	}
	if !strings.Contains(err.Error(), "exceed limit 4096x4096") {
		t.Fatalf("expected frame dimension limit error, got %v", err)
	}
}
