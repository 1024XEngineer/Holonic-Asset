package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestEditObjectPrototypeDefinesReferenceRolesAndEditScopes(t *testing.T) {
	prompt := prompts.EditObjectPrototype(
		"change only the chest trim to silver",
		"Top-Down",
		prompts.SolidMatteBackground("#00FF00"),
	)

	for _, expected := range []string{
		"Reference image 1, the first supplied image, is always the original object prototype",
		"Reference image 2 and every later supplied image are project reference images",
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
