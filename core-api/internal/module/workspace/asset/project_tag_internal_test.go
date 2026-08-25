package asset

import (
	"errors"
	"testing"
)

func TestProjectTagValidateCreateRejectsNilTag(t *testing.T) {
	var tag *ProjectTag
	if err := tag.validateCreate(); !errors.Is(err, ErrInvalidProjectTag) {
		t.Fatalf("expected invalid project tag error, got %v", err)
	}
}
