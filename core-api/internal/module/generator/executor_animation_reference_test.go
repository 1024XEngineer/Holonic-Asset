package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func TestLoadAnimationReferenceRetriesConnectionResetAndLogsAttempt(t *testing.T) {
	encoded := strings.TrimPrefix(animationTestOpaquePrototype(t), "data:image/png;base64,")
	imageBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode test image: %v", err)
	}

	calls := 0
	client := &http.Client{Transport: animationRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		body := io.ReadCloser(io.NopCloser(bytes.NewReader(imageBytes)))
		if calls == 1 {
			body = io.NopCloser(io.MultiReader(
				bytes.NewReader(imageBytes[:len(imageBytes)/2]),
				animationErrorReader{err: syscall.ECONNRESET},
			))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Server":  []string{"qiniu-test"},
				"X-Reqid": []string{"qiniu-request-123"},
			},
			Body: body,
		}, nil
	})}
	logs := &animationRecordingLogger{}
	service := &animationGenerationService{
		referenceHTTPClient: client,
		referenceMaxRetries: 1,
		referenceTimeout:    time.Second,
		referenceRetryDelay: time.Nanosecond,
		logger:              logs,
	}

	result, err := service.loadAnimationReference(
		context.Background(),
		"https://cdn.example.test/uploads/reference.png?X-Amz-Signature=secret",
	)
	if err != nil {
		t.Fatalf("load animation reference: %v", err)
	}
	if result == "" || calls != 2 {
		t.Fatalf("unexpected retry result: calls=%d result_bytes=%d", calls, len(result))
	}

	entry, ok := logs.find("animation reference response read failure")
	if !ok {
		t.Fatal("expected response read failure log")
	}
	for key, want := range map[string]any{
		"provider":            "reference_storage",
		"stage":               "download_animation_reference",
		"source_host":         "cdn.example.test",
		"endpoint":            "/uploads/reference.png",
		"attempt":             1,
		"max_attempts":        2,
		"status_code":         http.StatusOK,
		"upstream_request_id": "qiniu-request-123",
		"will_retry":          true,
		"error_kind":          "response_read",
		"transient":           true,
	} {
		if got := entry.fields[key]; got != want {
			t.Errorf("log field %q = %#v, want %#v", key, got, want)
		}
	}
	if strings.Contains(entry.fields["endpoint"].(string), "Signature") {
		t.Fatal("signed query must not be written to logs")
	}
}

func TestDefaultAnimationReferenceClientsUseIndependentConnectionPools(t *testing.T) {
	first := NewAnimationGenerationService(nil, nil).(*animationGenerationService)
	second := NewAnimationGenerationService(nil, nil).(*animationGenerationService)
	if first.referenceHTTPClient == second.referenceHTTPClient {
		t.Fatal("animation reference clients must not be shared")
	}
	firstTransport, firstOK := first.referenceHTTPClient.Transport.(*http.Transport)
	secondTransport, secondOK := second.referenceHTTPClient.Transport.(*http.Transport)
	if !firstOK || !secondOK || firstTransport == secondTransport || firstTransport == http.DefaultTransport || secondTransport == http.DefaultTransport {
		t.Fatal("animation reference clients must have independent transports")
	}
	if first.referenceHTTPClient.Timeout != defaultAnimationReferenceTimeout {
		t.Fatalf("reference client timeout = %s, want %s", first.referenceHTTPClient.Timeout, defaultAnimationReferenceTimeout)
	}
	if firstTransport.TLSHandshakeTimeout != defaultAnimationReferenceTimeout ||
		secondTransport.TLSHandshakeTimeout != defaultAnimationReferenceTimeout {
		t.Fatalf(
			"reference TLS handshake timeouts = %s and %s, want %s",
			firstTransport.TLSHandshakeTimeout,
			secondTransport.TLSHandshakeTimeout,
			defaultAnimationReferenceTimeout,
		)
	}
}

type animationRoundTripFunc func(*http.Request) (*http.Response, error)

func (function animationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type animationErrorReader struct{ err error }

func (reader animationErrorReader) Read([]byte) (int, error) { return 0, reader.err }

type animationRecordedLog struct {
	message string
	fields  map[string]any
}

type animationRecordingLogger struct {
	mu      sync.Mutex
	entries []animationRecordedLog
}

func (l *animationRecordingLogger) Debug(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *animationRecordingLogger) Info(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *animationRecordingLogger) Warn(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *animationRecordingLogger) Error(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (*animationRecordingLogger) Sync() error { return nil }

func (l *animationRecordingLogger) record(message string, fields []logger.Field) {
	values := make(map[string]any, len(fields))
	for _, field := range fields {
		values[field.Key] = field.Val
	}
	l.mu.Lock()
	l.entries = append(l.entries, animationRecordedLog{message: message, fields: values})
	l.mu.Unlock()
}

func (l *animationRecordingLogger) find(message string) (animationRecordedLog, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.entries {
		if entry.message == message {
			return entry, true
		}
	}
	return animationRecordedLog{}, false
}
