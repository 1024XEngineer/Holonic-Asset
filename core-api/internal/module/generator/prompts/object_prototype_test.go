package prompts_test

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/prompts"
)

func TestObjectPrototypeIncludesUserInputsAndPriority(t *testing.T) {
	prompt := prompts.ObjectPrototype("a wooden chest with two locks", "top_down")

	for _, expected := range []string{
		"The user requirements have the highest priority.",
		"a wooden chest with two locks",
		"top_down",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q: %s", expected, prompt)
		}
	}
}
