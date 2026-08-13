package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestEditObjectPrototypeDefinesReferenceRolesAndEditScopes(t *testing.T) {
	prompt := prompts.EditObjectPrototype(
		"a wooden chest",
		"change only the chest trim to silver",
		"Top-Down",
		4,
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
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected object edit prompt to contain %q: %s", expected, prompt)
		}
	}
}
