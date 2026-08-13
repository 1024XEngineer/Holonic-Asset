package video

import (
	"image"
	"strings"
	"testing"
)

func TestValidateExtractedFrameCountRejectsExcessFrames(t *testing.T) {
	err := validateExtractedFrameCount(maxExtractedFrames + 1)
	if err == nil {
		t.Fatal("expected excessive frame count to be rejected")
	}
	if !strings.Contains(err.Error(), "limit is 100") {
		t.Fatalf("expected frame limit error, got %v", err)
	}
}

func TestValidateExtractedFrameConfigsAcceptsSelectedFrameBudget(t *testing.T) {
	configs := make([]image.Config, 32)
	for index := range configs {
		configs[index] = image.Config{Width: 1024, Height: 1024}
	}

	if err := validateExtractedFrameConfigs(configs); err != nil {
		t.Fatalf("expected maximum selected frame set to fit memory budget: %v", err)
	}
}

func TestValidateExtractedFrameConfigsRejectsDecodedMemorySpike(t *testing.T) {
	configs := make([]image.Config, 33)
	for index := range configs {
		configs[index] = image.Config{Width: 1024, Height: 1024}
	}

	err := validateExtractedFrameConfigs(configs)
	if err == nil {
		t.Fatal("expected decoded frame memory spike to be rejected")
	}
	if !strings.Contains(err.Error(), "exceed 128 MiB memory budget") {
		t.Fatalf("expected decoded memory budget error, got %v", err)
	}
}

func TestValidateExtractedFrameConfigsRejectsOversizedFrame(t *testing.T) {
	err := validateExtractedFrameConfigs([]image.Config{{
		Width:  maxFrameDimension + 1,
		Height: 1,
	}})
	if err == nil {
		t.Fatal("expected oversized frame to be rejected")
	}
	if !strings.Contains(err.Error(), "exceed limit 4096x4096") {
		t.Fatalf("expected frame dimension limit error, got %v", err)
	}
}
