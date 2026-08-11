package videoclient

import "testing"

func TestProviderErrorIncludesHTTPStatus(t *testing.T) {
	err := (&ProviderError{
		Provider:   "qna",
		Kind:       ErrorKindUnavailable,
		StatusCode: 525,
		Transient:  true,
		Message:    "empty response",
	}).Error()
	if err != "qna provider: HTTP 525: empty response" {
		t.Fatalf("error = %q", err)
	}
}
