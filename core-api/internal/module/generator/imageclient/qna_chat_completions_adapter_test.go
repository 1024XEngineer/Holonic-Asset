package imageclient_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestQNAChatCompletionsAdapterGenerateSuccessMarkdownURL(t *testing.T) {
	imageBytes := []byte("fake-png-content-data")

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}

		var req struct {
			Model    string `json:"model"`
			N        int    `json:"n"`
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "google/nano-banana-2" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if req.N != 2 {
			t.Fatalf("candidate count = %d, want 2", req.N)
		}
		if len(req.Messages) != 1 || len(req.Messages[0].Content) != 1 || req.Messages[0].Content[0].Text != "a knight" {
			t.Fatalf("unexpected content: %+v", req.Messages)
		}

		resp := map[string]any{
			"id":      "chatcmpl-123",
			"created": 1700000000,
			"model":   "google/nano-banana-2",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Here is your generated image:\n\n![result](https://images.example/output.png)",
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
		BaseURL:      apiServer.URL,
		APIKey:       "test-key",
		DefaultModel: "google/nano-banana-2",
		DownloadHTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.String() != "https://images.example/output.png" {
				t.Fatalf("unexpected download URL: %s", request.URL)
			}
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{"image/png"}},
				Body:          io.NopCloser(bytes.NewReader(imageBytes)),
				ContentLength: int64(len(imageBytes)),
				Request:       request,
			}, nil
		})},
	})

	result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "a knight",
		Size:   "1024x1024",
		N:      2,
	})
	if err != nil {
		t.Fatalf("unexpected generate error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}
	expectedB64 := base64.StdEncoding.EncodeToString(imageBytes)
	if result.Images[0] != expectedB64 {
		t.Fatalf("image mismatch: got %s, want %s", result.Images[0], expectedB64)
	}
	if result.Usage.TotalTokens != 30 {
		t.Fatalf("usage mismatch: got %+v", result.Usage)
	}
}

func TestQNAChatCompletionsAdapterRejectsPrivateGeneratedImageURL(t *testing.T) {
	downloadCalled := false
	apiServer := newChatResponseServer(t, `{
		"choices":[{"message":{"content":"http://127.0.0.1/internal.png"}}]
	}`)
	defer apiServer.Close()

	provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
		BaseURL: apiServer.URL,
		DownloadHTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			downloadCalled = true
			return nil, errors.New("unexpected download")
		})},
	})

	_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
	var providerErr *imageclient.ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != imageclient.ErrorKindInvalidResponse ||
		providerErr.Transient {
		t.Fatalf("generate error = %v, want permanent invalid-response error", err)
	}
	if downloadCalled {
		t.Fatal("download client called for a private URL")
	}
}

func TestQNAChatCompletionsAdapterBoundsGeneratedImageDownloads(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		contentType   string
		contentLength int64
		body          string
		want          string
	}{
		{name: "status", statusCode: http.StatusBadGateway, contentType: "image/png", body: "bad", want: "status 502"},
		{name: "declared size", statusCode: http.StatusOK, contentType: "image/png", contentLength: 1 << 40, want: "exceeds"},
		{name: "content type", statusCode: http.StatusOK, contentType: "text/html", body: "not an image", want: "non-image"},
		{name: "empty", statusCode: http.StatusOK, contentType: "image/png", want: "empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apiServer := newChatResponseServer(t, `{
				"choices":[{"message":{"content":"https://images.example/output.png"}}]
			}`)
			defer apiServer.Close()

			provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
				BaseURL: apiServer.URL,
				DownloadHTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode:    test.statusCode,
						Status:        http.StatusText(test.statusCode),
						Header:        http.Header{"Content-Type": []string{test.contentType}},
						Body:          io.NopCloser(strings.NewReader(test.body)),
						ContentLength: test.contentLength,
						Request:       request,
					}, nil
				})},
			})

			_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("generate error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestQNAChatCompletionsAdapterParsesStructuredImageResponses(t *testing.T) {
	fakeB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	apiServer := newChatResponseServer(t, `{
		"choices":[
			{"message":{"images":[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+fakeB64+`"}}]}},
			{"message":{"content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+fakeB64+`"}}]}}
		]
	}`)
	defer apiServer.Close()

	provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{BaseURL: apiServer.URL})
	result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("generate structured images: %v", err)
	}
	if len(result.Images) != 2 || result.Images[0] != fakeB64 || result.Images[1] != fakeB64 {
		t.Fatalf("unexpected structured images: %+v", result.Images)
	}
}

func TestQNAChatCompletionsAdapterRejectsInvalidSuccessResponses(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{name: "malformed JSON", body: `{`, want: "decode chat completion"},
		{name: "no choices", body: `{}`, want: "no choices"},
		{name: "no image data", body: `{"choices":[{"message":{"content":"plain text"}}]}`, want: "no image data"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := newChatResponseServer(t, test.body)
			defer server.Close()
			provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{BaseURL: server.URL + "/v1"})
			_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("generate error = %v, want %q", err, test.want)
			}
		})
	}
}

func newChatResponseServer(t *testing.T, responseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(responseBody))
	}))
}

func TestQNAChatCompletionsAdapterEditMultiReferenceAndMask(t *testing.T) {
	fakeB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Type     string `json:"type"`
					Text     string `json:"text,omitempty"`
					ImageURL *struct {
						URL string `json:"url"`
					} `json:"image_url,omitempty"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(req.Messages))
		}
		parts := req.Messages[0].Content
		// 2 reference images + 1 mask image + 1 prompt text = 4 parts
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d: %+v", len(parts), parts)
		}
		if parts[0].ImageURL.URL != "https://example.com/ref1.png" {
			t.Errorf("part 0 mismatch: %s", parts[0].ImageURL.URL)
		}
		if parts[1].ImageURL.URL != "https://example.com/ref2.png" {
			t.Errorf("part 1 mismatch: %s", parts[1].ImageURL.URL)
		}
		if !strings.HasPrefix(parts[2].ImageURL.URL, "data:image/png;base64,") {
			t.Errorf("mask part mismatch: %s", parts[2].ImageURL.URL)
		}
		if parts[3].Text != "edit instructions" {
			t.Errorf("prompt part mismatch: %s", parts[3].Text)
		}

		resp := map[string]any{
			"id":      "chatcmpl-456",
			"created": 1700000001,
			"model":   "google/nano-banana-2",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "data:image/png;base64," + fakeB64,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
		BaseURL: apiServer.URL,
		APIKey:  "test-key",
	})

	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt: "edit instructions",
		ReferenceImages: []string{
			"https://example.com/ref1.png",
			"https://example.com/ref2.png",
		},
		MaskImage: fakeB64,
	})
	if err != nil {
		t.Fatalf("unexpected edit error: %v", err)
	}
	if len(result.Images) != 1 || result.Images[0] != fakeB64 {
		t.Fatalf("image mismatch: got %+v", result.Images)
	}
}

func TestQNAChatCompletionsAdapterStatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantKind   imageclient.ErrorKind
		wantRetry  bool
	}{
		{
			name:       "rate limit 429",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"message":"Rate limit reached"}}`,
			wantKind:   imageclient.ErrorKindRateLimited,
			wantRetry:  true,
		},
		{
			name:       "server error 503",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"error":{"message":"Model overloaded"}}`,
			wantKind:   imageclient.ErrorKindUnavailable,
			wantRetry:  true,
		},
		{
			name:       "bad request 400",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"Invalid prompt"}}`,
			wantKind:   imageclient.ErrorKindInvalidRequest,
			wantRetry:  false,
		},
		{
			name:       "unauthorized 401",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"Invalid key"}}`,
			wantKind:   imageclient.ErrorKindAuthentication,
			wantRetry:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
				BaseURL: server.URL,
				APIKey:  "key",
			})

			_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			var providerErr *imageclient.ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("expected ProviderError, got %T: %v", err, err)
			}
			if providerErr.Kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", providerErr.Kind, tt.wantKind)
			}
			if providerErr.Transient != tt.wantRetry {
				t.Errorf("transient = %v, want %v", providerErr.Transient, tt.wantRetry)
			}
		})
	}
}

func TestQNAChatCompletionsAdapterTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	provider := imageclient.NewQNAChatCompletionsAdapter(imageclient.QNAChatCompletionsAdapterConfig{
		BaseURL: server.URL,
		APIKey:  "key",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := provider.Generate(ctx, &imageclient.ProviderRequest{Prompt: "test"})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	var providerErr *imageclient.ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != imageclient.ErrorKindTimeout {
		t.Fatalf("expected timeout ProviderError, got %v", err)
	}
}
