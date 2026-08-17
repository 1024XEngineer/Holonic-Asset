package videoclient

import (
	"net/http"
	"testing"
	"time"
)

func TestQNAProviderUsesIndependentDefaultHTTPClient(t *testing.T) {
	first := NewQNAProvider(QNAConfig{})
	second := NewQNAProvider(QNAConfig{})

	if first.httpClient == second.httpClient {
		t.Fatal("QNA providers must not share the same default HTTP client")
	}
	firstTransport, ok := first.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("first QNA transport type = %T, want *http.Transport", first.httpClient.Transport)
	}
	secondTransport, ok := second.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("second QNA transport type = %T, want *http.Transport", second.httpClient.Transport)
	}
	if firstTransport == secondTransport || firstTransport == http.DefaultTransport || secondTransport == http.DefaultTransport {
		t.Fatal("QNA providers must use independent HTTP connection pools")
	}
	if first.pollTimeout != 45*time.Second {
		t.Fatalf("default poll timeout = %s, want 45s", first.pollTimeout)
	}
}
