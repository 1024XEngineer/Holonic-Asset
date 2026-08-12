package llmclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
)

func TestQNAProviderSendsMultimodalStructuredRequest(t *testing.T) {
	var received struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL *struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type       string `json:"type"`
			JSONSchema struct {
				Name   string          `json:"name"`
				Strict bool            `json:"strict"`
				Schema json.RawMessage `json:"schema"`
			} `json:"json_schema"`
		} `json:"response_format"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != llmclient.DefaultQNAChatCompletionsPath {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"id":"completion-1",
			"model":"vision-model",
			"choices":[{"message":{"content":"{\"layers\":[{\"id\":1}]}"}}],
			"usage":{"prompt_tokens":20,"completion_tokens":8,"total_tokens":28}
		}`))
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "vision-model",
		HTTPClient:   server.Client(),
	})
	result, err := provider.Complete(context.Background(), &llmclient.ProviderRequest{
		Prompt:    "arrange layers",
		ImageURLs: []string{"https://cdn.example.test/1.png", "data:image/png;base64,cG5n"},
		ResponseSchema: llmclient.JSONSchema{
			Name:   "scenery_layout",
			Schema: json.RawMessage(`{"type":"object","additionalProperties":false}`),
		},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if received.Model != "vision-model" || len(received.Messages) != 1 || received.Messages[0].Role != "user" {
		t.Fatalf("unexpected model or messages: %+v", received)
	}
	content := received.Messages[0].Content
	if len(content) != 3 || content[0].Type != "text" || content[0].Text != "arrange layers" {
		t.Fatalf("unexpected message content: %+v", content)
	}
	gotImages := []string{content[1].ImageURL.URL, content[2].ImageURL.URL}
	if !reflect.DeepEqual(gotImages, []string{"https://cdn.example.test/1.png", "data:image/png;base64,cG5n"}) {
		t.Fatalf("unexpected image inputs: %+v", gotImages)
	}
	if received.ResponseFormat.Type != "json_schema" || !received.ResponseFormat.JSONSchema.Strict || received.ResponseFormat.JSONSchema.Name != "scenery_layout" || string(received.ResponseFormat.JSONSchema.Schema) != `{"type":"object","additionalProperties":false}` {
		t.Fatalf("unexpected response format: %+v", received.ResponseFormat)
	}
	if result.ID != "completion-1" || result.Model != "vision-model" || string(result.JSON) != `{"layers":[{"id":1}]}` || result.Usage.TotalTokens != 28 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestQNAProviderRequestModelOverridesDefault(t *testing.T) {
	var model string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(request.Body).Decode(&payload)
		model = payload.Model
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{BaseURL: server.URL, DefaultModel: "default-model"})
	_, err := provider.Complete(context.Background(), &llmclient.ProviderRequest{
		Model:          "request-model",
		Prompt:         "layout",
		ImageURLs:      []string{"https://cdn.example.test/layer.png"},
		ResponseSchema: llmclient.JSONSchema{Name: "layout", Schema: json.RawMessage(`{"type":"object"}`)},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if model != "request-model" {
		t.Fatalf("model = %q, want request-model", model)
	}
}

func TestQNAProviderUsesRequestModelWhenResponseOmitsModel(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{}"}}]}`)),
			Header:     make(http.Header),
		}, nil
	})}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:    "https://llm.example.test",
		HTTPClient: client,
	})

	request := validProviderRequest()
	request.Model = "request-model"
	result, err := provider.Complete(context.Background(), request)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if result.Model != "request-model" {
		t.Fatalf("model = %q, want request-model", result.Model)
	}
}

func TestQNAProviderRejectsInvalidRequests(t *testing.T) {
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{})
	for name, request := range map[string]*llmclient.ProviderRequest{
		"nil request":   nil,
		"missing model": validProviderRequest(),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := provider.Complete(context.Background(), request)
			assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
		})
	}
}

func TestQNAProviderRejectsUnencodableSchema(t *testing.T) {
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{DefaultModel: "model"})
	request := validProviderRequest()
	request.ResponseSchema.Schema = json.RawMessage(`{"type":`)

	_, err := provider.Complete(context.Background(), request)
	assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
}

func TestQNAProviderRejectsInvalidBaseURL(t *testing.T) {
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      "://invalid",
		DefaultModel: "model",
	})

	_, err := provider.Complete(context.Background(), validProviderRequest())
	assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
}

func TestQNAProviderClassifiesHTTPStatusErrors(t *testing.T) {
	tests := []struct {
		status    int
		kind      llmclient.ErrorKind
		transient bool
	}{
		{http.StatusUnauthorized, llmclient.ErrorKindAuthentication, false},
		{http.StatusForbidden, llmclient.ErrorKindAuthentication, false},
		{http.StatusBadRequest, llmclient.ErrorKindInvalidRequest, false},
		{http.StatusUnprocessableEntity, llmclient.ErrorKindInvalidRequest, false},
		{http.StatusNotFound, llmclient.ErrorKindInvalidRequest, false},
		{http.StatusRequestTimeout, llmclient.ErrorKindTimeout, true},
		{http.StatusTooManyRequests, llmclient.ErrorKindRateLimited, true},
		{http.StatusServiceUnavailable, llmclient.ErrorKindUnavailable, true},
	}
	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(`{"error":{"message":"upstream failure"}}`))
			}))
			defer server.Close()

			provider := llmclient.NewQNAProvider(llmclient.QNAConfig{BaseURL: server.URL, DefaultModel: "model"})
			_, err := provider.Complete(context.Background(), validProviderRequest())
			var providerErr *llmclient.ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("error = %v, want ProviderError", err)
			}
			if providerErr.Kind != test.kind || providerErr.Transient != test.transient || providerErr.StatusCode != test.status || providerErr.Message != "upstream failure" {
				t.Fatalf("unexpected provider error: %+v", providerErr)
			}
		})
	}
}

func TestQNAProviderExtractsHTTPErrorMessages(t *testing.T) {
	tests := map[string]struct {
		body   string
		status string
		want   string
	}{
		"top-level message": {body: `{"message":"top level"}`, want: "top level"},
		"string error":      {body: `{"error":"plain failure"}`, want: "plain failure"},
		"plain body":        {body: "  gateway failure  ", want: "gateway failure"},
		"empty body":        {status: "418 Custom Status", want: "418 Custom Status"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status := test.status
			if status == "" {
				status = "418 I'm a teapot"
			}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusTeapot,
					Status:     status,
					Body:       io.NopCloser(strings.NewReader(test.body)),
					Header:     make(http.Header),
				}, nil
			})}
			provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
				BaseURL:      "https://llm.example.test",
				DefaultModel: "model",
				HTTPClient:   client,
			})

			_, err := provider.Complete(context.Background(), validProviderRequest())
			var providerErr *llmclient.ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("error = %v, want ProviderError", err)
			}
			if providerErr.Message != test.want {
				t.Fatalf("message = %q, want %q", providerErr.Message, test.want)
			}
		})
	}
}

func TestQNAProviderPreservesHTTPErrorBodyReadFailure(t *testing.T) {
	readErr := errors.New("read response body")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Body:       &errorReadCloser{err: readErr},
			Header:     make(http.Header),
		}, nil
	})}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      "https://llm.example.test",
		DefaultModel: "model",
		HTTPClient:   client,
	})

	_, err := provider.Complete(context.Background(), validProviderRequest())
	if !errors.Is(err, readErr) {
		t.Fatalf("error = %v, want body read error", err)
	}
}

func TestQNAProviderClassifiesCancellationAndTimeout(t *testing.T) {
	tests := []struct {
		name      string
		context   func() (context.Context, context.CancelFunc)
		kind      llmclient.ErrorKind
		transient bool
	}{
		{
			name: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			kind: llmclient.ErrorKindCanceled,
		},
		{
			name: "timeout",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Nanosecond)
			},
			kind:      llmclient.ErrorKindTimeout,
			transient: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := llmclient.NewQNAProvider(llmclient.QNAConfig{BaseURL: "http://127.0.0.1:1", DefaultModel: "model"})
			ctx, cancel := test.context()
			defer cancel()
			_, err := provider.Complete(ctx, validProviderRequest())
			var providerErr *llmclient.ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("error = %v, want ProviderError", err)
			}
			if providerErr.Kind != test.kind || providerErr.Transient != test.transient {
				t.Fatalf("unexpected provider error: %+v", providerErr)
			}
		})
	}
}

func TestQNAProviderClassifiesTransportErrors(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network unavailable")
	})}
	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      "https://llm.example.test",
		DefaultModel: "model",
		HTTPClient:   client,
	})
	_, err := provider.Complete(context.Background(), validProviderRequest())
	var providerErr *llmclient.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Kind != llmclient.ErrorKindTransport || !providerErr.Transient {
		t.Fatalf("unexpected provider error: %+v", providerErr)
	}
}

func TestQNAProviderRejectsInvalidResponses(t *testing.T) {
	for name, response := range map[string]string{
		"malformed response":  `{"choices":`,
		"missing choices":     `{"choices":[]}`,
		"empty content":       `{"choices":[{"message":{"content":""}}]}`,
		"unstructured output": `{"choices":[{"message":{"content":"not json"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(response))
			}))
			defer server.Close()
			provider := llmclient.NewQNAProvider(llmclient.QNAConfig{BaseURL: server.URL, DefaultModel: "model"})
			_, err := provider.Complete(context.Background(), validProviderRequest())
			assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidResponse)
		})
	}
}

func validProviderRequest() *llmclient.ProviderRequest {
	return &llmclient.ProviderRequest{
		Prompt:         "layout",
		ImageURLs:      []string{"https://cdn.example.test/layer.png"},
		ResponseSchema: llmclient.JSONSchema{Name: "layout", Schema: json.RawMessage(`{"type":"object"}`)},
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type errorReadCloser struct {
	err error
}

func (reader *errorReadCloser) Read([]byte) (int, error) {
	return 0, reader.err
}

func (*errorReadCloser) Close() error { return nil }
