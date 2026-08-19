package imageclient_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

type mockImageProvider struct {
	generateFunc func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error)
	editFunc     func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error)
	calls        int
}

func (m *mockImageProvider) Generate(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
	m.calls++
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &imageclient.ProviderResult{Images: []string{"img1"}}, nil
}

func (m *mockImageProvider) Edit(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
	m.calls++
	if m.editFunc != nil {
		return m.editFunc(ctx, req)
	}
	return &imageclient.ProviderResult{Images: []string{"img1"}}, nil
}

func TestFailoverProviderPrimarySuccess(t *testing.T) {
	primary := &mockImageProvider{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			if req.Model != "primary-model" {
				t.Fatalf("expected model primary-model, got %s", req.Model)
			}
			return &imageclient.ProviderResult{Images: []string{"primary-img"}}, nil
		},
	}
	fallback := &mockImageProvider{}

	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  "primary-model",
		FallbackModel: "fallback-model",
	})

	result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "test prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 || result.Images[0] != "primary-img" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback should not be called when primary succeeds")
	}
}

func TestFailoverProviderFailsOverOnTransientError(t *testing.T) {
	primary := &mockImageProvider{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, &imageclient.ProviderError{
				Kind:       imageclient.ErrorKindUnavailable,
				StatusCode: 503,
				Transient:  true,
				Message:    "model overloaded",
			}
		},
	}
	fallback := &mockImageProvider{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			if req.Model != "fallback-model" {
				t.Fatalf("expected fallback model, got %s", req.Model)
			}
			return &imageclient.ProviderResult{Images: []string{"fallback-img"}}, nil
		},
	}

	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  "primary-model",
		FallbackModel: "fallback-model",
	})

	result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "test prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 || result.Images[0] != "fallback-img" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if primary.calls != 1 || fallback.calls != 1 {
		t.Fatalf("expected 1 primary call and 1 fallback call, got primary=%d fallback=%d", primary.calls, fallback.calls)
	}
}

func TestFailoverProviderDoesNotFailOverOnPermanentError(t *testing.T) {
	primary := &mockImageProvider{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, &imageclient.ProviderError{
				Kind:       imageclient.ErrorKindInvalidRequest,
				StatusCode: 400,
				Transient:  false,
				Message:    "invalid prompt",
			}
		},
	}
	fallback := &mockImageProvider{}

	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  "primary-model",
		FallbackModel: "fallback-model",
	})

	_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "test prompt",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var providerErr *imageclient.ProviderError
	if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 permanent error, got %v", err)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback should not be called on permanent 400 error")
	}
}

func TestFailoverProviderPreservesBothErrorsAndUsesFallbackClassification(t *testing.T) {
	primaryErr := &imageclient.ProviderError{
		Provider: "primary", Kind: imageclient.ErrorKindUnavailable, Transient: true,
	}
	fallbackErr := &imageclient.ProviderError{
		Provider: "fallback", Kind: imageclient.ErrorKindAuthentication,
		StatusCode: http.StatusUnauthorized, Transient: false,
	}
	primary := &mockImageProvider{
		generateFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, primaryErr
		},
	}
	fallback := &mockImageProvider{
		generateFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, fallbackErr
		},
	}
	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary: primary, Fallback: fallback, FallbackModel: "fallback-model",
	})

	service := imageclient.NewImageGenerationService(provider)
	_, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt: "test", MaxAttempts: 2,
	})
	if err == nil {
		t.Fatal("expected both providers to fail")
	}
	if primary.calls != 1 || fallback.calls != 1 {
		t.Fatalf("permanent fallback failure was retried: primary=%d fallback=%d", primary.calls, fallback.calls)
	}
	if !errors.Is(err, primaryErr) || !errors.Is(err, fallbackErr) {
		t.Fatalf("error chain does not preserve both failures: %v", err)
	}
	if !imageclient.IsPermanent(err) || imageclient.IsTransient(err) {
		t.Fatalf("fallback classification was not preserved: %v", err)
	}
	if !strings.Contains(err.Error(), "primary failed") || !strings.Contains(err.Error(), "fallback (fallback-model) failed") {
		t.Fatalf("error lacks failover context: %v", err)
	}
}

func TestFailoverProviderEditPreservesTransientFallbackError(t *testing.T) {
	primaryErr := &imageclient.ProviderError{Kind: imageclient.ErrorKindTimeout, Transient: true}
	fallbackErr := &imageclient.ProviderError{Kind: imageclient.ErrorKindUnavailable, Transient: true}
	primary := &mockImageProvider{
		editFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, primaryErr
		},
	}
	fallback := &mockImageProvider{
		editFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, fallbackErr
		},
	}
	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary: primary, Fallback: fallback, FallbackModel: "fallback-model",
	})

	_, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
	if err == nil || !imageclient.IsTransient(err) || imageclient.IsPermanent(err) {
		t.Fatalf("error = %v, want transient fallback failure", err)
	}
	if !errors.Is(err, primaryErr) || !errors.Is(err, fallbackErr) {
		t.Fatalf("error chain does not preserve both failures: %v", err)
	}
}

func TestFailoverProviderSkipsFallbackAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	primary := &mockImageProvider{
		generateFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, &imageclient.ProviderError{Kind: imageclient.ErrorKindUnavailable, Transient: true}
		},
	}
	fallback := &mockImageProvider{}
	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{Primary: primary, Fallback: fallback})

	_, err := provider.Generate(ctx, &imageclient.ProviderRequest{Prompt: "test"})
	if err == nil || fallback.calls != 0 {
		t.Fatalf("canceled request used fallback: err=%v calls=%d", err, fallback.calls)
	}
}

func TestFailoverProviderEditFailover(t *testing.T) {
	primary := &mockImageProvider{
		editFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, &imageclient.ProviderError{
				Kind:       imageclient.ErrorKindRateLimited,
				StatusCode: 429,
				Transient:  true,
				Message:    "rate limited",
			}
		},
	}
	fallback := &mockImageProvider{
		editFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return &imageclient.ProviderResult{Images: []string{"fallback-edit-img"}}, nil
		},
	}

	provider := imageclient.NewFailoverImageProvider(imageclient.FailoverConfig{
		Primary:       primary,
		Fallback:      fallback,
		PrimaryModel:  "primary-model",
		FallbackModel: "fallback-model",
	})

	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "test edit",
		ReferenceImages: []string{"https://example.com/ref.png"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 || result.Images[0] != "fallback-edit-img" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
