package videoclient

import (
	"net/http"
	"testing"
	"time"
)

func TestNewDefaultQNAHTTPClientUsesExtendedTLSHandshakeTimeout(t *testing.T) {
	client := newDefaultQNAHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSHandshakeTimeout != defaultQNATLSHandshakeTimeout {
		t.Fatalf("TLS handshake timeout = %s, want %s", transport.TLSHandshakeTimeout, defaultQNATLSHandshakeTimeout)
	}
	if transport.TLSHandshakeTimeout != 45*time.Second {
		t.Fatalf("TLS handshake timeout = %s, want 45s", transport.TLSHandshakeTimeout)
	}
}
