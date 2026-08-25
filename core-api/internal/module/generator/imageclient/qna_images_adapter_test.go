package imageclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

type imageProviderLoggerStub struct {
	warnings []string
	fields   [][]logger.Field
}

func (*imageProviderLoggerStub) Debug(string, ...logger.Field) {}
func (*imageProviderLoggerStub) Info(string, ...logger.Field)  {}
func (s *imageProviderLoggerStub) Warn(message string, fields ...logger.Field) {
	s.warnings = append(s.warnings, message)
	s.fields = append(s.fields, fields)
}
func (*imageProviderLoggerStub) Error(string, ...logger.Field) {}
func (*imageProviderLoggerStub) Sync() error                   { return nil }

func TestQNAImagesAdapterGenerateUsesConfiguredKeyModelAndEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		var payload struct {
			Model   string   `json:"model"`
			Prompt  string   `json:"prompt"`
			Image   []string `json:"image"`
			Size    string   `json:"size"`
			Quality string   `json:"quality"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		if payload.Model != "configured-model" || payload.Prompt != "pixel sword" ||
			payload.Size != "1024x1024" || payload.Quality != "high" || len(payload.Image) != 0 {
			t.Fatalf("unexpected request payload: %+v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"created":123,
			"output_format":"webp",
			"size":"64x64",
			"data":[{"b64_json":"image-one"}],
			"usage":{"total_tokens":9,"input_tokens":3,"output_tokens":6,"ti_quantity":1,"req_count":1}
		}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "configured-model",
	})

	result, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "pixel sword",
		Size:   "64x64",
		Params: imageclient.Params{"quality": "high"},
	})
	if err != nil {
		t.Fatalf("generate image: %v", err)
	}

	want := &imageclient.ProviderResult{
		Images:       []string{"image-one"},
		OutputFormat: "webp",
		Size:         "64x64",
		CreatedAt:    123,
		Usage: imageclient.Usage{
			TotalTokens:      9,
			InputTokens:      3,
			OutputTokens:     6,
			TextToImageCount: 1,
			RequestCount:     1,
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("unexpected result:\nwant: %+v\n got: %+v", want, result)
	}
}

func TestQNAImagesAdapterPreservesValidCustomImageSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			Size string `json:"size"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		if payload.Size != "1504x1024" {
			t.Fatalf("custom size = %q, want 1504x1024", payload.Size)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"b64_json":"image"}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{BaseURL: server.URL, APIKey: "test-key"})
	if _, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{
		Prompt: "direction sheet",
		Size:   "1504x1024",
	}); err != nil {
		t.Fatalf("generate custom-size image: %v", err)
	}
}

func TestQNAImagesAdapterEditSendsReferenceImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/images/edits" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		var payload struct {
			Prompt string   `json:"prompt"`
			Image  []string `json:"image"`
			Mask   string   `json:"mask"`
			N      int      `json:"n"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		if payload.Prompt != "make it blue" ||
			!reflect.DeepEqual(payload.Image, []string{"data:image/png;base64,ref"}) ||
			payload.Mask != "data:image/png;base64,mask" || payload.N != 2 {
			t.Fatalf("unexpected edit payload: %+v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"b64_json":"edited-image"}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{BaseURL: server.URL, APIKey: "test-key"})

	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "make it blue",
		ReferenceImages: []string{"data:image/png;base64,ref"},
		MaskImage:       "data:image/png;base64,mask",
		N:               2,
	})
	if err != nil {
		t.Fatalf("edit image: %v", err)
	}
	if !reflect.DeepEqual(result.Images, []string{"edited-image"}) {
		t.Fatalf("unexpected images: %+v", result.Images)
	}
}

func TestQNAImagesAdapterEditRetriesWithoutMaskWhenProviderRejectsDocumentedFormat(t *testing.T) {
	requests := 0
	providerLogger := &imageProviderLoggerStub{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		var payload struct {
			Mask string `json:"mask"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if payload.Mask != "data:image/png;base64,mask" {
				t.Fatalf("first request omitted mask: %+v", payload)
			}
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":{"message":"mask must be an object: json: cannot unmarshal string into Go value"}}`))
			return
		}
		if payload.Mask != "" {
			t.Fatalf("fallback request retained mask: %+v", payload)
		}
		_, _ = writer.Write([]byte(`{"data":[{"b64_json":"fallback-image"}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{
		BaseURL: server.URL, APIKey: "test-key", Logger: providerLogger,
	})
	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "edit",
		ReferenceImages: []string{"data:image/png;base64,ref"},
		MaskImage:       "data:image/png;base64,mask",
	})
	if err != nil {
		t.Fatalf("fallback edit: %v", err)
	}
	if requests != 2 || !reflect.DeepEqual(result.Images, []string{"fallback-image"}) {
		t.Fatalf("unexpected fallback result: requests=%d result=%+v", requests, result)
	}
	if len(providerLogger.warnings) != 1 || len(providerLogger.fields) != 1 ||
		len(providerLogger.fields[0]) != 1 || providerLogger.fields[0][0].Key != "errorx" {
		t.Fatalf("unexpected fallback warning: %+v", providerLogger)
	}
}

func TestQNAImagesAdapterEditDoesNotDropMaskForOtherBadRequests(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests++
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":{"message":"invalid prompt"}}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{BaseURL: server.URL, APIKey: "test-key"})
	_, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "edit",
		ReferenceImages: []string{"data:image/png;base64,ref"},
		MaskImage:       "https://storage.test/mask.png",
	})
	if err == nil || requests != 1 {
		t.Fatalf("unexpected unrelated-error fallback: requests=%d err=%v", requests, err)
	}
}

func TestQNAImagesAdapterClassifiesStatusCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		body       string
		wantKind   imageclient.ErrorKind
		transient  bool
	}{
		{http.StatusUnauthorized, `{"message":"unauthorized"}`, imageclient.ErrorKindAuthentication, false},
		{http.StatusForbidden, `{"message":"forbidden"}`, imageclient.ErrorKindAuthentication, false},
		{http.StatusRequestTimeout, `{"message":"timed out"}`, imageclient.ErrorKindTimeout, true},
		{http.StatusTooManyRequests, `{"error":{"message":"rate limited"}}`, imageclient.ErrorKindRateLimited, true},
		{http.StatusBadRequest, `{"error":"bad request"}`, imageclient.ErrorKindInvalidRequest, false},
		{http.StatusUnprocessableEntity, `raw error string`, imageclient.ErrorKindInvalidRequest, false},
		{http.StatusInternalServerError, `{"message":"server error"}`, imageclient.ErrorKindUnavailable, true},
		{525, `SSL Handshake Failed`, imageclient.ErrorKindUnavailable, true},
	}

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(tt.statusCode)
			_, _ = writer.Write([]byte(tt.body))
		}))
		provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{BaseURL: server.URL, APIKey: "key"})
		_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
		server.Close()

		if err == nil {
			t.Fatalf("expected error for status %d, got nil", tt.statusCode)
		}
		if imageclient.IsTransient(err) != tt.transient {
			t.Fatalf("status %d: got transient=%v, want %v", tt.statusCode, imageclient.IsTransient(err), tt.transient)
		}
		if imageclient.IsPermanent(err) == tt.transient {
			t.Fatalf("status %d: got permanent=%v, want %v", tt.statusCode, imageclient.IsPermanent(err), !tt.transient)
		}
	}
}

func TestQNAImagesAdapterHandlesInvalidResponsePayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `invalid json`},
		{"empty data list", `{"created":123,"data":[]}`},
		{"empty b64 field", `{"created":123,"data":[{"b64_json":""}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider := imageclient.NewQNAImagesAdapter(imageclient.QNAImagesAdapterConfig{BaseURL: server.URL, APIKey: "key"})
			_, err := provider.Generate(context.Background(), &imageclient.ProviderRequest{Prompt: "test"})
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
			if !imageclient.IsTransient(err) {
				t.Fatalf("expected invalid response to be transient, got: %v", err)
			}
		})
	}
}

func TestQNAImagesAdapterErrorMethods(t *testing.T) {
	var nilErr *imageclient.ProviderError
	if nilErr.Error() != "" || nilErr.Unwrap() != nil {
		t.Fatalf("expected empty for nil error")
	}

	errWithKindOnly := &imageclient.ProviderError{Kind: imageclient.ErrorKindUnavailable}
	if errWithKindOnly.Error() != "image provider: unavailable" {
		t.Fatalf("unexpected error string: %s", errWithKindOnly.Error())
	}
}
