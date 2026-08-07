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

func TestCharacterDirectionCountFollowsPerspective(t *testing.T) {
	tests := []struct {
		perspective domain.Perspective
		want        uint
	}{
		{perspective: domain.SideOn, want: 2},
		{perspective: domain.TopDown, want: 4},
		{perspective: domain.Isometric, want: 8},
		{perspective: domain.Perspective("unsupported"), want: 0},
	}

	for _, test := range tests {
		if got := test.perspective.CharacterDirectionCount(); got != test.want {
			t.Fatalf("direction count for %q = %d, want %d", test.perspective, got, test.want)
		}
	}
}
