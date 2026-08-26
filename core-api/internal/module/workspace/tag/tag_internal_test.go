package tag

import (
	"errors"
	"testing"
)

func TestTagValidateCreateRejectsNilTag(t *testing.T) {
	var tag *Tag
	if err := tag.validateCreate(); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected invalid tag error, got %v", err)
	}
}
