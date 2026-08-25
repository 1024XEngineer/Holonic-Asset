package videoclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func TestQNAProviderGeneratesPollsAndDownloadsVideo(t *testing.T) {
	var polls atomic.Int32
	logs := &recordingLogger{}
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
		Logger:       logs,
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
	inProgress, ok := logs.find("qna video task still in progress")
	if !ok {
		t.Fatal("expected in-progress poll log")
	}
	for key, want := range map[string]any{
		"stage":           "poll",
		"request_id":      "request-1",
		"status_code":     http.StatusBadRequest,
		"task_status":     "IN_PROGRESS",
		"detail_type":     "request_in_progress",
		"will_retry":      false,
		"will_poll_again": true,
	} {
		if got := inProgress.fields[key]; got != want {
			t.Errorf("in-progress log field %q = %#v, want %#v", key, got, want)
		}
	}
	if _, ok := logs.find("qna video API request failed"); ok {
		t.Fatal("in-progress poll must not be logged as an API failure")
	}

	video, err := provider.Download(context.Background(), result.VideoURL)
	if err != nil {
		t.Fatalf("download video: %v", err)
	}
	if string(video) != "mp4" {
		t.Fatalf("video = %q, want mp4", video)
	}
}

func TestQNAProviderRoutesConfiguredVideoModel(t *testing.T) {
	const model = "acme/video-v1"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/queue/acme/video-v1/image-to-video" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"request_id": "configured-video",
			"video":      map[string]string{"url": "https://cdn.example.test/configured.mp4"},
		})
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Models: []videoclient.ModelConfig{
			{Name: model, Protocol: "fal_queue"},
		},
		HTTPClient: server.Client(),
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		Model:         model,
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate configured video model: %v", err)
	}
	if result.RequestID != "configured-video" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestQNAProviderRejectsInvalidConfiguredVideoRoutes(t *testing.T) {
	tests := []struct {
		name      string
		config    videoclient.QNAConfig
		model     string
		wantError string
	}{
		{
			name: "missing request model",
			config: videoclient.QNAConfig{
				Models: []videoclient.ModelConfig{
					{Name: "configured-model", Protocol: "fal_queue"},
				},
			},
			wantError: "video model is required",
		},
		{
			name: "unmapped model",
			config: videoclient.QNAConfig{
				Models: []videoclient.ModelConfig{
					{Name: "configured-model", Protocol: "fal_queue"},
				},
			},
			model:     "unmapped-model",
			wantError: `no video protocol is configured for model "unmapped-model"`,
		},
		{
			name: "unsupported protocol",
			config: videoclient.QNAConfig{
				Models: []videoclient.ModelConfig{
					{Name: "model", Protocol: "openai_video"},
				},
			},
			model:     "model",
			wantError: `unsupported video protocol "openai_video" for model "model"`,
		},
		{
			name: "duplicate model",
			config: videoclient.QNAConfig{
				Models: []videoclient.ModelConfig{
					{Name: "model", Protocol: "fal_queue"},
					{Name: "model", Protocol: "fal_queue"},
				},
			},
			model:     "model",
			wantError: `model "model" is assigned to multiple video protocols`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.config.APIKey = "test-key"
			provider := videoclient.NewQNAProvider(test.config)
			_, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
				Prompt:        "fixed camera",
				Model:         test.model,
				StartImageURL: "data:image/png;base64,cG5n",
			})
			var providerErr *videoclient.ProviderError
			if !errors.As(err, &providerErr) || providerErr.Kind != videoclient.ErrorKindInvalidRequest {
				t.Fatalf("error = %v, want invalid-request ProviderError", err)
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantError)
			}
		})
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

func TestQNAProviderRetriesPollTimeoutAndLogsWillRetry(t *testing.T) {
	var polls atomic.Int32
	logs := &recordingLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			_ = json.NewEncoder(writer).Encode(map[string]string{"request_id": "request-timeout"})
		case http.MethodGet:
			if polls.Add(1) == 1 {
				<-request.Context().Done()
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"status": "COMPLETED",
				"video":  map[string]string{"url": "https://cdn.example.test/video.mp4"},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		PollInterval: time.Millisecond,
		PollTimeout:  20 * time.Millisecond,
		MaxRetries:   1,
		RetryDelay:   time.Millisecond,
		HTTPClient:   server.Client(),
		Logger:       logs,
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if polls.Load() != 2 || result.RequestID != "request-timeout" {
		t.Fatalf("polls=%d result=%+v", polls.Load(), result)
	}

	entry, ok := logs.find("qna video API transport failure")
	if !ok {
		t.Fatal("expected poll timeout log")
	}
	for key, want := range map[string]any{
		"stage":        "poll",
		"request_id":   "request-timeout",
		"attempt":      1,
		"max_attempts": 2,
		"error_kind":   string(videoclient.ErrorKindTimeout),
		"transient":    true,
		"will_retry":   true,
	} {
		if got := entry.fields[key]; got != want {
			t.Errorf("timeout log field %q = %#v, want %#v", key, got, want)
		}
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

func TestQNAProviderUsesDefaultRetriesAndLogsRequestStage(t *testing.T) {
	var creates atomic.Int32
	logs := &recordingLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != videoclient.DefaultQNACreatePath {
			http.NotFound(writer, request)
			return
		}
		if creates.Add(1) < 3 {
			writer.Header().Set("Server", "test-edge")
			writer.Header().Set("CF-Ray", "test-ray")
			writer.Header().Set("Request-Id", "gateway-request-123")
			writer.WriteHeader(522)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"request_id": "request-default-retry",
			"video":      map[string]string{"url": "https://cdn.example.test/video.mp4"},
		})
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		RetryDelay: time.Millisecond,
		HTTPClient: server.Client(),
		Logger:     logs,
	})
	result, err := provider.Generate(context.Background(), &videoclient.ProviderRequest{
		Prompt:        "fixed camera",
		StartImageURL: "data:image/png;base64,cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if creates.Load() != 3 {
		t.Fatalf("create calls = %d, want 3", creates.Load())
	}
	if result.RequestID != "request-default-retry" {
		t.Fatalf("unexpected result: %+v", result)
	}

	entry, ok := logs.find("qna video API request failed")
	if !ok {
		t.Fatal("expected staged QNA failure log")
	}
	for key, want := range map[string]any{
		"provider":            "qna",
		"stage":               "create",
		"method":              http.MethodPost,
		"endpoint":            videoclient.DefaultQNACreatePath,
		"attempt":             1,
		"max_attempts":        4,
		"status_code":         522,
		"upstream_server":     "test-edge",
		"upstream_request_id": "gateway-request-123",
		"cf_ray":              "test-ray",
		"will_retry":          true,
		"error_kind":          string(videoclient.ErrorKindUnavailable),
		"transient":           true,
	} {
		if got := entry.fields[key]; got != want {
			t.Errorf("log field %q = %#v, want %#v", key, got, want)
		}
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
	logs := &recordingLogger{}
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
		Logger:       logs,
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
	entry, ok := logs.find("qna video task failed")
	if !ok {
		t.Fatal("expected QNA task failure log")
	}
	for key, want := range map[string]any{
		"provider":    "qna",
		"stage":       "poll",
		"request_id":  "request-failed",
		"task_status": "FAILED",
		"error_kind":  string(videoclient.ErrorKindTaskFailed),
	} {
		if got := entry.fields[key]; got != want {
			t.Errorf("log field %q = %#v, want %#v", key, got, want)
		}
	}
}

func TestQNAProviderRedactsSignedURLFromTransportFailureLog(t *testing.T) {
	logs := &recordingLogger{}
	client := &http.Client{Transport: qnaRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("dial failed for %s", request.URL.String())
	})}
	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		HTTPClient: client,
		MaxRetries: -1,
		Logger:     logs,
	})
	_, err := provider.Download(
		context.Background(),
		"https://cdn.example.test/video.mp4?token=super-secret&e=1787624733",
	)
	if err == nil {
		t.Fatal("expected transport failure")
	}

	entry, ok := logs.find("qna video download transport failure")
	if !ok {
		t.Fatal("expected transport failure log")
	}
	loggedErr, ok := entry.fields["errorx"].(error)
	if !ok {
		t.Fatalf("errorx type = %T, want error", entry.fields["errorx"])
	}
	if strings.Contains(loggedErr.Error(), "super-secret") || strings.Contains(loggedErr.Error(), "1787624733") {
		t.Fatalf("signed URL query leaked into log error: %q", loggedErr)
	}
	if !strings.Contains(loggedErr.Error(), "https://cdn.example.test/video.mp4") {
		t.Fatalf("sanitized error lost endpoint context: %q", loggedErr)
	}
}

func TestQNAProviderLogsTLSDownloadNetworkStages(t *testing.T) {
	logs := &recordingLogger{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("mp4"))
	}))
	defer server.Close()

	provider := videoclient.NewQNAProvider(videoclient.QNAConfig{
		HTTPClient: server.Client(),
		MaxRetries: -1,
		Logger:     logs,
	})
	video, err := provider.Download(context.Background(), server.URL+"/video.mp4?token=secret")
	if err != nil {
		t.Fatalf("download video: %v", err)
	}
	if string(video) != "mp4" {
		t.Fatalf("video = %q, want mp4", video)
	}

	entry, ok := logs.find("qna video download completed")
	if !ok {
		t.Fatal("expected successful download log")
	}
	for key, want := range map[string]any{
		"stage":             "download",
		"endpoint":          "/video.mp4",
		"status_code":       http.StatusOK,
		"http_protocol":     "HTTP/1.1",
		"connection_reused": false,
	} {
		if got := entry.fields[key]; got != want {
			t.Errorf("network log field %q = %#v, want %#v", key, got, want)
		}
	}
	if remote, _ := entry.fields["remote_addr"].(string); strings.TrimSpace(remote) == "" {
		t.Errorf("remote_addr = %#v, want a resolved peer", entry.fields["remote_addr"])
	}
	if version, _ := entry.fields["tls_version"].(string); strings.TrimSpace(version) == "" {
		t.Errorf("tls_version = %#v, want TLS version", entry.fields["tls_version"])
	}
}

type qnaRoundTripFunc func(*http.Request) (*http.Response, error)

func (function qnaRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type recordedLogEntry struct {
	message string
	fields  map[string]any
}

type recordingLogger struct {
	mu      sync.Mutex
	entries []recordedLogEntry
}

func (l *recordingLogger) Debug(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *recordingLogger) Info(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *recordingLogger) Warn(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (l *recordingLogger) Error(message string, fields ...logger.Field) {
	l.record(message, fields)
}

func (*recordingLogger) Sync() error { return nil }

func (l *recordingLogger) record(message string, fields []logger.Field) {
	values := make(map[string]any, len(fields))
	for _, field := range fields {
		values[field.Key] = field.Val
	}
	l.mu.Lock()
	l.entries = append(l.entries, recordedLogEntry{message: message, fields: values})
	l.mu.Unlock()
}

func (l *recordingLogger) find(message string) (recordedLogEntry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.entries {
		if entry.message == message {
			return entry, true
		}
	}
	return recordedLogEntry{}, false
}
