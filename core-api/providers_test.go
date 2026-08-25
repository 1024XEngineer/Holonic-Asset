package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/llmclient"
	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
)

func TestInitImageServiceRoutesConfiguredModelProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer model-key" {
			t.Errorf("authorization = %q, want Bearer model-key", request.Header.Get("Authorization"))
		}
		if request.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload.Model != "studio/custom-image" {
			t.Errorf("model = %q, want studio/custom-image", payload.Model)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}}]}`))
	}))
	defer server.Close()

	service := InitImageService(config.ImageClientConfig{
		DefaultModel: "studio/custom-image",
		Models: []config.ModelConfig{
			{
				Name:     "studio/custom-image",
				Protocol: "chat_completions",
				BaseURL:  server.URL,
				APIKey:   "model-key",
			},
		},
	}, nil)
	result, err := service.Generate(context.Background(), &imageclient.GenerateRequest{Prompt: "generate"})
	if err != nil {
		t.Fatalf("generate image: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("images = %d, want 1", len(result.Images))
	}
}

func TestInitLLMServiceUsesIndependentConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer llm-key" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "vision-model" {
			t.Fatalf("model = %q, want vision-model", payload.Model)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer server.Close()

	service := InitLLMService(config.LLMClientConfig{
		BaseURL:      server.URL,
		APIKey:       "llm-key",
		DefaultModel: "vision-model",
		Models: []config.ModelConfig{
			{Name: "vision-model", Protocol: "chat_completions"},
		},
	})
	_, err := service.Complete(context.Background(), &llmclient.CompletionRequest{
		Prompt: "analyze",
		Images: []llmclient.ImageInput{{URL: "https://cdn.example.test/layer.png"}},
		ResponseSchema: llmclient.JSONSchema{
			Name:   "layout",
			Schema: json.RawMessage(`{"type":"object"}`),
		},
	})
	if err != nil {
		t.Fatalf("complete through initialized service: %v", err)
	}
}

func TestInitVideoServiceRoutesConfiguredModelProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/queue/bytedance/seedance-2.0/image-to-video" {
			t.Errorf("path = %q, want Seedance create path", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"request_id":"video-1","video":{"url":"https://cdn.example.test/video.mp4"}}`))
	}))
	defer server.Close()

	service := InitVideoService(config.VideoClientConfig{
		BaseURL: server.URL,
		APIKey:  "video-key",
		Models: []config.ModelConfig{
			{Name: "bytedance/seedance-2.0", Protocol: "fal_queue"},
		},
	}, nil)
	result, err := service.Generate(context.Background(), &videoclient.GenerateRequest{
		Prompt:     "animate",
		Model:      "bytedance/seedance-2.0",
		StartImage: videoclient.ReferenceImage{Base64: "cG5n"},
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if result.RequestID != "video-1" || result.VideoURL != "https://cdn.example.test/video.mp4" {
		t.Fatalf("unexpected video result: %+v", result)
	}
}
