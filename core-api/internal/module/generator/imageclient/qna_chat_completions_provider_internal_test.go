package imageclient

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type internalRoundTripFunc func(*http.Request) (*http.Response, error)

func (f internalRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReader) Close() error             { return nil }

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

func TestNewQNAChatCompletionsProviderDefaults(t *testing.T) {
	provider := NewQNAChatCompletionsProvider(QNAChatCompletionsConfig{})
	if provider.baseURL != DefaultQNABaseURL || provider.defaultModel != DefaultQNAChatCompletionsModel {
		t.Fatalf("unexpected defaults: base=%q model=%q", provider.baseURL, provider.defaultModel)
	}
	if provider.httpClient.Timeout != defaultChatHTTPTimeout || provider.downloadHTTPClient == nil {
		t.Fatalf("default clients are not configured: %+v", provider)
	}
}

func TestQNAChatCompletionsProviderRejectsInvalidEndpoint(t *testing.T) {
	provider := NewQNAChatCompletionsProvider(QNAChatCompletionsConfig{BaseURL: "://invalid"})
	_, err := provider.Generate(context.Background(), &ProviderRequest{Prompt: "test"})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != ErrorKindInvalidRequest {
		t.Fatalf("generate error = %v, want invalid request", err)
	}
}

func TestClassifyChatRequestError(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		err       error
		wantKind  ErrorKind
		transient bool
	}{
		{
			name: "canceled context", ctx: canceledContext(), err: errors.New("request failed"),
			wantKind: ErrorKindCanceled, transient: false,
		},
		{
			name: "expired context", ctx: expiredContext(), err: errors.New("request failed"),
			wantKind: ErrorKindTimeout, transient: true,
		},
		{
			name: "client timeout", ctx: context.Background(),
			err:      &url.Error{Op: "Get", URL: "https://images.example", Err: timeoutError{}},
			wantKind: ErrorKindTimeout, transient: true,
		},
		{
			name: "transport failure", ctx: context.Background(), err: errors.New("connection refused"),
			wantKind: ErrorKindTransport, transient: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			providerErr := classifyChatRequestError(test.ctx, test.err)
			if providerErr.Kind != test.wantKind || providerErr.Transient != test.transient {
				t.Fatalf("classification = (%s, %t), want (%s, %t)",
					providerErr.Kind, providerErr.Transient, test.wantKind, test.transient)
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}

func TestClassifyChatStatusCoversProviderResponses(t *testing.T) {
	tests := []struct {
		statusCode int
		wantKind   ErrorKind
		transient  bool
	}{
		{statusCode: http.StatusUnprocessableEntity, wantKind: ErrorKindInvalidRequest},
		{statusCode: http.StatusForbidden, wantKind: ErrorKindAuthentication},
		{statusCode: http.StatusRequestTimeout, wantKind: ErrorKindTimeout, transient: true},
		{statusCode: http.StatusInternalServerError, wantKind: ErrorKindUnavailable, transient: true},
		{statusCode: http.StatusBadGateway, wantKind: ErrorKindUnavailable, transient: true},
		{statusCode: http.StatusGatewayTimeout, wantKind: ErrorKindUnavailable, transient: true},
		{statusCode: 599, wantKind: ErrorKindUnavailable, transient: true},
		{statusCode: http.StatusTeapot, wantKind: ErrorKindInvalidResponse},
	}

	for _, test := range tests {
		kind, transient := classifyChatStatus(test.statusCode)
		if kind != test.wantKind || transient != test.transient {
			t.Fatalf("status %d = (%s, %t), want (%s, %t)",
				test.statusCode, kind, transient, test.wantKind, test.transient)
		}
	}
}

func TestChatErrorMessageShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "top-level message", body: `{"message":"top level"}`, want: "top level"},
		{name: "nested text", body: `{"error":"nested text"}`, want: "nested text"},
		{name: "plain text", body: "  upstream unavailable  ", want: "upstream unavailable"},
		{name: "empty object", body: `{}`, want: `{}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chatErrorMessage([]byte(test.body)); got != test.want {
				t.Fatalf("message = %q, want %q", got, test.want)
			}
		})
	}

	response := &http.Response{
		StatusCode: http.StatusTeapot,
		Status:     "418 I'm a teapot",
		Body:       io.NopCloser(strings.NewReader("")),
	}
	err := chatStatusError(response)
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Message != response.Status {
		t.Fatalf("status error = %v, want response status message", err)
	}
}

func TestChatMessageUnmarshalShapes(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		var message chatMessage
		if err := message.UnmarshalJSON([]byte(`{`)); err == nil {
			t.Fatal("malformed message was accepted")
		}
	})

	tests := []struct {
		name        string
		body        string
		wantContent any
		wantParts   int
	}{
		{name: "missing content", body: `{"role":"assistant"}`, wantContent: ""},
		{name: "string", body: `{"content":"hello"}`, wantContent: "hello"},
		{name: "parts", body: `{"content":[{"type":"text","text":"hello"}]}`, wantParts: 1},
		{name: "unknown object", body: `{"content":{"value":1}}`, wantContent: `{"value":1}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var message chatMessage
			if err := message.UnmarshalJSON([]byte(test.body)); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			if test.wantParts > 0 {
				if len(message.ContentParts) != test.wantParts || message.contentString() != "" {
					t.Fatalf("unexpected content parts: %+v", message)
				}
				return
			}
			if message.Content != test.wantContent {
				t.Fatalf("content = %#v, want %#v", message.Content, test.wantContent)
			}
		})
	}
}

func TestChatCompletionsImageReferenceHelpers(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
	if got := formatChatImageRef(""); got != "" {
		t.Fatalf("empty reference = %q", got)
	}
	if got := formatChatImageRef(raw); got != "data:image/png;base64,"+raw {
		t.Fatalf("raw base64 reference = %q", got)
	}
	if got := formatChatImageRef("asset-id"); got != "asset-id" {
		t.Fatalf("opaque reference = %q", got)
	}
	if got := stripDataURLPrefix("not-a-data-url"); got != "not-a-data-url" {
		t.Fatalf("plain value = %q", got)
	}
	if !isLikelyBase64("  " + raw + "\n") {
		t.Fatal("whitespace-wrapped base64 was rejected")
	}
}

func TestQNAChatCompletionsProviderImageResolutionFailures(t *testing.T) {
	baseProvider := func(roundTrip internalRoundTripFunc) *QNAChatCompletionsProvider {
		return NewQNAChatCompletionsProvider(QNAChatCompletionsConfig{
			BaseURL:            "https://api.example",
			DownloadHTTPClient: &http.Client{Transport: roundTrip},
		})
	}

	t.Run("raw base64", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
		provider := baseProvider(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("download must not run")
		})
		got, err := provider.resolveImageToB64(context.Background(), raw)
		if err != nil || got != raw {
			t.Fatalf("raw base64 result = %q, error = %v", got, err)
		}
	})

	t.Run("malformed URL", func(t *testing.T) {
		provider := baseProvider(nil)
		_, err := provider.resolveImageToB64(context.Background(), "http://[::1")
		if err == nil {
			t.Fatal("malformed URL was accepted")
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		provider := baseProvider(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset")
		})
		_, err := provider.resolveImageToB64(context.Background(), "https://images.example/output.png")
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || providerErr.Kind != ErrorKindTransport {
			t.Fatalf("download error = %v, want transport error", err)
		}
	})

	t.Run("body read failure", func(t *testing.T) {
		provider := baseProvider(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       failingReader{},
				Request:    request,
			}, nil
		})
		_, err := provider.resolveImageToB64(context.Background(), "https://images.example/output.png")
		if err == nil || !strings.Contains(err.Error(), "read downloaded image data") {
			t.Fatalf("download error = %v, want body read failure", err)
		}
	})

	t.Run("streamed response exceeds limit", func(t *testing.T) {
		provider := baseProvider(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{"image/png"}},
				Body:          io.NopCloser(io.LimitReader(zeroReader{}, maxGeneratedImageBytes+1)),
				ContentLength: -1,
				Request:       request,
			}, nil
		})
		_, err := provider.resolveImageToB64(context.Background(), "https://images.example/output.png")
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("download error = %v, want size rejection", err)
		}
	})
}

func TestQNAChatCompletionsProviderExtractImageBranches(t *testing.T) {
	provider := NewQNAChatCompletionsProvider(QNAChatCompletionsConfig{
		BaseURL: "https://api.example",
		DownloadHTTPClient: &http.Client{Transport: internalRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(strings.NewReader("image")),
				Request:    request,
			}, nil
		})},
	})

	tests := []struct {
		name    string
		choice  chatChoice
		wantErr bool
	}{
		{
			name: "images field error",
			choice: chatChoice{Message: chatMessage{Images: []chatContentPart{{
				Type: "image_url", ImageURL: &chatImageURL{URL: "http://127.0.0.1/image.png"},
			}}}},
			wantErr: true,
		},
		{
			name: "content parts error",
			choice: chatChoice{Message: chatMessage{ContentParts: []chatContentPart{{
				Type: "image_url", ImageURL: &chatImageURL{URL: "http://127.0.0.1/image.png"},
			}}}},
			wantErr: true,
		},
		{
			name:    "markdown error",
			choice:  chatChoice{Message: chatMessage{Content: "![image](http://127.0.0.1/image.png)"}},
			wantErr: true,
		},
		{
			name:    "plain URL error",
			choice:  chatChoice{Message: chatMessage{Content: "http://127.0.0.1/image.png"}},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := provider.extractImages(context.Background(), []chatChoice{test.choice})
			if (err != nil) != test.wantErr {
				t.Fatalf("extract error = %v, wantErr = %t", err, test.wantErr)
			}
		})
	}

	images, err := provider.extractImages(context.Background(), []chatChoice{{
		Message: chatMessage{Content: "download https://images.example/output.png"},
	}})
	if err != nil || len(images) != 1 {
		t.Fatalf("plain URL images = %v, error = %v", images, err)
	}
}
