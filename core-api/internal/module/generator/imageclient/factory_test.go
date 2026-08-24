package imageclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestIsChatProtocolModelRoutesGoogleNamespace(t *testing.T) {
	if !imageclient.IsChatProtocolModel("google/imagen-4.0-generate-001") {
		t.Fatal("google/* model was not routed through Chat Completions")
	}
}
