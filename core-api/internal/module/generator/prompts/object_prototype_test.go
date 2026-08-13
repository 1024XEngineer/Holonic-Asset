package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestObjectPrototypeIncludesInputsStyleAndProcessingConstraints(t *testing.T) {
	background := prompts.SolidMatteBackground("#00FF00")
	prompt := prompts.ObjectPrototype("a wooden chest with two locks", "Top-Down", background)

	for _, expected := range []string{
		"pipeline processing requirements have the highest priority",
		"uniform, solid #00FF00 colour",
		"Do not output transparency or a checkerboard",
		"strictly follow their art style",
		"exactly 4 direction views",
		"2 row x 2 column sheet",
		"normal reading order",
		"reading-order indexes",
		"zero-based array index is the direction identity",
		"index 0 = front, index 1 = right, index 2 = back, index 3 = left",
		"a wooden chest with two locks",
		"Top-Down",
		"<direction_count>\n4\n</direction_count>",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected object prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestTransparentBackgroundRequiresRealAlpha(t *testing.T) {
	constraint := prompts.TransparentBackground()
	for _, expected := range []string{"real alpha channel", "Do not draw a checkerboard pattern"} {
		if !strings.Contains(constraint, expected) {
			t.Fatalf("expected transparent background constraint to contain %q: %s", expected, constraint)
		}
	}
}
