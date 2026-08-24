package project

import "testing"

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
