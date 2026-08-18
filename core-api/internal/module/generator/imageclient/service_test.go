package imageclient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

type imageProviderStub struct {
	generateCalls int
	editCalls     int
	generateFunc  func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error)
	editFunc      func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error)
}

func (s *imageProviderStub) Generate(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
	s.generateCalls++
	if s.generateFunc != nil {
		return s.generateFunc(ctx, req)
	}
	return &imageclient.ProviderResult{Images: []string{"cG5n"}}, nil
}

func (s *imageProviderStub) Edit(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
	s.editCalls++
	if s.editFunc != nil {
		return s.editFunc(ctx, req)
	}
	return &imageclient.ProviderResult{Images: []string{"cG5n"}}, nil
}

func TestImageGenerationServiceRetriesTransientErrorsUntilSuccess(t *testing.T) {
	transientErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindUnavailable,
		StatusCode: 525,
		Transient:  true,
		Message:    "SSL Handshake Failed",
	}

	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			if callCount < 2 {
				return nil, transientErr
			}
			return &imageclient.ProviderResult{Images: []string{"c3VjY2Vzcw=="}}, nil
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	res, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt:      "hero character",
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("expected success on second attempt, got: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls to provider, got: %d", callCount)
	}
	if len(res.Images) != 1 || res.Images[0].Base64 != "c3VjY2Vzcw==" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestImageGenerationServiceRetriesEmptyResultsUntilSuccess(t *testing.T) {
	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			if callCount == 1 {
				return &imageclient.ProviderResult{}, nil
			}
			return &imageclient.ProviderResult{Images: []string{"c3VjY2Vzcw=="}}, nil
		},
	}

	result, err := imageclient.NewImageGenerationService(provider).Generate(
		context.Background(),
		&imageclient.GenerateRequest{Prompt: "hero character", MaxAttempts: 2},
	)
	if err != nil {
		t.Fatalf("retry empty result: %v", err)
	}
	if callCount != 2 || len(result.Images) != 1 || result.Images[0].Base64 != "c3VjY2Vzcw==" {
		t.Fatalf("unexpected retry result: calls=%d result=%+v", callCount, result)
	}
}

func TestImageGenerationServiceStopsAtMaxAttempts(t *testing.T) {
	transientErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindUnavailable,
		StatusCode: 525,
		Transient:  true,
		Message:    "SSL Handshake Failed",
	}

	for _, maxAttempts := range []int{2, 3} {
		callCount := 0
		provider := &imageProviderStub{
			generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
				callCount++
				return nil, transientErr
			},
		}

		service := imageclient.NewImageGenerationService(provider)
		_, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
			Prompt:      "hero character",
			MaxAttempts: maxAttempts,
		})
		if err == nil {
			t.Fatalf("expected error after %d attempts, got nil", maxAttempts)
		}
		if callCount != maxAttempts {
			t.Fatalf("expected %d calls, got %d", maxAttempts, callCount)
		}
	}
}

func TestImageGenerationServiceFailsImmediatelyOnPermanentError(t *testing.T) {
	permanentErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindInvalidRequest,
		StatusCode: 400,
		Transient:  false,
		Message:    "invalid prompt",
	}

	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			return nil, permanentErr
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	_, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt:      "bad prompt",
		MaxAttempts: 3,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 call without retrying permanent error, got %d", callCount)
	}
}

func TestImageGenerationServiceFailsImmediatelyOnUnclassifiedError(t *testing.T) {
	wantErr := errors.New("unclassified provider failure")
	provider := &imageProviderStub{
		generateFunc: func(context.Context, *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			return nil, wantErr
		},
	}

	_, err := imageclient.NewImageGenerationService(provider).Generate(
		context.Background(),
		&imageclient.GenerateRequest{Prompt: "hero character", MaxAttempts: 3},
	)
	if !errors.Is(err, wantErr) || provider.generateCalls != 1 {
		t.Fatalf("unclassified error should not retry: calls=%d err=%v", provider.generateCalls, err)
	}
}

func TestImageGenerationServiceRespectsContextCancellation(t *testing.T) {
	transientErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindUnavailable,
		StatusCode: 525,
		Transient:  true,
		Message:    "SSL Handshake Failed",
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(c context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			cancel() // Cancel context during first attempt
			return nil, transientErr
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	_, err := service.Generate(ctx, &imageclient.GenerateRequest{
		Prompt:      "hero character",
		MaxAttempts: 3,
	})
	if !errors.Is(err, context.Canceled) && !errors.Is(err, transientErr) {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call before cancellation, got %d", callCount)
	}
}

func TestImageGenerationServiceRetriesEditCallsUntilSuccess(t *testing.T) {
	transientErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindUnavailable,
		StatusCode: 502,
		Transient:  true,
		Message:    "Bad Gateway",
	}

	callCount := 0
	provider := &imageProviderStub{
		editFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			if callCount < 2 {
				return nil, transientErr
			}
			return &imageclient.ProviderResult{
				Images:       []string{"ZWRpdGVk"},
				OutputFormat: "webp",
				Size:         "1024x1024",
			}, nil
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	res, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt:          "modify character",
		ReferenceImages: []string{"ref1"},
		Params:          imageclient.Params{"quality": "hd"},
		MaxAttempts:     2,
	})
	if err != nil {
		t.Fatalf("expected success on second attempt, got: %v", err)
	}
	if callCount != 2 || provider.editCalls != 2 {
		t.Fatalf("expected 2 edit calls, got %d", callCount)
	}
	if len(res.Images) != 1 || res.Images[0].Base64 != "ZWRpdGVk" || res.Images[0].MediaType != "image/webp" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestImageGenerationServiceRetriesWhenImageContainsEmptyString(t *testing.T) {
	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			if callCount == 1 {
				return &imageclient.ProviderResult{Images: []string{""}}, nil
			}
			return &imageclient.ProviderResult{
				Images:       []string{"dmFsaWQ="},
				OutputFormat: "jpg",
			}, nil
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	res, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt:      "generate scenery",
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatalf("expected retry after empty image, got: %v", err)
	}
	if callCount != 2 || len(res.Images) != 1 || res.Images[0].Base64 != "dmFsaWQ=" || res.Images[0].MediaType != "image/jpeg" {
		t.Fatalf("unexpected result: calls=%d res=%+v", callCount, res)
	}
}

func TestImageGenerationServiceDefaultsMaxAttemptsTo1WhenUnset(t *testing.T) {
	transientErr := &imageclient.ProviderError{
		Kind:       imageclient.ErrorKindUnavailable,
		StatusCode: 525,
		Transient:  true,
		Message:    "SSL Handshake Failed",
	}

	callCount := 0
	provider := &imageProviderStub{
		generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
			callCount++
			return nil, transientErr
		},
	}

	service := imageclient.NewImageGenerationService(provider)
	_, err := service.Generate(context.Background(), &imageclient.GenerateRequest{
		Prompt:      "hero character",
		MaxAttempts: 0, // Should default to 1 attempt
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call with default MaxAttempts, got %d", callCount)
	}
}

func TestImageGenerationServiceFormatsMediaType(t *testing.T) {
	for _, format := range []struct {
		input string
		want  string
	}{
		{"", "image/png"},
		{"png", "image/png"},
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"webp", "image/webp"},
		{"custom", "image/custom"},
	} {
		provider := &imageProviderStub{
			generateFunc: func(ctx context.Context, req *imageclient.ProviderRequest) (*imageclient.ProviderResult, error) {
				return &imageclient.ProviderResult{
					Images:       []string{"aW1n"},
					OutputFormat: format.input,
				}, nil
			},
		}
		service := imageclient.NewImageGenerationService(provider)
		res, err := service.Generate(context.Background(), &imageclient.GenerateRequest{Prompt: "test"})
		if err != nil {
			t.Fatalf("format %q failed: %v", format.input, err)
		}
		if res.Images[0].MediaType != format.want {
			t.Fatalf("format %q: got %q, want %q", format.input, res.Images[0].MediaType, format.want)
		}
	}
}
