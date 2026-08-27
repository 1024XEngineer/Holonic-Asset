package project

import (
	"bytes"
	cryptorand "crypto/rand"
	"strconv"
	"testing"
)

func TestPromptHelpersDirect(t *testing.T) {
	if got := perspectivePrompt(Perspective("Custom")); got != "choose the 2D pixel-art gameplay perspective best suited to the brief and keep it consistent with an authentic playable screenshot" {
		t.Fatalf("unexpected custom perspective prompt: %q", got)
	}

	if got := platformPrompt(PlatformType("Console")); got != "an unspecified target platform; infer suitable controls, orientation, and information density from the user brief" {
		t.Fatalf("unexpected custom platform prompt: %q", got)
	}

	if got := promptValue("   ", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback for whitespace value, got %q", got)
	}
	if got := promptValue("custom-val", "fallback"); got != "custom-val" {
		t.Fatalf("expected value for non-empty value, got %q", got)
	}
}

func TestRandomReferenceSeedFallsBackWhenReaderFails(t *testing.T) {
	original := cryptorand.Reader
	cryptorand.Reader = bytes.NewReader(nil)
	defer func() { cryptorand.Reader = original }()

	seed := randomReferenceSeed()
	value, err := strconv.ParseInt(seed, 10, 64)
	if err != nil {
		t.Fatalf("fallback seed %q is not a base-10 integer: %v", seed, err)
	}
	if value < 0 || value >= 1_000_000_000 {
		t.Fatalf("fallback seed %d is outside [0, 1_000_000_000)", value)
	}
}
