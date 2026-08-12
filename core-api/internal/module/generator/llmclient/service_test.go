package llmclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

type llmProviderStub struct {
	request *llmclient.ProviderRequest
	result  *llmclient.ProviderResult
	err     error
}

func (s *llmProviderStub) Complete(
	_ context.Context,
	request *llmclient.ProviderRequest,
) (*llmclient.ProviderResult, error) {
	s.request = request
	return s.result, s.err
}

func TestLLMServiceNormalizesAndCopiesMultimodalRequest(t *testing.T) {
	provider := &llmProviderStub{result: &llmclient.ProviderResult{
		ID:    "completion-1",
		Model: "vision-model",
		JSON:  json.RawMessage(` {"layers":[]} `),
		Usage: llmclient.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14},
	}}
	service := llmclient.NewLLMService(provider)
	request := &llmclient.CompletionRequest{
		Prompt: "  arrange the scenery layers  ",
		Images: []llmclient.ImageInput{
			{URL: "  https://cdn.example.test/background.png  "},
			{URL: "data:image/png;base64,cG5n"},
		},
		Model: "  vision-model  ",
		ResponseSchema: llmclient.JSONSchema{
			Name:   "  scenery_layout  ",
			Schema: json.RawMessage(` {"type":"object"} `),
		},
	}

	result, err := service.Complete(context.Background(), request)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	wantRequest := &llmclient.ProviderRequest{
		Prompt:    "arrange the scenery layers",
		ImageURLs: []string{"https://cdn.example.test/background.png", "data:image/png;base64,cG5n"},
		Model:     "vision-model",
		ResponseSchema: llmclient.JSONSchema{
			Name:   "scenery_layout",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	}
	if !reflect.DeepEqual(provider.request, wantRequest) {
		t.Fatalf("unexpected provider request:\nwant: %+v\n got: %+v", wantRequest, provider.request)
	}
	if result.ID != "completion-1" || result.Model != "vision-model" || string(result.JSON) != `{"layers":[]}` || result.Usage.TotalTokens != 14 {
		t.Fatalf("unexpected result: %+v", result)
	}

	provider.request.ImageURLs[0] = "changed"
	provider.request.ResponseSchema.Schema[0] = '['
	result.JSON[0] = '['
	if request.Images[0].URL != "  https://cdn.example.test/background.png  " || string(request.ResponseSchema.Schema) != ` {"type":"object"} ` || string(provider.result.JSON) != ` {"layers":[]} ` {
		t.Fatal("service did not isolate caller and provider data")
	}
}

func TestLLMServiceRejectsInvalidRequests(t *testing.T) {
	valid := func() *llmclient.CompletionRequest {
		return &llmclient.CompletionRequest{
			Prompt: "layout",
			Images: []llmclient.ImageInput{{URL: "https://cdn.example.test/layer.png"}},
			ResponseSchema: llmclient.JSONSchema{
				Name:   "layout",
				Schema: json.RawMessage(`{"type":"object"}`),
			},
		}
	}

	tests := map[string]func() *llmclient.CompletionRequest{
		"nil request": func() *llmclient.CompletionRequest { return nil },
		"missing prompt": func() *llmclient.CompletionRequest {
			request := valid()
			request.Prompt = " "
			return request
		},
		"missing images": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images = nil
			return request
		},
		"invalid image URL": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images[0].URL = "file:///tmp/layer.png"
			return request
		},
		"invalid image data": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images[0].URL = "data:image/png;base64,not-base64"
			return request
		},
		"incomplete image data URI": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images[0].URL = "data:image/png;base64"
			return request
		},
		"empty image data": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images[0].URL = "data:image/png;base64,"
			return request
		},
		"malformed image URL": func() *llmclient.CompletionRequest {
			request := valid()
			request.Images[0].URL = "://missing-scheme"
			return request
		},
		"missing schema name": func() *llmclient.CompletionRequest {
			request := valid()
			request.ResponseSchema.Name = ""
			return request
		},
		"invalid schema": func() *llmclient.CompletionRequest {
			request := valid()
			request.ResponseSchema.Schema = json.RawMessage(`[]`)
			return request
		},
	}

	service := llmclient.NewLLMService(&llmProviderStub{})
	for name, buildRequest := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := service.Complete(context.Background(), buildRequest())
			assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
		})
	}
}

func TestLLMServiceRejectsMissingProvider(t *testing.T) {
	service := llmclient.NewLLMService(nil)
	_, err := service.Complete(context.Background(), validCompletionRequest())
	assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
}

func TestLLMServiceDescribesUnsupportedImageURL(t *testing.T) {
	request := validCompletionRequest()
	request.Images[0].URL = "file:///tmp/layer.png"
	service := llmclient.NewLLMService(&llmProviderStub{})

	_, err := service.Complete(context.Background(), request)
	var providerErr *llmclient.ProviderError
	if !errors.As(err, &providerErr) || providerErr.Cause == nil {
		t.Fatalf("error = %v, want ProviderError with URL cause", err)
	}
	if got := providerErr.Cause.Error(); got != "parse \"file:///tmp/layer.png\": image URL must use HTTP(S) or an image data URI" {
		t.Fatalf("cause = %q", got)
	}
}

func TestLLMServicePreservesProviderError(t *testing.T) {
	providerErr := &llmclient.ProviderError{
		Provider:  "qna",
		Kind:      llmclient.ErrorKindUnavailable,
		Transient: true,
		Message:   "temporarily unavailable",
	}
	service := llmclient.NewLLMService(&llmProviderStub{err: providerErr})

	_, err := service.Complete(context.Background(), validCompletionRequest())
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want original provider error", err)
	}
}

func TestLLMServiceRejectsInvalidProviderResponses(t *testing.T) {
	request := &llmclient.CompletionRequest{
		Prompt: "layout",
		Images: []llmclient.ImageInput{{URL: "https://cdn.example.test/layer.png"}},
		ResponseSchema: llmclient.JSONSchema{
			Name:   "layout",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	}
	for name, result := range map[string]*llmclient.ProviderResult{
		"nil result":   nil,
		"empty JSON":   {JSON: nil},
		"invalid JSON": {JSON: json.RawMessage(`{"layers":`)},
	} {
		t.Run(name, func(t *testing.T) {
			service := llmclient.NewLLMService(&llmProviderStub{result: result})
			_, err := service.Complete(context.Background(), request)
			assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidResponse)
		})
	}
}

func assertProviderErrorKind(t *testing.T, err error, kind llmclient.ErrorKind) {
	t.Helper()
	var providerErr *llmclient.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Kind != kind {
		t.Fatalf("error kind = %q, want %q", providerErr.Kind, kind)
	}
}

func validCompletionRequest() *llmclient.CompletionRequest {
	return &llmclient.CompletionRequest{
		Prompt: "layout",
		Images: []llmclient.ImageInput{{URL: "https://cdn.example.test/layer.png"}},
		ResponseSchema: llmclient.JSONSchema{
			Name:   "layout",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	}
}

var _ llmclient.LLMProvider = (*llmProviderStub)(nil)
