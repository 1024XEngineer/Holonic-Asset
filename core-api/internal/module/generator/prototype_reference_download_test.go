package generator

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

func TestValidatePrototypeReferenceURLRejectsNonPublicLiteralTargets(t *testing.T) {
	for _, value := range []string{
		"http://127.0.0.1/reference.png",
		"http://10.0.0.1/reference.png",
		"http://100.64.0.1/reference.png",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.10/reference.png",
		"http://[::1]/reference.png",
		"http://[fe80::1]/reference.png",
		"http://[64:ff9b::a9fe:a9fe]/latest/meta-data",
		"http://[2002:7f00:1::]/reference.png",
	} {
		t.Run(value, func(t *testing.T) {
			parsed, err := url.Parse(value)
			if err != nil {
				t.Fatalf("parse fixture URL: %v", err)
			}
			if err := validatePrototypeReferenceURL(parsed); err == nil || !strings.Contains(err.Error(), "not public") {
				t.Fatalf("validation error = %v, want non-public rejection", err)
			}
		})
	}

	publicURL, err := url.Parse("https://8.8.8.8/reference.png")
	if err != nil {
		t.Fatalf("parse public fixture URL: %v", err)
	}
	if err := validatePrototypeReferenceURL(publicURL); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}
}

func TestPrototypeReferenceDialerRejectsMixedPublicAndPrivateResolution(t *testing.T) {
	dialCalled := false
	dialer := prototypeReferenceDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, nil
		},
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			dialCalled = true
			return nil, errors.New("unexpected dial")
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "references.example:443")
	if err == nil || !strings.Contains(err.Error(), "not public") {
		t.Fatalf("dial error = %v, want non-public rejection", err)
	}
	if dialCalled {
		t.Fatal("dial attempted before all resolved addresses were validated")
	}
}

func TestPrototypeReferenceDialerPinsValidatedIPAddress(t *testing.T) {
	var dialedAddress string
	dialer := prototypeReferenceDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		},
		dialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialedAddress = address
			return nil, errors.New("stop after capturing address")
		},
	}

	_, _ = dialer.DialContext(context.Background(), "tcp", "references.example:443")
	if dialedAddress != "8.8.8.8:443" {
		t.Fatalf("dialed address = %q, want resolved public IP", dialedAddress)
	}
}

func TestPrototypeReferenceHTTPClientUsesBoundedSecureTransport(t *testing.T) {
	client := newPrototypeReferenceHTTPClient()
	if client.Timeout != defaultPrototypeReferenceTimeout {
		t.Fatalf("client timeout = %s, want %s", client.Timeout, defaultPrototypeReferenceTimeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("prototype reference transport must not use environment proxies")
	}
	if transport.DialContext == nil || transport.ResponseHeaderTimeout == 0 || transport.TLSHandshakeTimeout == 0 {
		t.Fatalf("prototype reference transport is missing bounded timeouts: %+v", transport)
	}

	redirectURL, err := url.Parse("https://cdn.example.com/redirected-reference.png")
	if err != nil {
		t.Fatalf("parse redirect fixture: %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: redirectURL}, []*http.Request{{}}); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect error = %v, want redirects disabled", err)
	}
}
