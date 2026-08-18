package videoclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
)

func TestQNAProviderGeneratesPollsAndDownloadsVideo(t *testing.T) {
	var polls atomic.Int32
	var received struct {
		Prompt        string `json:"prompt"`
		ImageURL      string `json:"image_url"`
		Resolution    string `json:"resolution"`
		Duration      string `json:"duration"`
		AspectRatio   string `json:"aspect_ratio"`
		GenerateAudio bool   `json:"generate_audio"`
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == videoclient.DefaultQNACreatePath:
			if request.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("unexpected authorization header: %q", request.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
				t.Errorf("decode create request: %v", err)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"status":     "IN_QUEUE",
				"request_id": "request-1",
			})
		case request.Method == http.MethodGet && request.URL.Path == videoclient.DefaultQNAResultPath+"/request-1":
			if polls.Add(1) == 1 {
				writer.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(writer).Encode(map[string]any{
					"status": "IN_PROGRESS",
					"detail": map[string]string{"type": "request_in_progress"},
				})
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"status": "COMPLETED",
				"result": map[string]any{
					"video": map[string]string{"url": server.URL + "/video.mp4"},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/video.mp4":
			if authorization := request.Header.Get("Authorization"); authorization != "" {
				t.Errorf("download leaked provider authorization: %q", authorization)
			}
			_, _ = writer.Write([]byte("mp4"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		PollInterval: time.Millisecond,
		HTTPClient:   server.Client(),
	})
	longPrompt := strings.Repeat("角色和道具必须保持完整。", 400)
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        longPrompt,
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if result.RequestID != "request-1" || result.VideoURL != server.URL+"/video.mp4" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if received.ImageURL != "data:image/png;base64,cG5n" || received.Resolution != "720p" ||
		received.Duration != "5" || received.AspectRatio != "1:1" || received.GenerateAudio {
		t.Fatalf("unexpected create payload: %+v", received)
	}
	if characters := utf8.RuneCountInString(received.Prompt); characters > 2450 {
		t.Fatalf("provider received %d prompt characters, want at most 2450", characters)
	}
	if !utf8.ValidString(received.Prompt) {
		t.Fatal("provider prompt is not valid UTF-8")
	}

	video, err := provider.Download(context.Background(), result.VideoURL)
	if err != nil {
		t.Fatalf("download video: %v", err)
	}
	if string(video) != "mp4" {
		t.Fatalf("video = %q, want mp4", video)
	}
}

func TestQNAProviderSendsStartAndEndImages(t *testing.T) {
	var received struct {
		Prompt      string `json:"prompt"`
		ImageURL    string `json:"image_url"`
		EndImageURL string `json:"end_image_url"`
		Resolution  string `json:"resolution"`
		Duration    string `json:"duration"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != videoclient.DefaultQNACreatePath {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Errorf("decode create request: %v", err)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"request_id": "request-boundaries",
			"video":      map[string]string{"url": "https://cdn.example.test/boundaries.mp4"},
		})
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "interpolate from the start boundary to the end boundary",
		StartImageURL: "data:image/png;base64,c3RhcnQ=",
		EndImageURL:   "data:image/png;base64,ZW5k",
		Resolution:    "720p",
		Duration:      5,
	})
	if err != nil {
		t.Fatalf("generate boundary video: %v", err)
	}
	if result.RequestID != "request-boundaries" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if received.ImageURL != "data:image/png;base64,c3RhcnQ=" ||
		received.EndImageURL != "data:image/png;base64,ZW5k" {
		t.Fatalf("unexpected boundary images: %+v", received)
	}
}

func TestQNAProviderRetriesTransientCreateStatus(t *testing.T) {
	var creates atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != videoclient.DefaultQNACreatePath {
			http.NotFound(writer, request)
			return
		}
		if creates.Add(1) == 1 {
			writer.WriteHeader(525)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"request_id": "request-retry",
			"video":      map[string]string{"url": "https://cdn.example.test/video.mp4"},
		})
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		MaxRetries: 1,
		RetryDelay: time.Millisecond,
		HTTPClient: server.Client(),
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if creates.Load() != 2 {
		t.Fatalf("create calls = %d, want 2", creates.Load())
	}
	if result.RequestID != "request-retry" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestQNAProviderRetriesTransientPollAndDownloadStatus(t *testing.T) {
	var polls atomic.Int32
	var downloads atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == videoclient.DefaultQNACreatePath:
			_ = json.NewEncoder(writer).Encode(map[string]string{"request_id": "request-retry"})
		case request.Method == http.MethodGet && request.URL.Path == videoclient.DefaultQNAResultPath+"/request-retry":
			if polls.Add(1) == 1 {
				writer.WriteHeader(525)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"status": "COMPLETED",
				"result": map[string]any{
					"video": map[string]string{"url": server.URL + "/video.mp4"},
				},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/video.mp4":
			if downloads.Add(1) == 1 {
				writer.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = writer.Write([]byte("mp4"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		PollInterval: time.Millisecond,
		MaxRetries:   1,
		RetryDelay:   time.Millisecond,
		HTTPClient:   server.Client(),
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	video, err := provider.Download(context.Background(), result.VideoURL)
	if err != nil {
		t.Fatalf("download video: %v", err)
	}
	if polls.Load() != 2 || downloads.Load() != 2 || string(video) != "mp4" {
		t.Fatalf("polls=%d downloads=%d video=%q", polls.Load(), downloads.Load(), video)
	}
}

func TestQNAProviderReturnsStableTaskFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			_ = json.NewEncoder(writer).Encode(map[string]string{"request_id": "request-failed"})
		case http.MethodGet:
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"status": "FAILED",
				"detail": map[string]string{"msg": "content rejected"},
			})
		}
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		PollInterval: time.Millisecond,
		HTTPClient:   server.Client(),
	})
	_, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		StartImageURL: "data:image/png;base64,cG5n",
	})
	var providerErr *videoclient.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Kind != videoclient.ErrorKindTaskFailed || providerErr.Transient {
		t.Fatalf("unexpected provider error: %+v", providerErr)
	}
	if !strings.Contains(providerErr.Message, "content rejected") {
		t.Fatalf("unexpected error message: %q", providerErr.Message)
	}
}
