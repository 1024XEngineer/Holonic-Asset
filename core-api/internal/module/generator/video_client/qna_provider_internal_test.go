package videoclient

import (
	"net/http"
	"testing"
	"time"
)

func TestQNAProviderUsesIndependentDefaultHTTPClient(t *testing.T) {
	first := NewQNAProvider(QNAConfig{})
	second := NewQNAProvider(QNAConfig{})
	firstAdapter, ok := first.downloader.(*qnaFalQueueAdapter)
	if !ok {
		t.Fatalf("first downloader type = %T, want *qnaFalQueueAdapter", first.downloader)
	}
	secondAdapter, ok := second.downloader.(*qnaFalQueueAdapter)
	if !ok {
		t.Fatalf("second downloader type = %T, want *qnaFalQueueAdapter", second.downloader)
	}

	if firstAdapter.httpClient == secondAdapter.httpClient {
		t.Fatal("QNA providers must not share the same default HTTP client")
	}
	firstTransport, ok := firstAdapter.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("first QNA transport type = %T, want *http.Transport", firstAdapter.httpClient.Transport)
	}
	secondTransport, ok := secondAdapter.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("second QNA transport type = %T, want *http.Transport", secondAdapter.httpClient.Transport)
	}
	if firstTransport == secondTransport || firstTransport == http.DefaultTransport || secondTransport == http.DefaultTransport {
		t.Fatal("QNA providers must use independent HTTP connection pools")
	}
	if firstAdapter.pollTimeout != 45*time.Second {
		t.Fatalf("default poll timeout = %s, want 45s", firstAdapter.pollTimeout)
	}
}
