package handler

import "testing"

func TestProjectTagHandlerErrorAllowsNil(t *testing.T) {
	if err := projectTagHandlerError(nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
