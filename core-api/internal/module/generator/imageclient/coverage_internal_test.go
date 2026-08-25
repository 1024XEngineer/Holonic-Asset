package imageclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
)

type coverageImageAdapter struct {
	editErr error
}

func (*coverageImageAdapter) Generate(context.Context, *ProviderRequest) (*ProviderResult, error) {
	return &ProviderResult{Images: []string{"image"}}, nil
}

func (a *coverageImageAdapter) Edit(context.Context, *ProviderRequest) (*ProviderResult, error) {
	return nil, a.editErr
}

type coverageImageProvider struct {
	cancel context.CancelFunc
}

func (p *coverageImageProvider) Generate(context.Context, *ProviderRequest) (*ProviderResult, error) {
	if p.cancel != nil {
		p.cancel()
	}
	return nil, &ProviderError{Kind: ErrorKindUnavailable, Transient: true, Message: "retry"}
}

func (*coverageImageProvider) Edit(context.Context, *ProviderRequest) (*ProviderResult, error) {
	return nil, errors.New("unexpected edit")
}

func TestImageCoverageLegacyFactoryFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/images/generations":
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte(`{"error":{"message":"overloaded"}}`))
		case "/v1/chat/completions":
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,aW1hZ2U="}}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := NewImageProvider(FactoryConfig{
		BaseURL:       server.URL,
		DefaultModel:  "openai/gpt-image-2",
		FallbackModel: "google/gemini-image",
	})
	result, err := provider.Generate(context.Background(), &ProviderRequest{Prompt: "test"})
	if err != nil || len(result.Images) != 1 {
		t.Fatalf("legacy fallback result = %+v, error = %v", result, err)
	}
}

func TestImageCoverageProtocolHelpers(t *testing.T) {
	adapter := NewQNAChatCompletionsAdapter(QNAChatCompletionsAdapterConfig{})
	if _, err := adapter.extractImages(context.Background(), []chatChoice{{
		Message: chatMessage{Content: "data:image/png;base64,not-base64"},
	}}); err == nil {
		t.Fatal("invalid inline image was accepted")
	}

	for _, value := range []string{"not-a-data-url", "data:image/png;base64,"} {
		if _, err := parseImageDataURL(value); err == nil {
			t.Fatalf("invalid data URL %q was accepted", value)
		}
	}
	if got := chatErrorMessage([]byte(`{"error":{"message":"nested"}}`)); got != "nested" {
		t.Fatalf("nested error message = %q", got)
	}

	dialer := generatedImageDialer{}
	addresses, err := dialer.resolve(context.Background(), "127.0.0.1")
	if err != nil || len(addresses) != 1 || addresses[0].String() != "127.0.0.1" {
		t.Fatalf("literal address resolution = %v, error = %v", addresses, err)
	}

	providerErr := &ProviderError{Kind: ErrorKindInvalidRequest, StatusCode: http.StatusUnprocessableEntity}
	if shouldRetryQNAEditWithoutMask(&ProviderRequest{MaskImage: "mask"}, providerErr) {
		t.Fatal("non-400 mask error was retried")
	}
	if got := normalizeQNAImageSize("invalid"); got != "invalid" {
		t.Fatalf("invalid size = %q", got)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	classifications := []error{
		classifyQNARequestError(canceled, errors.New("transport")),
		classifyQNARequestError(context.Background(), context.DeadlineExceeded),
		classifyQNARequestError(context.Background(), errors.New("transport")),
	}
	wantKinds := []ErrorKind{ErrorKindCanceled, ErrorKindTimeout, ErrorKindTransport}
	for index, err := range classifications {
		var classified *ProviderError
		if !errors.As(err, &classified) || classified.Kind != wantKinds[index] {
			t.Fatalf("classification %d = %v, want %s", index, err, wantKinds[index])
		}
	}
	if kind, transient := classifyQNAStatus(http.StatusTeapot); kind != ErrorKindInvalidRequest || transient {
		t.Fatalf("teapot classification = (%s, %t)", kind, transient)
	}
	cause := errors.New("cause message")
	if got := newQNAError(ErrorKindTransport, 0, true, "", cause).Message; got != cause.Error() {
		t.Fatalf("derived message = %q", got)
	}
}

func TestImageCoverageGatewayEditFailure(t *testing.T) {
	wantErr := errors.New("edit failed")
	provider := &QNAProvider{
		defaultModel: "model",
		adapters: map[string]protocolAdapter{
			"model": &coverageImageAdapter{editErr: wantErr},
		},
	}
	if _, err := provider.Edit(context.Background(), &ProviderRequest{}); !errors.Is(err, wantErr) {
		t.Fatalf("edit error = %v, want %v", err, wantErr)
	}
	if _, err := provider.Edit(context.Background(), nil); err == nil {
		t.Fatal("nil edit request was accepted")
	}
}

func TestImageCoverageSDKErrorClassification(t *testing.T) {
	apiErr := &qnasdk.Error{StatusCode: http.StatusInternalServerError}
	status, message, ok := qnaSDKAPIError(apiErr)
	if !ok || status != http.StatusInternalServerError || message == "" {
		t.Fatalf("SDK API error = (%d, %q, %t)", status, message, ok)
	}
	if isQNASDKConfigurationError(nil) || isQNASDKResponseDecodeError(nil) {
		t.Fatal("nil SDK error was classified")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	imageErrors := []error{
		classifyQNAImageSDKError(canceled, errors.New("request")),
		classifyQNAImageSDKError(context.Background(), errors.New("WithBaseURL failed")),
		classifyQNAImageSDKError(context.Background(), errors.New("error parsing response JSON")),
		classifyQNAImageSDKError(context.Background(), errors.New("transport")),
	}
	chatErrors := []error{
		classifyChatSDKError(canceled, errors.New("request")),
		classifyChatSDKError(context.Background(), errors.New("transport")),
	}
	for _, err := range append(imageErrors, chatErrors...) {
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) {
			t.Fatalf("SDK classification = %T, want ProviderError", err)
		}
	}
}

func TestImageCoverageCanceledRetryBackoff(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	provider := &coverageImageProvider{}
	_, err := NewImageGenerationService(provider).Generate(ctx, &GenerateRequest{
		Prompt:      "test",
		MaxAttempts: 2,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("retry error = %v, want deadline exceeded", err)
	}

	canceled, stop := context.WithCancel(context.Background())
	stop()
	if err := backoffSleep(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("backoff error = %v", err)
	}
}
