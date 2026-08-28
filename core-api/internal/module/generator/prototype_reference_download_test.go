package generator

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

type prototypeDownloadRoundTripFunc func(*http.Request) (*http.Response, error)

func (f prototypeDownloadRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type prototypeDownloadReadCloser struct {
	err error
}

func (r prototypeDownloadReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (prototypeDownloadReadCloser) Close() error {
	return nil
}

type prototypeDownloadTimeoutError struct{}

func (prototypeDownloadTimeoutError) Error() string   { return "timed out" }
func (prototypeDownloadTimeoutError) Timeout() bool   { return true }
func (prototypeDownloadTimeoutError) Temporary() bool { return true }

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

func TestValidatePrototypeReferenceURLRejectsMalformedTargets(t *testing.T) {
	tests := []struct {
		name  string
		value *url.URL
		want  string
	}{
		{name: "missing URL", want: "URL is required"},
		{name: "unsupported scheme", value: &url.URL{Scheme: "file", Host: "example.com"}, want: "scheme"},
		{name: "missing host", value: &url.URL{Scheme: "https"}, want: "host is required"},
		{name: "IPv6 zone", value: &url.URL{Scheme: "https", Host: "[fe80::1%25en0]"}, want: "zones are unsupported"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePrototypeReferenceURL(test.value)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want error containing %q", err, test.want)
			}
		})
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

func TestPrototypeReferenceDialerRejectsAddressWithoutPort(t *testing.T) {
	dialer := prototypeReferenceDialer{}

	_, err := dialer.DialContext(context.Background(), "tcp", "references.example")
	if err == nil || !strings.Contains(err.Error(), "parse prototype reference address") {
		t.Fatalf("dial error = %v, want address parse failure", err)
	}
}

func TestPrototypeReferenceDialerReportsLookupFailure(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	dialer := prototypeReferenceDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return nil, lookupErr
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "references.example:443")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("dial error = %v, want wrapped lookup failure", err)
	}
}

func TestPrototypeReferenceDialerRejectsEmptyResolution(t *testing.T) {
	dialer := prototypeReferenceDialer{
		lookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{{}}, nil
		},
	}

	_, err := dialer.DialContext(context.Background(), "tcp", "references.example:443")
	if err == nil || !strings.Contains(err.Error(), "no IP addresses") {
		t.Fatalf("dial error = %v, want empty resolution failure", err)
	}
}

func TestPrototypeReferenceDialerConnectsToPublicLiteralAddress(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = server.Close() }()

	var dialedAddress string
	dialer := prototypeReferenceDialer{
		dialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialedAddress = address
			return client, nil
		},
	}

	connection, err := dialer.DialContext(context.Background(), "tcp", "8.8.8.8:443")
	if err != nil {
		t.Fatalf("dial public literal address: %v", err)
	}
	defer func() { _ = connection.Close() }()
	if dialedAddress != "8.8.8.8:443" {
		t.Fatalf("dialed address = %q, want public literal address", dialedAddress)
	}
}

func TestPrototypeReferenceHTTPClientUsesBoundedSecureTransport(t *testing.T) {
	client := newPrototypeReferenceHTTPClient()
	if client.Timeout != 2*time.Minute {
		t.Fatalf("client timeout = %s, want 2m", client.Timeout)
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
	if transport.ForceAttemptHTTP2 {
		t.Fatal("prototype reference transport must have ForceAttemptHTTP2 = false")
	}
	if transport.TLSNextProto == nil {
		t.Fatal("prototype reference transport must initialize TLSNextProto to disable HTTP/2")
	}

	redirectURL, err := url.Parse("https://cdn.example.com/redirected-reference.png")
	if err != nil {
		t.Fatalf("parse redirect fixture: %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: redirectURL}, []*http.Request{{}}); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect error = %v, want redirects disabled", err)
	}
}

func TestDownloadPrototypeReferenceRetriesResponseReadFailure(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       prototypeDownloadReadCloser{err: context.DeadlineExceeded},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("reference-image")),
			}, nil
		}),
	}}

	body, err := executor.downloadPrototypeReference(context.Background(), parsed)
	if err != nil {
		t.Fatalf("download reference: %v", err)
	}
	if string(body) != "reference-image" {
		t.Fatalf("downloaded body = %q, want reference-image", body)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
}

func TestDownloadPrototypeReferenceRetriesTransientHTTPStatus(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("temporarily unavailable")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("reference-image")),
			}, nil
		}),
	}}

	body, err := executor.downloadPrototypeReference(context.Background(), parsed)
	if err != nil {
		t.Fatalf("download reference: %v", err)
	}
	if string(body) != "reference-image" {
		t.Fatalf("downloaded body = %q, want reference-image", body)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
}

func TestDownloadPrototypeReferenceDoesNotRetryDeterministicHTTPStatus(t *testing.T) {
	parsed, err := url.Parse("https://references.example/missing.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}),
	}}

	_, err = executor.downloadPrototypeReference(context.Background(), parsed)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("download error = %v, want HTTP 404", err)
	}
	if attempts != 1 {
		t.Fatalf("download attempts = %d, want 1", attempts)
	}
}

func TestDownloadPrototypeReferenceRetriesTransientRequestFailure(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	logs := &recordingLogger{}
	executor := &executor{
		logger: logs,
		referenceHTTPClient: &http.Client{Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, io.EOF
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("reference-image")),
			}, nil
		})},
	}

	body, err := executor.downloadPrototypeReference(context.Background(), parsed)
	if err != nil {
		t.Fatalf("download reference: %v", err)
	}
	if string(body) != "reference-image" || attempts != 2 {
		t.Fatalf("downloaded body = %q after %d attempts", body, attempts)
	}
	if len(logs.entries) != 2 || logs.entries[0].Level != "warn" || logs.entries[1].Level != "debug" {
		t.Fatalf("unexpected download logs: %+v", logs.entries)
	}
	fields := make(map[string]any, len(logs.entries[0].Fields))
	for _, field := range logs.entries[0].Fields {
		fields[field.Key] = field.Val
	}
	if fields["source_host"] != "references.example" || fields["endpoint"] != "/reference.png" {
		t.Fatalf("unexpected reference log fields: %+v", fields)
	}
	if fields["will_retry"] != true || fields["errorx"] == nil {
		t.Fatalf("missing retry error fields: %+v", fields)
	}
}

func TestDownloadPrototypeReferenceRejectsOversizedContentLength(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: maxPrototypeReferenceBytes + 1,
				Body:          io.NopCloser(strings.NewReader("oversized")),
			}, nil
		}),
	}}

	_, err = executor.downloadPrototypeReference(context.Background(), parsed)
	if err == nil || !strings.Contains(err.Error(), "reference exceeds") {
		t.Fatalf("download error = %v, want size limit rejection", err)
	}
}

func TestDownloadPrototypeReferenceDoesNotRetryPermanentReadFailure(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       prototypeDownloadReadCloser{err: errors.New("corrupt response")},
			}, nil
		}),
	}}

	_, err = executor.downloadPrototypeReference(context.Background(), parsed)
	if err == nil || !strings.Contains(err.Error(), "read reference") {
		t.Fatalf("download error = %v, want read failure", err)
	}
	if attempts != 1 {
		t.Fatalf("download attempts = %d, want 1", attempts)
	}
}

func TestDownloadPrototypeReferenceRetriesEmptyResponses(t *testing.T) {
	parsed, err := url.Parse("https://references.example/reference.png")
	if err != nil {
		t.Fatalf("parse reference URL: %v", err)
	}
	attempts := 0
	executor := &executor{referenceHTTPClient: &http.Client{
		Transport: prototypeDownloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}}

	_, err = executor.downloadPrototypeReference(context.Background(), parsed)
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("download error = %v, want empty response failure", err)
	}
	if attempts != defaultPrototypeReferenceMaxRetries+1 {
		t.Fatalf("download attempts = %d, want %d", attempts, defaultPrototypeReferenceMaxRetries+1)
	}
}

func TestPrototypeReferenceRequestTransientClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil"},
		{name: "canceled", err: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "EOF", err: io.EOF, want: true},
		{name: "wrapped URL error", err: &url.Error{Op: "Get", URL: "https://example.com", Err: io.ErrUnexpectedEOF}, want: true},
		{name: "network operation", err: &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset")}, want: true},
		{name: "network timeout", err: prototypeDownloadTimeoutError{}, want: true},
		{name: "permanent", err: errors.New("invalid response")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := prototypeReferenceRequestTransient(test.err); got != test.want {
				t.Fatalf("transient classification = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSleepBeforePrototypeReferenceRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sleepBeforePrototypeReferenceRetry(ctx, 2)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retry wait error = %v, want context cancellation", err)
	}
}

func TestPrototypeReferenceDialerPrioritizesIPv4Addresses(t *testing.T) {
	dialer := prototypeReferenceDialer{
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

func TestLivePrototypeReferenceDownloadAgainstRealQiniuEndpoint(t *testing.T) {
	client := newPrototypeReferenceHTTPClient()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, "https://xe-6-2.s3.cn-east-1.qiniucs.com", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("download against real Qiniu S3 endpoint failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	t.Logf("Successfully connected to Qiniu S3! Proto=%s, StatusCode=%d", resp.Proto, resp.StatusCode)
	if resp.Proto != "HTTP/1.1" {
		t.Fatalf("proto = %q, want HTTP/1.1", resp.Proto)
	}
}
