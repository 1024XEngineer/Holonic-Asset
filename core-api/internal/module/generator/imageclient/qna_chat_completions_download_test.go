package imageclient

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

func TestValidateGeneratedImageURLRejectsUnsafeTargets(t *testing.T) {
	for _, value := range []string{
		"http://127.0.0.1/image.png",
		"http://10.0.0.1/image.png",
		"http://100.64.0.1/image.png",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.10/image.png",
		"http://192.0.2.1/image.png",
		"http://[::1]/image.png",
		"http://[fe80::1]/image.png",
		"http://[64:ff9b::a9fe:a9fe]/latest/meta-data",
		"http://[2002:7f00:1::]/image.png",
	} {
		t.Run(value, func(t *testing.T) {
			parsed, err := url.Parse(value)
			if err != nil {
				t.Fatalf("parse fixture URL: %v", err)
			}
			if err := validateGeneratedImageURL(parsed); err == nil || !strings.Contains(err.Error(), "not public") {
				t.Fatalf("validation error = %v, want non-public rejection", err)
			}
		})
	}

	for _, value := range []string{"file:///tmp/image.png", "https://[fe80::1%25lo0]/image.png", "https:///image.png"} {
		t.Run(value, func(t *testing.T) {
			parsed, err := url.Parse(value)
			if err != nil {
				t.Fatalf("parse fixture URL: %v", err)
			}
			if err := validateGeneratedImageURL(parsed); err == nil {
				t.Fatal("unsafe URL was accepted")
			}
		})
	}

	for _, value := range []string{"https://8.8.8.8/image.png", "https://images.example/image.png"} {
		parsed, err := url.Parse(value)
		if err != nil {
			t.Fatalf("parse public URL: %v", err)
		}
		if err := validateGeneratedImageURL(parsed); err != nil {
			t.Fatalf("public URL rejected: %v", err)
		}
	}

	if err := validateGeneratedImageURL(nil); err == nil {
		t.Fatal("nil URL was accepted")
	}
}

func TestGeneratedImageDialerRejectsMixedResolution(t *testing.T) {
	dialCalled := false
	dialer := generatedImageDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, nil
		},
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			dialCalled = true
			return nil, errors.New("unexpected dial")
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "images.example:443")
	if err == nil || !strings.Contains(err.Error(), "not public") {
		t.Fatalf("dial error = %v, want non-public rejection", err)
	}
	if dialCalled {
		t.Fatal("dial attempted before all resolved addresses were validated")
	}
}

func TestGeneratedImageDialerPinsValidatedAddresses(t *testing.T) {
	var dialed []string
	dialer := generatedImageDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1")}, nil
		},
		dialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialed = append(dialed, address)
			if len(dialed) == 1 {
				return nil, errors.New("first address unavailable")
			}
			connection, peer := net.Pipe()
			_ = peer.Close()
			return connection, nil
		},
	}

	connection, err := dialer.DialContext(context.Background(), "tcp", "images.example:443")
	if err != nil || connection == nil {
		t.Fatalf("dial result = (%v, %v), want successful test dial", connection, err)
	}
	_ = connection.Close()
	if len(dialed) != 2 || dialed[0] != "8.8.8.8:443" || dialed[1] != "1.1.1.1:443" {
		t.Fatalf("dialed addresses = %v", dialed)
	}
}

func TestGeneratedImageDialerReportsResolutionAndDialFailures(t *testing.T) {
	t.Run("invalid address", func(t *testing.T) {
		dialer := generatedImageDialer{}
		if _, err := dialer.DialContext(context.Background(), "tcp", "missing-port"); err == nil {
			t.Fatal("invalid address was accepted")
		}
	})

	t.Run("lookup failure", func(t *testing.T) {
		dialer := generatedImageDialer{
			lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
		}
		if _, err := dialer.DialContext(context.Background(), "tcp", "images.example:443"); err == nil ||
			!strings.Contains(err.Error(), "lookup failed") {
			t.Fatalf("lookup error = %v", err)
		}
	})

	t.Run("no addresses", func(t *testing.T) {
		dialer := generatedImageDialer{
			lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
				return []netip.Addr{{}}, nil
			},
		}
		if _, err := dialer.DialContext(context.Background(), "tcp", "images.example:443"); err == nil ||
			!strings.Contains(err.Error(), "no IP addresses") {
			t.Fatalf("empty resolution error = %v", err)
		}
	})

	t.Run("all dials fail", func(t *testing.T) {
		wantErr := errors.New("dial failed")
		dialer := generatedImageDialer{
			lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
			dialContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, wantErr
			},
		}
		if _, err := dialer.DialContext(context.Background(), "tcp", "images.example:443"); err == nil || !errors.Is(err, wantErr) {
			t.Fatalf("dial error = %v", err)
		}
	})
}

func TestGeneratedImageHTTPClientUsesSecureTransport(t *testing.T) {
	client := newGeneratedImageHTTPClient()
	if client.Timeout != generatedImageDownloadTimeout {
		t.Fatalf("client timeout = %s, want %s", client.Timeout, generatedImageDownloadTimeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("generated image transport must not use environment proxies")
	}
	if transport.DialContext == nil || transport.ResponseHeaderTimeout == 0 ||
		transport.TLSHandshakeTimeout == 0 || transport.MaxResponseHeaderBytes == 0 {
		t.Fatalf("generated image transport is missing bounds: %+v", transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatal("generated image transport must have ForceAttemptHTTP2 = false")
	}
	if transport.TLSClientConfig == nil || len(transport.TLSClientConfig.NextProtos) != 1 || transport.TLSClientConfig.NextProtos[0] != "http/1.1" {
		t.Fatalf("generated image transport TLSClientConfig.NextProtos = %v, want [\"http/1.1\"]", transport.TLSClientConfig)
	}
	if transport.TLSNextProto == nil {
		t.Fatal("generated image transport must initialize TLSNextProto")
	}

	privateURL, err := url.Parse("http://169.254.169.254/latest/meta-data")
	if err != nil {
		t.Fatalf("parse redirect fixture: %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: privateURL}, []*http.Request{{}}); err == nil {
		t.Fatal("redirect to link-local metadata address was accepted")
	}
	if err := client.CheckRedirect(&http.Request{URL: privateURL}, make([]*http.Request, 10)); err == nil ||
		!strings.Contains(err.Error(), "redirect limit") {
		t.Fatalf("redirect limit error = %v", err)
	}
}

func TestGeneratedImageDialerPrioritizesIPv4Addresses(t *testing.T) {
	dialer := generatedImageDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("240e:1234::1"),
				netip.MustParseAddr("1.2.3.4"),
				netip.MustParseAddr("240e:5678::2"),
				netip.MustParseAddr("5.6.7.8"),
			}, nil
		},
	}

	addrs, err := dialer.resolve(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(addrs) != 4 {
		t.Fatalf("len(addrs) = %d, want 4", len(addrs))
	}
	if !addrs[0].Is4() || !addrs[1].Is4() {
		t.Fatalf("expected IPv4 addresses first, got: %v", addrs)
	}
	if addrs[0].String() != "1.2.3.4" || addrs[1].String() != "5.6.7.8" {
		t.Fatalf("IPv4 addresses not preserved in order: %v", addrs)
	}
	if !addrs[2].Is6() || !addrs[3].Is6() {
		t.Fatalf("expected IPv6 addresses after IPv4, got: %v", addrs)
	}
}
