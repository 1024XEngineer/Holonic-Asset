package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestCharacterPrototypeIncludesFullBodyStyleAndDirectionLayout(t *testing.T) {
	prompt := prompts.CharacterPrototype(
		"a silver-armored dragon-born interstellar soldier",
		"Side-On",
		prompts.SolidMatteBackground("#00FF00"),
	)

	for _, expected := range []string{
		"complete full-body character",
		"same game as the project references",
		"uniform, solid #00FF00 colour",
		"exactly 2 full-body direction views",
		"1 row x 2 column sheet",
		"left-facing and right-facing views in that order",
		"normal reading order",
		"Complete the first row before starting the second row",
		"equal gutters and equal margins",
		"one regular output sheet",
		"silver-armored dragon-born interstellar soldier",
		"Side-On",
		"<direction_count>\n2\n</direction_count>",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected character prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestCharacterPrototypeDerivesDirectionLayoutFromPerspective(t *testing.T) {
	tests := []struct {
		name        string
		perspective string
		direction   string
		expected    []string
	}{
		{
			name:        "side on",
			perspective: "Side-On",
			direction:   "2",
			expected:    []string{"Side-on perspective", "exactly 2 full-body direction views", "1 row x 2 column sheet", "left-facing and right-facing views in that order"},
		},
		{
			name:        "top down",
			perspective: "Top-Down",
			direction:   "4",
			expected:    []string{"Top-down perspective", "exactly 4 full-body direction views", "2 row x 2 column sheet", "front, right, back, and left views in that order"},
		},
		{
			name:        "isometric",
			perspective: "Isometric",
			direction:   "8",
			expected:    []string{"Isometric perspective", "exactly 8 full-body direction views", "2 row x 4 column sheet", "front, front-right, right, back-right, back, back-left, left, and front-left views in that order"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.CharacterPrototype(
				"a readable player character",
				test.perspective,
				prompts.TransparentBackground(),
			)
			for _, expected := range test.expected {
				if !strings.Contains(prompt, expected) {
					t.Fatalf("expected %s prompt to contain %q: %s", test.perspective, expected, prompt)
				}
			}
			if !strings.Contains(prompt, "<direction_count>\n"+test.direction+"\n</direction_count>") {
				t.Fatalf("expected %s direction count in prompt: %s", test.direction, prompt)
			}
		})
	}
}
