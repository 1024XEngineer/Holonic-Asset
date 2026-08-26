package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestEditObjectPrototypeDefinesReferenceRolesAndEditScopes(t *testing.T) {
	prompt := prompts.EditObjectPrototype(
		"a wooden chest",
		"change only the chest trim to silver",
		"Top-Down",
		4,
		assetdomain.Size{Width: 48, Height: 48},
		prompts.SolidMatteBackground("#00FF00"),
	)

	for _, expected := range []string{
		"backend supplied exactly 4 current prototype direction image(s)",
		"Treat every supplied reference image as part of the original object prototype",
		"No user or project reference image is supplied",
		"zero-based array index is the direction identity",
		"index 0 = front, index 1 = right, index 2 = back, index 3 = left",
		"a wooden chest",
		"Minor edit",
		"Major edit",
		"Mixed edit",
		"uniform, solid #00FF00 colour",
		"change only the chest trim to silver",
		"Top-Down",
		"<asset_dimensions>\n{\"width\":48,\"height\":48}\n</asset_dimensions>",
		"Use the full 48 x 48 logical prototype canvas",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected object edit prompt to contain %q: %s", expected, prompt)
		}
	}
}

func TestEditObjectPrototypeExplicitlyLowersDetailAt32Pixels(t *testing.T) {
	prompt := prompts.EditObjectPrototype(
		"a wooden chest",
		"change the trim",
		"Top-Down",
		4,
		assetdomain.Size{Width: 32, Height: 32},
		prompts.TransparentBackground(),
	)
	for _, expected := range []string{
		"short edge of 32 pixels or less",
		"Explicitly lower the design detail level before rendering",
		"Omit secondary texture, seams, folds, small accessories, tiny construction parts, logos, and decorative marks",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected 32px object edit prompt to contain %q: %s", expected, prompt)
		}
	}
}
