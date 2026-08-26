package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

type mockReferenceResolver struct {
	resolveFunc func(ctx context.Context, reference string) (string, error)
}

func (m *mockReferenceResolver) ResolveReference(ctx context.Context, reference string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, reference)
	}
	return reference, nil
}

func TestLoadAnimationReference(t *testing.T) {
	png1x1Bytes, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")

	t.Run("empty reference", func(t *testing.T) {
		svc := &animationGenerationService{}
		_, err := svc.loadAnimationReference(context.Background(), "")
		if err == nil || err.Error() != "generator: animation reference image is required" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resolver error", func(t *testing.T) {
		svc := &animationGenerationService{
			referenceResolver: &mockReferenceResolver{
				resolveFunc: func(ctx context.Context, reference string) (string, error) {
					return "", errors.New("resolver failure")
				},
			},
		}
		_, err := svc.loadAnimationReference(context.Background(), "obj-key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("resolver returns empty string", func(t *testing.T) {
		svc := &animationGenerationService{
			referenceResolver: &mockReferenceResolver{
				resolveFunc: func(ctx context.Context, reference string) (string, error) {
					return "   ", nil
				},
			},
		}
		_, err := svc.loadAnimationReference(context.Background(), "obj-key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-http reference returns canonical string", func(t *testing.T) {
		svc := &animationGenerationService{}
		raw := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png1x1Bytes)
		got, err := svc.loadAnimationReference(context.Background(), raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != raw {
			t.Fatalf("got %q, want %q", got, raw)
		}
	})

	t.Run("malformed url", func(t *testing.T) {
		svc := &animationGenerationService{}
		_, err := svc.loadAnimationReference(context.Background(), "http://::invalid-url")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("url without host", func(t *testing.T) {
		svc := &animationGenerationService{}
		_, err := svc.loadAnimationReference(context.Background(), "http:///path/only")
		if err == nil {
			t.Fatal("expected host required error")
		}
	})

	t.Run("successful http download", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("X-Reqid", "req-12345")
			_, _ = w.Write(png1x1Bytes)
		}))
		defer server.Close()

		logger := &recordingLogger{}
		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
			logger:              logger,
		}
		canonical, err := svc.loadAnimationReference(context.Background(), server.URL+"/test.png")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if canonical == "" {
			t.Fatal("expected non-empty canonical base64")
		}
		if len(logger.entries) == 0 {
			t.Fatal("expected logs recorded")
		}
	})

	t.Run("empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
		}
		_, err := svc.loadAnimationReference(context.Background(), server.URL)
		if err == nil {
			t.Fatal("expected error for empty response body")
		}
	})

	t.Run("corrupt image body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("not an image"))
		}))
		defer server.Close()

		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
		}
		_, err := svc.loadAnimationReference(context.Background(), server.URL)
		if err == nil {
			t.Fatal("expected error decoding image")
		}
	})

	t.Run("transient status retry then success", func(t *testing.T) {
		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := attempts.Add(1)
			if count == 1 {
				w.Header().Set("Request-Id", "retry-req")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limited"))
				return
			}
			w.Header().Set("X-Request-Id", "ok-req")
			_, _ = w.Write(png1x1Bytes)
		}))
		defer server.Close()

		logger := &recordingLogger{}
		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
			referenceMaxRetries: 2,
			referenceRetryDelay: time.Millisecond,
			logger:              logger,
		}
		canonical, err := svc.loadAnimationReference(context.Background(), server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if canonical == "" {
			t.Fatal("expected non-empty canonical base64")
		}
		if attempts.Load() != 2 {
			t.Fatalf("expected 2 attempts, got %d", attempts.Load())
		}
	})

	t.Run("non-transient status fail fast", func(t *testing.T) {
		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer server.Close()

		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
			referenceMaxRetries: 3,
			referenceRetryDelay: time.Millisecond,
		}
		_, err := svc.loadAnimationReference(context.Background(), server.URL)
		if err == nil {
			t.Fatal("expected error")
		}
		if attempts.Load() != 1 {
			t.Fatalf("expected exactly 1 attempt, got %d", attempts.Load())
		}
	})

	t.Run("transport error retry exhausted", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		serverURL := server.URL
		server.Close()

		parsed, _ := url.Parse(serverURL)
		logger := &recordingLogger{}
		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
			referenceMaxRetries: 1,
			referenceRetryDelay: time.Millisecond,
			logger:              logger,
		}
		_, err := svc.downloadAnimationReference(context.Background(), parsed)
		if err == nil {
			t.Fatal("expected transport error")
		}
	})

	t.Run("retry canceled by context", func(t *testing.T) {
		var attempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		svc := &animationGenerationService{
			referenceHTTPClient: server.Client(),
			referenceMaxRetries: 2,
			referenceRetryDelay: 50 * time.Millisecond,
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		parsed, _ := url.Parse(server.URL)
		_, err := svc.downloadAnimationReference(ctx, parsed)
		if err == nil {
			t.Fatal("expected canceled error")
		}
	})

	t.Run("default timeout, retries and client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png1x1Bytes)
		}))
		defer server.Close()

		svc := &animationGenerationService{
			referenceHTTPClient: nil, // defaults to newDefaultAnimationReferenceHTTPClient()
			referenceMaxRetries: 0,
			referenceTimeout:    0,
			referenceRetryDelay: 0,
		}
		canonical, err := svc.loadAnimationReference(context.Background(), server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if canonical == "" {
			t.Fatal("expected non-empty result")
		}
	})

	t.Run("log failure kinds: timeout and canceled", func(t *testing.T) {
		rec := &recordingLogger{}
		svc := &animationGenerationService{logger: rec}
		parsed, _ := url.Parse("http://example.com/test.png")
		startedAt := time.Now()

		svc.logAnimationReferenceFailure("timeout err", parsed, 1, 3, 0, startedAt, 0, "server", "req-1", true, context.DeadlineExceeded)
		svc.logAnimationReferenceFailure("canceled err", parsed, 1, 3, 0, startedAt, 0, "server", "req-1", false, context.Canceled)

		if len(rec.entries) != 2 {
			t.Fatalf("expected 2 log entries, got %d", len(rec.entries))
		}
	})
}
