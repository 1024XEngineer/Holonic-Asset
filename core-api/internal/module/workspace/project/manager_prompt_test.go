package project

import (
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

func TestPerspectivePromptCoversAllPerspectives(t *testing.T) {
	tests := []struct {
		perspective Perspective
		fragment    string
	}{
		{perspective: PerspectiveTopDown, fragment: "tile-based playfield"},
		{perspective: PerspectiveSideOn, fragment: "layered parallax backgrounds"},
		{perspective: PerspectiveIsometric, fragment: "walkable tiles"},
		{perspective: Perspective("unknown"), fragment: "best suited to the brief"},
	}
	for _, tc := range tests {
		if prompt := perspectivePrompt(tc.perspective); !strings.Contains(prompt, tc.fragment) {
			t.Errorf("perspective %q prompt %q does not contain %q", tc.perspective, prompt, tc.fragment)
		}
	}
}

func TestPlatformPromptCoversAllPlatforms(t *testing.T) {
	tests := []struct {
		platform PlatformType
		fragment string
	}{
		{platform: PlatformTypePC, fragment: "keyboard/controller"},
		{platform: PlatformTypeMobile, fragment: "touch-friendly"},
		{platform: PlatformTypeWeb, fragment: "responsive gameplay layout"},
		{platform: PlatformType("unknown"), fragment: "unspecified target platform"},
	}
	for _, tc := range tests {
		if prompt := platformPrompt(tc.platform); !strings.Contains(prompt, tc.fragment) {
			t.Errorf("platform %q prompt %q does not contain %q", tc.platform, prompt, tc.fragment)
		}
	}
}

func TestReferenceDataURLDefaultsMediaType(t *testing.T) {
	if got := referenceDataURL(imageclient.GeneratedImage{Base64: "payload"}); got != "data:image/png;base64,payload" {
		t.Fatalf("default reference data URL = %q", got)
	}
	if got := referenceDataURL(imageclient.GeneratedImage{MediaType: " image/jpeg ", Base64: "payload"}); got != "data:image/jpeg;base64,payload" {
		t.Fatalf("explicit reference data URL = %q", got)
	}
}
