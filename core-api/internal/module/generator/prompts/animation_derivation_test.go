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
		if !strings.Contains(prompt, "spray cleaning liquid straight forward") {
			t.Errorf("expected action name in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "Cute pixel cleaning robot") {
			t.Errorf("expected description in prompt, got %q", prompt)
		}
		if !strings.Contains(prompt, "16-bit retro pixel art") {
			t.Errorf("expected custom style in prompt, got %q", prompt)
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
