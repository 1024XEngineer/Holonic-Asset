package imageclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

func TestFactoryChatCompletionsProtocolNames(t *testing.T) {
	const imageData = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	for name, protocol := range map[string]string{
		"new protocol name":    "chat_completions",
		"legacy protocol name": "gemini_chat",
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" {
					t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + imageData + `"}}]}`))
			}))
			defer server.Close()

			provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
				BaseURL:      server.URL,
				DefaultModel: "arbitrary-chat-image-model",
				Provider:     protocol,
			})

			result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Images) != 1 {
				t.Fatalf("expected 1 image, got %d", len(result.Images))
			}
		})
	}
}

func TestFactoryNewImageProviderAutoSelection(t *testing.T) {
	t.Run("selects QNA Chat Completions for google models", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" {
				t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}}]}`))
		}))
		defer server.Close()

		provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
			BaseURL:      server.URL,
			DefaultModel: "google/nano-banana-2",
		})

		result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(result.Images))
		}
	})

	t.Run("selects QNA for gpt-image-2 model", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/images/generations" {
				t.Fatalf("expected /v1/images/generations, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}]}`))
		}))
		defer server.Close()

		provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
			BaseURL:      server.URL,
			DefaultModel: "openai/gpt-image-2",
		})

		result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(result.Images))
		}
	})

	t.Run("wires failover when fallbackModel is set", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Path == "/v1/images/generations" {
				// Primary gpt-image-2 fails with 503
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":{"message":"overloaded"}}`))
				return
			}
			if r.URL.Path == "/v1/chat/completions" {
				// Fallback nano-banana-2 succeeds
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}}]}`))
				return
			}
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}))
		defer server.Close()

		provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
			BaseURL:       server.URL,
			DefaultModel:  "openai/gpt-image-2",
			FallbackModel: "google/nano-banana-2",
			Models: []imageclient.ModelConfig{
				{
					Name:     "openai/gpt-image-2",
					Protocol: "openai_images",
				},
				{
					Name:     "google/nano-banana-2",
					Protocol: "chat_completions",
				},
			},
		})

		result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(result.Images))
		}
		if calls != 2 {
			t.Fatalf("expected 2 calls (primary fail + fallback success), got %d", calls)
		}
	})
}

func TestFactoryConfiguredModelsRouteEachRequestModel(t *testing.T) {
	type observedRequest struct {
		path  string
		model string
	}
	observed := make(chan observedRequest, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		observed <- observedRequest{path: r.URL.Path, model: payload.Model}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/chat/completions" {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL:      server.URL,
		DefaultModel: "openai/gpt-image-2",
		Models: []imageclient.ModelConfig{
			{
				Name:     "openai/gpt-image-2",
				Protocol: "openai_images",
			},
			{
				Name:     "google/gemini-3.1-pro-image",
				Protocol: "chat_completions",
			},
			{
				Name:     "google/gemini-3.1-flash-lite-image",
				Protocol: "chat_completions",
			},
		},
	})

	if _, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "default"}); err != nil {
		t.Fatalf("generate with default model: %v", err)
	}
	if _, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "override",
		Model:  "google/gemini-3.1-flash-lite-image",
	}); err != nil {
		t.Fatalf("generate with request model: %v", err)
	}

	first := <-observed
	second := <-observed
	if first.path != "/v1/images/generations" || first.model != "openai/gpt-image-2" {
		t.Fatalf("unexpected default route: %+v", first)
	}
	if second.path != "/v1/chat/completions" || second.model != "google/gemini-3.1-flash-lite-image" {
		t.Fatalf("unexpected request-model route: %+v", second)
	}
}

func TestFactoryConfiguredProviderOverridesModelNameHeuristics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected configured chat protocol, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL:      server.URL,
		DefaultModel: "openai/custom-chat-image",
		Models: []imageclient.ModelConfig{
			{
				Name:     "openai/custom-chat-image",
				Protocol: "chat_completions",
			},
		},
	})

	if _, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"}); err != nil {
		t.Fatalf("generate through configured protocol: %v", err)
	}
}

func TestFactoryConfiguredModelsRejectUnmappedModel(t *testing.T) {
	provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		DefaultModel: "openai/gpt-image-2",
		Models: []imageclient.ModelConfig{
			{
				Name:     "openai/gpt-image-2",
				Protocol: "openai_images",
			},
		},
	})

	_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "test",
		Model:  "google/unconfigured-image-model",
	})
	if err == nil || !imageclient.IsPermanent(err) {
		t.Fatalf("expected permanent routing error, got %v", err)
	}
	if !strings.Contains(err.Error(), `no image protocol is configured for model "google/unconfigured-image-model"`) {
		t.Fatalf("unexpected routing error: %v", err)
	}
}

func TestFactoryConfiguredModelsRouteEdit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			t.Errorf("expected images edit protocol, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewImageProvider(imageclient.FactoryConfig{
		BaseURL:       server.URL,
		DefaultModel:  "openai/gpt-image-2",
		FallbackModel: "openai/gpt-image-2",
		Models: []imageclient.ModelConfig{
			{
				Name:     "",
				Protocol: "openai_images",
			},
			{
				Name:     "openai/gpt-image-2",
				Protocol: "openai_images",
			},
		},
	})

	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "edit",
		ReferenceImages: []string{"data:image/png;base64,aW1hZ2U="},
	})
	if err != nil {
		t.Fatalf("edit through configured protocol: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected one edited image, got %d", len(result.Images))
	}
}

func TestFactoryConfiguredModelsRejectInvalidRoutingConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		config    imageclient.FactoryConfig
		request   *imageclient.ProviderRequest
		edit      bool
		wantError string
	}{
		{
			name: "nil request",
			config: imageclient.FactoryConfig{
				DefaultModel: "model",
				Models: []imageclient.ModelConfig{
					{Name: "model", Protocol: "openai_images"},
				},
			},
			wantError: "image request is required",
		},
		{
			name: "missing model",
			config: imageclient.FactoryConfig{
				Models: []imageclient.ModelConfig{
					{Name: "configured-model", Protocol: "openai_images"},
				},
			},
			request:   &imageclient.ProviderRequest{Prompt: "test"},
			wantError: "image model is required",
		},
		{
			name: "unsupported protocol",
			config: imageclient.FactoryConfig{
				DefaultModel: "model",
				Models: []imageclient.ModelConfig{
					{Name: "model", Protocol: "unknown_protocol"},
				},
			},
			request:   &imageclient.ProviderRequest{Prompt: "test"},
			wantError: `unsupported image protocol "unknown_protocol"`,
		},
		{
			name: "duplicate model mapping",
			config: imageclient.FactoryConfig{
				DefaultModel: "model",
				Models: []imageclient.ModelConfig{
					{Name: "model", Protocol: "openai_images"},
					{Name: "model", Protocol: "chat_completions"},
				},
			},
			request:   &imageclient.ProviderRequest{Prompt: "test"},
			edit:      true,
			wantError: `model "model" is assigned to multiple image protocols`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := imageclient.NewImageProvider(test.config)
			var err error
			if test.edit {
				_, err = provider.Edit(context.Background(), test.request)
			} else {
				_, err = provider.Generate(context.Background(), test.request)
			}
			if err == nil || !imageclient.IsPermanent(err) {
				t.Fatalf("expected permanent routing error, got %v", err)
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantError)
			}
		})
	}
}

func TestIsChatProtocolModelRoutesGoogleNamespace(t *testing.T) {
	if !imageclient.IsChatProtocolModel("google/imagen-4.0-generate-001") {
		t.Fatal("google/* model was not routed through Chat Completions")
	}
}
