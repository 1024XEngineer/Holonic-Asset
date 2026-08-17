package imageclient

import (
	"errors"
	"fmt"
	"testing"
)

func TestProviderErrorRetryClassification(t *testing.T) {
	transient := &ProviderError{Kind: ErrorKindRateLimited, Transient: true}
	permanent := &ProviderError{Kind: ErrorKindAuthentication, Transient: false}
	ordinary := errors.New("unclassified")

	if !IsTransient(fmt.Errorf("generate: %w", transient)) || IsPermanent(transient) {
		t.Fatal("transient provider error was misclassified")
	}
	if !IsPermanent(fmt.Errorf("generate: %w", permanent)) || IsTransient(permanent) {
		t.Fatal("permanent provider error was misclassified")
	}
	if IsTransient(ordinary) || IsPermanent(ordinary) {
		t.Fatal("unclassified error must remain unclassified")
	}
}
