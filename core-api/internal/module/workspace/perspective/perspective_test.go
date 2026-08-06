package perspective_test

import (
	"reflect"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/perspective"
)

func TestPerspectiveHasExactlyThreeSupportedValues(t *testing.T) {
	supported := domain.Values()
	want := []domain.Perspective{"Top-Down", "Side-On", "Isometric"}
	if !reflect.DeepEqual(supported, want) {
		t.Fatalf("unexpected perspectives: %v", supported)
	}

	for _, perspective := range supported {
		if !perspective.Valid() {
			t.Errorf("expected %q to be valid", perspective)
		}
	}
	for _, perspective := range []domain.Perspective{"", "TopDown", "SideOn", "top_down", "side_on"} {
		if perspective.Valid() {
			t.Errorf("expected legacy perspective %q to be invalid", perspective)
		}
	}
}
