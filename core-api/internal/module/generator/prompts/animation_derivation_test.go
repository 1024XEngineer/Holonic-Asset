package prompts

import (
	"strings"
	"testing"
)

func TestBuildAnimationDerivationVideo(t *testing.T) {
	t.Run("default options builds valid universal prompt", func(t *testing.T) {
		prompt := BuildAnimationDerivationVideo(AnimationDerivationOptions{})
		if !strings.Contains(prompt, "LOCKED STATIC CAMERA") {
			t.Errorf("expected prompt to contain camera lock, got %q", prompt)
		}
		if !strings.Contains(prompt, "CRITICAL ORIENTATION LOCK") {
			t.Errorf("expected prompt to contain orientation lock, got %q", prompt)
		}
		if !strings.Contains(prompt, "MULTI-REFERENCE CONTRACT") {
			t.Errorf("expected prompt to contain multi-reference contract, got %q", prompt)
		}
		if !strings.Contains(prompt, "FRAME 1 MUST reproduce Image 1") {
			t.Errorf("expected prompt to lock the opening frame to Image 1, got %q", prompt)
		}
		if !strings.Contains(prompt, "Use Image 2 ONLY") {
			t.Errorf("expected prompt to limit Image 2 to action and effects, got %q", prompt)
		}
		if !strings.Contains(prompt, DefaultAnimationStyle) {
			t.Errorf("expected default style %q, got %q", DefaultAnimationStyle, prompt)
		}
	})

	t.Run("specified action and target orientation are included", func(t *testing.T) {
		prompt := BuildAnimationDerivationVideo(AnimationDerivationOptions{
			Description:       "Cute pixel cleaning robot with backpack water tank",
			Style:             "16-bit retro pixel art",
			Action:            "spray cleaning liquid straight forward",
			TargetOrientation: "Up / North (Back View)",
			SourceOrientation: "Left (Side-On View)",
			FrameCount:        8,
			FrameWidth:        256,
			FrameHeight:       192,
		})

		if !strings.Contains(prompt, "Up / North (Back View)") {
			t.Errorf("expected target orientation in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "Left (Side-On View)") {
			t.Errorf("expected source orientation in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "spray cleaning liquid straight forward") {
			t.Errorf("expected action name in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "Cute pixel cleaning robot") {
			t.Errorf("expected description in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "16-bit retro pixel art") {
			t.Errorf("expected custom style in prompt, got %q", prompt)
		}
		for _, required := range []string{
			"NEVER copy Image 2's character orientation",
			"exactly 8 ordered animation frames",
			"256x192 output frame",
			"canvas boundary rule below always wins",
			"shorten and contain the effect rather than touching or crossing an edge",
			"compress ONLY its longitudinal reach along the target facing axis",
			"Preserve its lateral width, particle density, opacity, texture, turbulence, timing, terminal burst/splash, and overall visual intensity",
		} {
			if !strings.Contains(prompt, required) {
				t.Errorf("expected derivation constraint %q, got %q", required, prompt)
			}
		}
	})

	t.Run("character length limit is enforced", func(t *testing.T) {
		longDesc := strings.Repeat("very long description text ", 50)
		longStyle := strings.Repeat("hyper detailed pixel art style ", 30)
		longAction := strings.Repeat("spray massive torrents of water ", 30)

		prompt := BuildAnimationDerivationVideo(AnimationDerivationOptions{
			Description:       longDesc,
			Style:             longStyle,
			Action:            longAction,
			TargetOrientation: "Down / South",
		})

		if len(prompt) > MaxAnimationVideoCharacters {
			t.Errorf("prompt length %d exceeded MaxAnimationVideoCharacters %d", len(prompt), MaxAnimationVideoCharacters)
		}
	})
}

func TestBuildAnimationDerivationImage(t *testing.T) {
	prompt := BuildAnimationDerivationImage(AnimationImageDerivationOptions{
		Description:       "blue maintenance robot",
		Style:             "crisp 16-bit pixel art",
		Action:            "spray cleaning liquid",
		TargetOrientation: "Right / East",
		SourceOrientation: "Left / West",
		FrameCount:        8,
		Columns:           4,
		Rows:              2,
		FrameWidth:        256,
		FrameHeight:       192,
	})
	for _, required := range []string{
		"TOP panel",
		"LOWER panel",
		"Right / East",
		"Left / West",
		"output cell N must represent source cell N",
		"exactly 8 chronological frames",
		"4x2 grid",
		"256x192 frame",
		"Do not output the top prototype panel",
		"chroma-green (#00FF00)",
	} {
		if !strings.Contains(prompt, required) {
			t.Errorf("expected image derivation constraint %q, got %q", required, prompt)
		}
	}
}
