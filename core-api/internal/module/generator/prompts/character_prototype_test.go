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
		prompts.PrototypeReferenceState{HasProjectReference: true, HasCreatingReference: true},
	)

	for _, expected := range []string{
		"complete full-body character",
		"Reference image 1 is the project prototype image and is the Style Reference",
		"Reference image 2 is the creating reference image",
		"creating reference image is always a strong reference",
		"uniform, solid #00FF00 colour",
		"exactly 2 direction views",
		"1 row x 2 column sheet",
		"SIDE-ON SCALE LOCK",
		"same pixel height",
		"never treat the second cell as an independently framed composition",
		"redraw the mismatched view rather than compensating with a different zoom",
		"normal reading order",
		"Complete the first row before starting the second row",
		"reading-order indexes",
		"zero-based array index is the direction identity",
		"index 0 = left, index 1 = right",
		"never render both cells facing the same way",
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
			expected:    []string{"Side-on perspective", "exactly 2 direction views", "1 row x 2 column sheet"},
		},
		{
			name:        "top down",
			perspective: "Top-Down",
			direction:   "4",
			expected:    []string{"Top-down perspective", "exactly 4 direction views", "2 row x 2 column sheet"},
		},
		{
			name:        "isometric",
			perspective: "Isometric",
			direction:   "8",
			expected:    []string{"Isometric perspective", "exactly 8 direction views", "2 row x 4 column sheet"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prompt := prompts.CharacterPrototype(
				"a readable player character",
				test.perspective,
				prompts.TransparentBackground(),
				prompts.PrototypeReferenceState{},
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
