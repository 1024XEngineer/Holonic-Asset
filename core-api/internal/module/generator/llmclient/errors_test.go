package llmclient_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

func TestProviderErrorFormattingAndUnwrap(t *testing.T) {
	cause := errors.New("connection reset")
	tests := []struct {
		name string
		err  *llmclient.ProviderError
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{
			name: "provider message",
			err: &llmclient.ProviderError{
				Provider: "qna",
				Kind:     llmclient.ErrorKindTransport,
				Message:  "request failed",
				Cause:    cause,
			},
			want: "qna provider: request failed",
		},
		{
			name: "default provider and kind",
			err:  &llmclient.ProviderError{Kind: llmclient.ErrorKindUnavailable},
			want: "LLM provider: unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}

	providerErr := tests[1].err
	if !errors.Is(providerErr, cause) || !errors.Is(providerErr.Unwrap(), cause) {
		t.Fatalf("cause was not preserved: %v", providerErr)
	}
	var nilProviderErr *llmclient.ProviderError
	if nilProviderErr.Unwrap() != nil {
		t.Fatal("nil ProviderError must not unwrap to a cause")
	}
}

func TestIsTransient(t *testing.T) {
	transient := &llmclient.ProviderError{Transient: true}
	permanent := &llmclient.ProviderError{Transient: false}

	if !llmclient.IsTransient(fmt.Errorf("complete request: %w", transient)) {
		t.Fatal("wrapped transient provider error was not detected")
	}
	if llmclient.IsTransient(permanent) {
		t.Fatal("permanent provider error was reported as transient")
	}
	if llmclient.IsTransient(errors.New("ordinary error")) {
		t.Fatal("ordinary error was reported as transient")
	}
}
