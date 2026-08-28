package generator

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

const (
	defaultPrototypeReferenceTimeout        = 2 * time.Minute
	defaultPrototypeReferenceDialTimeout    = 10 * time.Second
	defaultPrototypeReferenceHeaderTimeout  = 30 * time.Second
	defaultPrototypeReferenceTLSHandshaking = 15 * time.Second
	defaultPrototypeReferenceMaxRetries     = 2
	defaultPrototypeReferenceRetryDelay     = 500 * time.Millisecond
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
	transport.ForceAttemptHTTP2 = false
	transport.TLSClientConfig = &tls.Config{
		NextProtos: []string{"http/1.1"},
	}
	transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	return &http.Client{
		Transport: transport,
		Timeout:   defaultPrototypeReferenceTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (e *executor) downloadPrototypeReference(ctx context.Context, referenceURL *url.URL) ([]byte, error) {
	client := e.referenceHTTPClient
	if client == nil {
		client = newPrototypeReferenceHTTPClient()
	}
	attempts := defaultPrototypeReferenceMaxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		startedAt := time.Now()
		requestContext, cancel := context.WithTimeout(ctx, defaultPrototypeReferenceTimeout)
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, referenceURL.String(), nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("create reference download request: %w", err)
		}

		response, err := client.Do(request)
		if err != nil {
			cancel()
			willRetry := attempt < attempts && ctx.Err() == nil && prototypeReferenceRequestTransient(err)
			e.logPrototypeReferenceFailure(referenceURL, attempt, attempts, 0, startedAt, 0, willRetry, err)
			if !willRetry {
				return nil, fmt.Errorf("download reference: %w", err)
			}
			if err := sleepBeforePrototypeReferenceRetry(ctx, attempt); err != nil {
				return nil, fmt.Errorf("download reference: %w", err)
			}
			continue
		}

		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<10))
			_ = response.Body.Close()
			cancel()
			willRetry := attempt < attempts && ctx.Err() == nil &&
				prototypeReferenceStatusTransient(response.StatusCode)
			e.logPrototypeReferenceFailure(
				referenceURL, attempt, attempts, response.StatusCode, startedAt, len(body), willRetry, readErr,
			)
			if willRetry {
				if err := sleepBeforePrototypeReferenceRetry(ctx, attempt); err != nil {
					return nil, fmt.Errorf("download reference: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf(
				"download reference: HTTP %d: %s",
				response.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}

		if response.ContentLength > maxPrototypeReferenceBytes {
			_ = response.Body.Close()
			cancel()
			return nil, fmt.Errorf("reference exceeds %d bytes", maxPrototypeReferenceBytes)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxPrototypeReferenceBytes+1))
		closeErr := response.Body.Close()
		cancel()
		if readErr == nil {
			readErr = closeErr
		}
		if readErr != nil {
			willRetry := attempt < attempts && ctx.Err() == nil && prototypeReferenceRequestTransient(readErr)
			e.logPrototypeReferenceFailure(
				referenceURL, attempt, attempts, response.StatusCode, startedAt, len(body), willRetry, readErr,
			)
			if !willRetry {
				return nil, fmt.Errorf("read reference: %w", readErr)
			}
			if err := sleepBeforePrototypeReferenceRetry(ctx, attempt); err != nil {
				return nil, fmt.Errorf("read reference: %w", err)
			}
			continue
		}
		if len(body) > maxPrototypeReferenceBytes {
			return nil, fmt.Errorf("reference exceeds %d bytes", maxPrototypeReferenceBytes)
		}
		if len(body) == 0 {
			willRetry := attempt < attempts && ctx.Err() == nil
			emptyErr := errors.New("empty response")
			e.logPrototypeReferenceFailure(
				referenceURL, attempt, attempts, response.StatusCode, startedAt, 0, willRetry, emptyErr,
			)
			if !willRetry {
				return nil, fmt.Errorf("download reference: %w", emptyErr)
			}
			if err := sleepBeforePrototypeReferenceRetry(ctx, attempt); err != nil {
				return nil, fmt.Errorf("download reference: %w", err)
			}
			continue
		}
		e.logPrototypeReferenceSuccess(referenceURL, attempt, attempts, response.StatusCode, startedAt, len(body))
		return body, nil
	}
	return nil, fmt.Errorf("download reference: exhausted retries")
}

func prototypeReferenceStatusTransient(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooEarly ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func prototypeReferenceRequestTransient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return prototypeReferenceRequestTransient(urlErr.Err)
	}
	var operationErr *net.OpError
	if errors.As(err, &operationErr) {
		return true
	}
	var networkErr net.Error
	return errors.As(err, &networkErr) && networkErr.Timeout()
}

func sleepBeforePrototypeReferenceRetry(ctx context.Context, attempt int) error {
	delay := defaultPrototypeReferenceRetryDelay
	for range min(attempt-1, 4) {
		delay *= 2
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (e *executor) prototypeReferenceLogFields(
	referenceURL *url.URL,
	attempt int,
	attempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
) []logger.Field {
	return []logger.Field{
		logger.String("provider", "reference_storage"),
		logger.String("stage", "download_prototype_reference"),
		logger.String("method", http.MethodGet),
		logger.String("source_host", referenceURL.Hostname()),
		logger.String("endpoint", referenceURL.EscapedPath()),
		logger.Int("attempt", attempt),
		logger.Int("max_attempts", attempts),
		logger.Int("status_code", statusCode),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		logger.Int("response_bytes", responseBytes),
	}
}

func (e *executor) logPrototypeReferenceFailure(
	referenceURL *url.URL,
	attempt int,
	attempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
	willRetry bool,
	err error,
) {
	if e.logger == nil {
		return
	}
	fields := e.prototypeReferenceLogFields(
		referenceURL, attempt, attempts, statusCode, startedAt, responseBytes,
	)
	fields = append(fields, logger.Any("will_retry", willRetry))
	if err != nil {
		fields = append(fields, logger.Error(err))
	}
	e.logger.Warn("prototype reference download failed", fields...)
}

func (e *executor) logPrototypeReferenceSuccess(
	referenceURL *url.URL,
	attempt int,
	attempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
) {
	if e.logger == nil {
		return
	}
	e.logger.Debug(
		"prototype reference download completed",
		e.prototypeReferenceLogFields(referenceURL, attempt, attempts, statusCode, startedAt, responseBytes)...,
	)
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
	sort.SliceStable(addresses, func(i, j int) bool {
		return addresses[i].Is4() && !addresses[j].Is4()
	})
	return addresses, nil
}
