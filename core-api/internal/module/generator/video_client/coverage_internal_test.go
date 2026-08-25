package videoclient

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

type coverageVideoRoundTripFunc func(*http.Request) (*http.Response, error)

func (f coverageVideoRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type coverageVideoLogger struct{}

func (*coverageVideoLogger) Debug(string, ...logger.Field) {}
func (*coverageVideoLogger) Info(string, ...logger.Field)  {}
func (*coverageVideoLogger) Warn(string, ...logger.Field)  {}
func (*coverageVideoLogger) Error(string, ...logger.Field) {}
func (*coverageVideoLogger) Sync() error                   { return nil }

type coverageVideoReadCloser struct {
	data     string
	readErr  error
	closeErr error
	onRead   func()
	onClose  func()
	read     bool
}

func (r *coverageVideoReadCloser) Read(buffer []byte) (int, error) {
	if r.onRead != nil {
		r.onRead()
		r.onRead = nil
	}
	if r.read {
		return 0, r.readErr
	}
	r.read = true
	if r.data != "" {
		return copy(buffer, r.data), r.readErr
	}
	return 0, r.readErr
}

func (r *coverageVideoReadCloser) Close() error {
	if r.onClose != nil {
		r.onClose()
		r.onClose = nil
	}
	return r.closeErr
}

type coverageTimeoutError struct{}

func (coverageTimeoutError) Error() string { return "timeout" }
func (coverageTimeoutError) Timeout() bool { return true }

type coverageVideoProvider struct {
	generateErr error
}

func (p *coverageVideoProvider) Generate(context.Context, *ProviderRequest) (*ProviderResult, error) {
	return nil, p.generateErr
}

func (*coverageVideoProvider) Download(context.Context, string) ([]byte, error) {
	return []byte("video"), nil
}

func coverageVideoResponse(status int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       body,
		Proto:      "HTTP/1.1",
	}
}

func coverageVideoAdapter(roundTrip coverageVideoRoundTripFunc) *qnaFalQueueAdapter {
	return newQNAFalQueueAdapter(qnaFalQueueAdapterConfig{
		BaseURL:      "https://video.example.test",
		APIKey:       "key",
		PollInterval: time.Nanosecond,
		PollTimeout:  time.Millisecond,
		MaxRetries:   -1,
		RetryDelay:   time.Nanosecond,
		HTTPClient:   &http.Client{Transport: roundTrip},
	})
}

func TestVideoCoverageErrorAndPureHelperBranches(t *testing.T) {
	var nilErr *ProviderError
	if nilErr.Error() != "" || nilErr.Unwrap() != nil {
		t.Fatal("nil provider error methods returned data")
	}
	cause := errors.New("cause")
	withStatus := &ProviderError{StatusCode: 418, Kind: ErrorKindInvalidRequest, Cause: cause}
	withoutStatus := &ProviderError{Kind: ErrorKindTransport}
	if !strings.Contains(withStatus.Error(), "HTTP 418") || !strings.Contains(withoutStatus.Error(), string(ErrorKindTransport)) {
		t.Fatalf("unexpected provider errors: %q / %q", withStatus, withoutStatus)
	}
	if !errors.Is(withStatus, cause) || IsTransient(withStatus) || !IsTransient(&ProviderError{Transient: true}) {
		t.Fatal("provider error wrapping or transient classification failed")
	}

	adapter := newQNAFalQueueAdapter(qnaFalQueueAdapterConfig{})
	if adapter.httpClient == nil {
		t.Fatal("default adapter HTTP client is nil")
	}
	if _, err := adapter.endpoint("://invalid"); err != nil {
		t.Fatalf("valid base with unusual path failed: %v", err)
	}
	invalidBase := &qnaFalQueueAdapter{baseURL: "://invalid"}
	if _, err := invalidBase.endpoint("/requests"); err == nil {
		t.Fatal("invalid API base was accepted")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	deadline, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	classifications := []struct {
		status int
		err    error
		kind   ErrorKind
	}{
		{err: context.Canceled, kind: ErrorKindCanceled},
		{err: context.DeadlineExceeded, kind: ErrorKindTimeout},
		{err: coverageTimeoutError{}, kind: ErrorKindTimeout},
		{status: 200, err: errors.New("read"), kind: ErrorKindInvalidResponse},
		{err: errors.New("transport"), kind: ErrorKindTransport},
	}
	for _, test := range classifications {
		kind, _ := classifyQNARequestFailure(test.status, test.err)
		if kind != test.kind {
			t.Fatalf("request failure = %s, want %s", kind, test.kind)
		}
	}
	contextErrors := []struct {
		ctx  context.Context
		kind ErrorKind
	}{
		{ctx: canceled, kind: ErrorKindCanceled},
		{ctx: deadline, kind: ErrorKindTimeout},
		{ctx: context.Background(), kind: ErrorKindTransport},
	}
	for _, test := range contextErrors {
		var providerErr *ProviderError
		if err := adapter.contextError(test.ctx, errors.New("request")); !errors.As(err, &providerErr) || providerErr.Kind != test.kind {
			t.Fatalf("context error = %v, want kind %s", err, test.kind)
		}
	}
	if got := adapter.error(ErrorKindTransport, 0, true, "", cause).Message; got != cause.Error() {
		t.Fatalf("derived provider message = %q", got)
	}
	if err := adapter.sleepBeforeRetry(canceled, 1); err == nil {
		t.Fatal("canceled retry sleep succeeded")
	}
	if adapter.statusError(http.StatusUnauthorized, nil) == nil {
		t.Fatal("status error is nil")
	}

	statuses := []int{
		http.StatusUnauthorized,
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusBadRequest,
		520,
		http.StatusInternalServerError,
		http.StatusTeapot,
	}
	for _, status := range statuses {
		kind, _ := classifyQNAStatus(status)
		if kind == "" {
			t.Fatalf("status %d was not classified", status)
		}
	}

	messages := map[string]string{
		"":                              "empty response",
		`{"detail":{"msg":"detail"}}`:   "detail",
		`{"message":"message"}`:         "message",
		`{"error":{"message":"error"}}`: "error",
		" plain ":                       "plain",
		strings.Repeat("x", 1201):       strings.Repeat("x", 1200) + "…",
	}
	for body, want := range messages {
		if got := responseMessage([]byte(body)); got != want {
			t.Fatalf("response message = %q, want %q", got, want)
		}
	}
	if limitCharacters("value", 0) != "" {
		t.Fatal("non-positive prompt limit was ignored")
	}
	if _, err := parseHTTPURL("://invalid"); err == nil {
		t.Fatal("invalid URL was accepted")
	}
	if _, err := readLimited(&coverageVideoReadCloser{readErr: errors.New("read")}, 1); err == nil {
		t.Fatal("reader failure was ignored")
	}
	if _, err := readLimited(strings.NewReader("too long"), 2); err == nil {
		t.Fatal("oversized response was accepted")
	}
	if retryDelayForAttempt(0, 1) != time.Second || retryDelayForAttempt(time.Second, 10) != 30*time.Second {
		t.Fatal("retry delay boundaries are incorrect")
	}
	if err := sleepWithContext(canceled, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleep cancellation = %v", err)
	}
	if err := sleepWithContext(context.Background(), time.Nanosecond); err != nil {
		t.Fatalf("short sleep = %v", err)
	}
	if endpointLogValue("://invalid") != "" {
		t.Fatal("invalid log endpoint was retained")
	}
	if qnaLogError(nil, "https://video.example.test") != nil {
		t.Fatal("nil log error became non-nil")
	}
}

func TestVideoCoverageNetworkTraceCallbacks(t *testing.T) {
	trace := newQNARequestTrace()
	clientTrace := trace.clientTrace()
	clientTrace.DNSStart(httptrace.DNSStartInfo{})
	clientTrace.DNSStart(httptrace.DNSStartInfo{})
	clientTrace.DNSDone(httptrace.DNSDoneInfo{})
	clientTrace.ConnectStart("tcp", "first")
	clientTrace.ConnectStart("tcp", "second")
	clientTrace.ConnectDone("tcp", "", nil)
	clientTrace.ConnectDone("tcp", "remote", nil)
	clientTrace.GotConn(httptrace.GotConnInfo{})
	left, right := net.Pipe()
	defer func() { _ = left.Close() }()
	defer func() { _ = right.Close() }()
	clientTrace.GotConn(httptrace.GotConnInfo{Conn: left, Reused: true, WasIdle: true})
	clientTrace.TLSHandshakeStart()
	clientTrace.TLSHandshakeDone(tls.ConnectionState{Version: tls.VersionTLS13, NegotiatedProtocol: "h2"}, nil)
	clientTrace.GotFirstResponseByte()

	snapshot := trace.snapshot(time.Now().Add(-time.Second), "")
	if snapshot.remoteAddr == "" || snapshot.tlsVersion != "TLS1.3" || snapshot.protocol != "h2" {
		t.Fatalf("network snapshot = %+v", snapshot)
	}
	var nilTrace *qnaNetworkTrace
	if got := nilTrace.snapshot(time.Now(), "HTTP/1.1"); got.protocol != "HTTP/1.1" {
		t.Fatalf("nil trace snapshot = %+v", got)
	}
	if durationMilliseconds(time.Now(), time.Now().Add(-time.Second)) != 0 {
		t.Fatal("negative trace duration was accepted")
	}
	if durationMilliseconds(time.Now().Add(-time.Millisecond), time.Time{}) < 0 {
		t.Fatal("open trace duration is negative")
	}
	for version, want := range map[uint16]string{
		tls.VersionTLS10: "TLS1.0",
		tls.VersionTLS11: "TLS1.1",
		tls.VersionTLS12: "TLS1.2",
		tls.VersionTLS13: "TLS1.3",
		0:                "",
	} {
		if got := tlsVersionName(version); got != want {
			t.Fatalf("TLS version %d = %q, want %q", version, got, want)
		}
	}
	ctx := context.Background()
	if withQNAHTTPTrace(ctx, nil) != ctx {
		t.Fatal("nil HTTP trace changed context")
	}
}

func TestVideoCoverageGenerateAndTaskFailures(t *testing.T) {
	adapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport")
	})
	if _, err := adapter.Generate(context.Background(), nil); err == nil {
		t.Fatal("nil request was accepted")
	}
	noKey := *adapter
	noKey.apiKey = ""
	if _, err := noKey.Generate(context.Background(), &ProviderRequest{}); err == nil {
		t.Fatal("empty API key was accepted")
	}
	if _, err := adapter.Generate(context.Background(), &ProviderRequest{Prompt: "test"}); err == nil {
		t.Fatal("missing reference image was accepted")
	}
	invalidBase := *adapter
	invalidBase.baseURL = "://invalid"
	if _, err := invalidBase.Generate(context.Background(), &ProviderRequest{Prompt: "test", StartImageURL: "image"}); err == nil {
		t.Fatal("invalid API base was accepted")
	}
	if _, err := adapter.Generate(context.Background(), &ProviderRequest{Prompt: "test", StartImageURL: "image"}); err == nil {
		t.Fatal("create transport failure was ignored")
	}

	emptyCreate := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusOK, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	if _, err := emptyCreate.Generate(context.Background(), &ProviderRequest{Prompt: "test", StartImageURL: "image"}); err == nil {
		t.Fatal("empty create response was accepted")
	}

	nonRetryable := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusBadRequest, io.NopCloser(strings.NewReader(`{"message":"bad"}`))), nil
	})
	if _, _, err := nonRetryable.createTask(context.Background(), "https://video.example.test/create", nil); err == nil {
		t.Fatal("non-retryable create status was accepted")
	}

	retryAdapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusServiceUnavailable, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	retryAdapter.maxRetries = 1
	retryAdapter.retryDelay = time.Hour
	canceled, cancel := context.WithCancel(context.Background())
	retryAdapter.httpClient.Transport = coverageVideoRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusServiceUnavailable, &coverageVideoReadCloser{
			data:    `{}`,
			readErr: io.EOF,
			onClose: cancel,
		}), nil
	})
	if _, _, err := retryAdapter.createTask(canceled, "https://video.example.test/create", nil); err == nil {
		t.Fatal("canceled create retry succeeded")
	}
}

func TestVideoCoveragePollAndWaitFailures(t *testing.T) {
	transportFailure := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport")
	})
	if _, _, _, err := transportFailure.pollTask(context.Background(), "https://video.example.test/poll", "id"); err == nil {
		t.Fatal("poll transport failure was ignored")
	}

	retryStatus := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusServiceUnavailable, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	if _, _, _, err := retryStatus.pollTask(context.Background(), "https://video.example.test/poll", "id"); err == nil {
		t.Fatal("final retryable poll status was accepted")
	}
	retryStatus.maxRetries = 1
	retryStatus.retryDelay = time.Hour
	canceledPoll, cancelPoll := context.WithCancel(context.Background())
	retryStatus.httpClient.Transport = coverageVideoRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusServiceUnavailable, &coverageVideoReadCloser{
			data:    `{}`,
			readErr: io.EOF,
			onClose: cancelPoll,
		}), nil
	})
	if _, _, _, err := retryStatus.pollTask(canceledPoll, "https://video.example.test/poll", "id"); err == nil {
		t.Fatal("canceled poll retry succeeded")
	}

	invalidBase := *retryStatus
	invalidBase.baseURL = "://invalid"
	if _, err := invalidBase.waitForTask(context.Background(), "id", "/requests"); err == nil {
		t.Fatal("invalid poll base was accepted")
	}
	canceledWait := *retryStatus
	canceledWait.pollInterval = time.Hour
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := canceledWait.waitForTask(canceled, "id", "/requests"); err == nil {
		t.Fatal("canceled poll sleep succeeded")
	}
	if _, err := transportFailure.waitForTask(context.Background(), "id", "/requests"); err == nil {
		t.Fatal("poll transport failure was ignored by wait")
	}

	responses := 0
	waitAdapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		responses++
		body := `{"status":"IN_PROGRESS"}`
		if responses == 2 {
			body = `{"video":{"url":"https://cdn.example.test/video.mp4"}}`
		}
		return coverageVideoResponse(http.StatusOK, io.NopCloser(strings.NewReader(body))), nil
	})
	result, err := waitAdapter.waitForTask(context.Background(), "id", "/requests")
	if err != nil || result.VideoURL == "" {
		t.Fatalf("wait result = %+v, error = %v", result, err)
	}

	failingWait := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusTeapot, io.NopCloser(strings.NewReader("teapot"))), nil
	})
	if _, err := failingWait.waitForTask(context.Background(), "id", "/requests"); err == nil {
		t.Fatal("unexpected poll status was accepted")
	}
}

func TestVideoCoverageDownloadFailuresAndRetries(t *testing.T) {
	adapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport")
	})
	if _, err := adapter.Download(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid download URL was accepted")
	}
	//nolint:staticcheck // This boundary test verifies that a nil context returns an error instead of panicking.
	if _, err := adapter.Download(nil, "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("nil download context was accepted")
	}
	if _, err := adapter.Download(context.Background(), "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("download transport failure was ignored")
	}

	canceledContext, cancelTransport := context.WithCancel(context.Background())
	transportRetry := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		cancelTransport()
		return nil, errors.New("transport")
	})
	transportRetry.maxRetries = 1
	transportRetry.retryDelay = time.Hour
	if _, err := transportRetry.Download(canceledContext, "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("canceled transport retry succeeded")
	}

	calls := 0
	transportSuccess := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("transport")
		}
		return coverageVideoResponse(http.StatusOK, io.NopCloser(strings.NewReader("video"))), nil
	})
	transportSuccess.maxRetries = 1
	if data, err := transportSuccess.Download(context.Background(), "https://cdn.example.test/video.mp4"); err != nil || string(data) != "video" {
		t.Fatalf("retried transport download = %q, error = %v", data, err)
	}

	statusFinal := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusBadRequest, io.NopCloser(strings.NewReader("bad"))), nil
	})
	if _, err := statusFinal.Download(context.Background(), "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("bad download status was accepted")
	}

	statusContext, cancelStatus := context.WithCancel(context.Background())
	statusRetry := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		body := &coverageVideoReadCloser{data: "busy", readErr: io.EOF, onRead: cancelStatus}
		return coverageVideoResponse(http.StatusServiceUnavailable, body), nil
	})
	statusRetry.maxRetries = 1
	statusRetry.retryDelay = time.Hour
	if _, err := statusRetry.Download(statusContext, "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("canceled status retry succeeded")
	}

	readFinal := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusOK, &coverageVideoReadCloser{readErr: errors.New("read")}), nil
	})
	if _, err := readFinal.Download(context.Background(), "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("download read failure was ignored")
	}

	readContext, cancelRead := context.WithCancel(context.Background())
	readRetry := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusOK, &coverageVideoReadCloser{
			readErr: errors.New("read"),
			onClose: cancelRead,
		}), nil
	})
	readRetry.maxRetries = 1
	readRetry.retryDelay = time.Hour
	if _, err := readRetry.Download(readContext, "https://cdn.example.test/video.mp4"); err == nil {
		t.Fatal("canceled read retry succeeded")
	}

	readCalls := 0
	readSuccess := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		readCalls++
		if readCalls == 1 {
			return coverageVideoResponse(http.StatusOK, &coverageVideoReadCloser{readErr: errors.New("read")}), nil
		}
		return coverageVideoResponse(http.StatusOK, io.NopCloser(strings.NewReader("video"))), nil
	})
	readSuccess.maxRetries = 1
	if data, err := readSuccess.Download(context.Background(), "https://cdn.example.test/video.mp4"); err != nil || string(data) != "video" {
		t.Fatalf("retried read download = %q, error = %v", data, err)
	}
}

func TestVideoCoverageDoJSONAndLoggingBranches(t *testing.T) {
	logs := &coverageVideoLogger{}
	adapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusOK, &coverageVideoReadCloser{readErr: errors.New("read")}), nil
	})
	adapter.logger = logs
	if _, _, err := adapter.doJSON(context.Background(), http.MethodGet, "://invalid", nil, qnaRequestTrace{}, nil); err == nil {
		t.Fatal("invalid JSON endpoint was accepted")
	}
	if _, _, err := adapter.doJSON(context.Background(), http.MethodGet, "https://video.example.test", nil, qnaRequestTrace{}, nil); err == nil {
		t.Fatal("JSON response read failure was ignored")
	}

	decodeAdapter := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return coverageVideoResponse(http.StatusOK, io.NopCloser(strings.NewReader("not-json"))), nil
	})
	decodeAdapter.logger = logs
	if _, _, err := decodeAdapter.doJSON(context.Background(), http.MethodGet, "https://video.example.test", nil, qnaRequestTrace{Stage: "poll"}, &qnaVideoResponse{}); err == nil {
		t.Fatal("invalid JSON response was accepted")
	}

	nilLogger := coverageVideoAdapter(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unused transport")
	})
	nilLogger.logRequestResult(qnaRequestTrace{}, http.MethodGet, "https://video.example.test", 200, time.Now(), "", "", "", 0, nil)
	nilLogger.logRequestFailure("failure", qnaRequestTrace{}, http.MethodGet, "https://video.example.test", 0, time.Now(), "", "", "", false, errors.New("failure"))
	adapter.logRequestResult(qnaRequestTrace{Stage: "download"}, http.MethodGet, "https://video.example.test/video", http.StatusOK, time.Now(), "", "", "", 1, nil)
	adapter.logRequestResult(qnaRequestTrace{Stage: "download"}, http.MethodGet, "https://video.example.test/video", http.StatusBadRequest, time.Now(), "", "", "", 1, nil)
}

func TestVideoCoverageProviderAndServiceBoundaries(t *testing.T) {
	provider := NewQNAProvider(QNAConfig{Models: []ModelConfig{
		{Name: "", Protocol: "fal_queue"},
		{Name: "model", Protocol: "fal_queue"},
	}})
	if len(provider.adapters) != 1 {
		t.Fatalf("adapter count = %d, want 1", len(provider.adapters))
	}
	if _, _, err := provider.route(nil); err == nil {
		t.Fatal("nil provider request was accepted")
	}

	wantErr := errors.New("provider failed")
	service := NewVideoGenerationService(&coverageVideoProvider{generateErr: wantErr})
	if _, err := service.Generate(context.Background(), &GenerateRequest{
		Prompt:     "test",
		StartImage: ReferenceImage{Base64: "image"},
	}); !errors.Is(err, wantErr) {
		t.Fatalf("service generate error = %v, want %v", err, wantErr)
	}
	if _, err := service.Download(context.Background(), " "); err == nil {
		t.Fatal("empty video URL was accepted")
	}
}
