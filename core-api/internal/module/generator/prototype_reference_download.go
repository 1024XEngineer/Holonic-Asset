package generator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const (
	defaultPrototypeReferenceTimeout        = 30 * time.Second
	defaultPrototypeReferenceDialTimeout    = 10 * time.Second
	defaultPrototypeReferenceHeaderTimeout  = 15 * time.Second
	defaultPrototypeReferenceTLSHandshaking = 10 * time.Second
)

var blockedPrototypeReferencePrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fec0::/10"),
}

type prototypeReferenceDialer struct {
	lookupNetIP func(context.Context, string, string) ([]netip.Addr, error)
	dialContext func(context.Context, string, string) (net.Conn, error)
}

func newPrototypeReferenceHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: defaultPrototypeReferenceDialTimeout}
	secureDialer := prototypeReferenceDialer{
		lookupNetIP: net.DefaultResolver.LookupNetIP,
		dialContext: dialer.DialContext,
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = secureDialer.DialContext
	transport.ResponseHeaderTimeout = defaultPrototypeReferenceHeaderTimeout
	transport.TLSHandshakeTimeout = defaultPrototypeReferenceTLSHandshaking
	return &http.Client{
		Transport: transport,
		Timeout:   defaultPrototypeReferenceTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func validatePrototypeReferenceURL(value *url.URL) error {
	if value == nil {
		return fmt.Errorf("prototype reference URL is required")
	}
	scheme := strings.ToLower(value.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("prototype reference URL scheme %q is unsupported", value.Scheme)
	}
	host := value.Hostname()
	if host == "" {
		return fmt.Errorf("prototype reference URL host is required")
	}
	if strings.Contains(host, "%") {
		return fmt.Errorf("prototype reference URL host zones are unsupported")
	}
	if address, err := netip.ParseAddr(host); err == nil {
		return validatePrototypeReferenceAddress(address)
	}
	return nil
}

func validatePrototypeReferenceAddress(address netip.Addr) error {
	address = address.Unmap()
	if !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() ||
		address.IsMulticast() ||
		address.IsUnspecified() {
		return fmt.Errorf("prototype reference address %q is not public", address)
	}
	for _, prefix := range blockedPrototypeReferencePrefixes {
		if prefix.Contains(address) {
			return fmt.Errorf("prototype reference address %q is not public", address)
		}
	}
	return nil
}

func (d prototypeReferenceDialer) DialContext(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse prototype reference address: %w", err)
	}

	addresses, err := d.resolve(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, candidate := range addresses {
		if err := validatePrototypeReferenceAddress(candidate); err != nil {
			return nil, fmt.Errorf("resolve prototype reference host %q: %w", host, err)
		}
	}

	var dialErrors []error
	for _, candidate := range addresses {
		connection, dialErr := d.dialContext(
			ctx,
			network,
			net.JoinHostPort(candidate.String(), port),
		)
		if dialErr == nil {
			return connection, nil
		}
		dialErrors = append(dialErrors, dialErr)
	}
	return nil, fmt.Errorf("dial prototype reference host %q: %w", host, errors.Join(dialErrors...))
}

func (d prototypeReferenceDialer) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{address.Unmap()}, nil
	}
	resolved, err := d.lookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve prototype reference host %q: %w", host, err)
	}
	addresses := make([]netip.Addr, 0, len(resolved))
	for _, value := range resolved {
		if value.IsValid() {
			addresses = append(addresses, value.Unmap())
		}
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve prototype reference host %q: no IP addresses", host)
	}
	return addresses, nil
}
