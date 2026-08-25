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
		Models: []llmclient.ModelConfig{
			{Name: "vision-model", Protocol: "chat_completions"},
		},
		HTTPClient: server.Client(),
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

func TestQNAProviderRejectsInvalidConfiguredModelRoutes(t *testing.T) {
	tests := []struct {
		name      string
		config    llmclient.QNAConfig
		model     string
		wantError string
	}{
		{
			name: "unmapped model",
			config: llmclient.QNAConfig{
				DefaultModel: "unmapped-model",
				Models: []llmclient.ModelConfig{
					{Name: "configured-model", Protocol: "chat_completions"},
				},
			},
			wantError: `no LLM protocol is configured for model "unmapped-model"`,
		},
		{
			name: "unsupported protocol",
			config: llmclient.QNAConfig{
				DefaultModel: "model",
				Models: []llmclient.ModelConfig{
					{Name: "model", Protocol: "responses"},
				},
			},
			wantError: `unsupported LLM protocol "responses" for model "model"`,
		},
		{
			name: "duplicate model",
			config: llmclient.QNAConfig{
				DefaultModel: "model",
				Models: []llmclient.ModelConfig{
					{Name: "model", Protocol: "chat_completions"},
					{Name: "model", Protocol: "chat_completions"},
				},
			},
			wantError: `model "model" is assigned to multiple LLM protocols`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := llmclient.NewQNAProvider(test.config)
			request := validProviderRequest()
			request.Model = test.model
			_, err := provider.Complete(context.Background(), request)
			assertProviderErrorKind(t, err, llmclient.ErrorKindInvalidRequest)
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantError)
			}
		})
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

func TestQNAProviderFallsBackWhenJSONSchemaResponseFormatIsUnavailable(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		var payload struct {
			ResponseFormat struct {
				Type       string           `json:"type"`
				JSONSchema *json.RawMessage `json:"json_schema"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if requestCount == 1 {
			if payload.ResponseFormat.Type != "json_schema" || payload.ResponseFormat.JSONSchema == nil {
				t.Fatalf("first response format = %+v, want json_schema", payload.ResponseFormat)
			}
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":{"message":"The request is invalid: This response_format type is unavailable now."}}`))
			return
		}
		if payload.ResponseFormat.Type != "json_object" || payload.ResponseFormat.JSONSchema != nil {
			t.Fatalf("fallback response format = %+v, want json_object without json_schema", payload.ResponseFormat)
		}
		_, _ = writer.Write([]byte(`{"id":"fallback-1","choices":[{"message":{"content":"{\"layers\":[]}"}}]}`))
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      server.URL,
		DefaultModel: "deepseek/deepseek-v4-flash-20260731",
		HTTPClient:   server.Client(),
	})
	result, err := provider.Complete(context.Background(), validProviderRequest())
	if err != nil {
		t.Fatalf("complete with response format fallback: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if result.ID != "fallback-1" || string(result.JSON) != `{"layers":[]}` {
		t.Fatalf("unexpected fallback result: %+v", result)
	}
}

func TestQNAProviderDisablesDeepSeekThinkingForStructuredOutput(t *testing.T) {
	var payload struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Thinking  *struct {
			Type string `json:"type"`
		} `json:"thinking"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      server.URL,
		DefaultModel: "deepseek/deepseek-v4-flash-20260731",
		HTTPClient:   server.Client(),
	})
	if _, err := provider.Complete(context.Background(), validProviderRequest()); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if payload.Model != "deepseek/deepseek-v4-flash-20260731" {
		t.Fatalf("model = %q, want DeepSeek model", payload.Model)
	}
	if payload.MaxTokens != 8192 {
		t.Fatalf("max_tokens = %d, want 8192", payload.MaxTokens)
	}
	if payload.Thinking == nil || payload.Thinking.Type != "disabled" {
		t.Fatalf("thinking = %+v, want disabled", payload.Thinking)
	}
}

func TestQNAProviderAcceptsSingleJSONMarkdownFence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		response, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "```json\n{\"layers\":[]}\n```"}}}})
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      server.URL,
		DefaultModel: "model",
	})
	result, err := provider.Complete(context.Background(), validProviderRequest())
	if err != nil {
		t.Fatalf("complete fenced JSON: %v", err)
	}
	if string(result.JSON) != `{"layers":[]}` {
		t.Fatalf("JSON = %s, want {\"layers\":[]}", result.JSON)
	}
}

func TestQNAProviderRetriesInvalidJSONObjectResponse(t *testing.T) {
	requestCount := 0
	var prompts []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		var payload struct {
			Messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		prompts = append(prompts, payload.Messages[0].Content[0].Text)
		switch requestCount {
		case 1:
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":{"message":"response_format json_schema unavailable"}}`))
		case 2:
			if payload.ResponseFormat.Type != "json_object" {
				t.Fatalf("fallback format = %q, want json_object", payload.ResponseFormat.Type)
			}
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"not JSON"}}]}`))
		case 3:
			if payload.ResponseFormat.Type != "json_object" {
				t.Fatalf("retry format = %q, want json_object", payload.ResponseFormat.Type)
			}
			response, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"layers":[]}`}}}})
			_, _ = writer.Write(response)
		}
	}))
	defer server.Close()

	provider := llmclient.NewQNAProvider(llmclient.QNAConfig{
		BaseURL:      server.URL,
		DefaultModel: "model",
	})
	result, err := provider.Complete(context.Background(), validProviderRequest())
	if err != nil {
		t.Fatalf("complete after retry: %v", err)
	}
	if requestCount != 3 || string(result.JSON) != `{"layers":[]}` {
		t.Fatalf("requests = %d, result = %s", requestCount, result.JSON)
	}
	if len(prompts) != 3 || !strings.Contains(prompts[1], "Follow this JSON Schema exactly") || !strings.Contains(prompts[2], "previous response was not valid") {
		t.Fatalf("unexpected fallback prompts: %#v", prompts)
	}
}
