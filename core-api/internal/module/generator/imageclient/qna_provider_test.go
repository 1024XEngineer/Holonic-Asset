package imageclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
)

func TestQNAProviderGenerateUsesConfiguredKeyModelAndEndpoint(t *testing.T) {
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

	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{
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

func TestQNAProviderEditSendsReferenceImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/images/edits" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		var payload struct {
			Prompt string   `json:"prompt"`
			Image  []string `json:"image"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		if payload.Prompt != "make it blue" || !reflect.DeepEqual(payload.Image, []string{"data:image/png;base64,ref"}) {
			t.Fatalf("unexpected edit payload: %+v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"b64_json":"edited-image"}]}`))
	}))
	defer server.Close()

	provider := imageclient.NewQNAProvider(imageclient.QNAConfig{BaseURL: server.URL, APIKey: "test-key"})

	result, err := provider.Edit(context.Background(), &imageclient.ProviderRequest{
		Prompt:          "make it blue",
		ReferenceImages: []string{"data:image/png;base64,ref"},
	})
	if err != nil {
		t.Fatalf("edit image: %v", err)
	}
	if !reflect.DeepEqual(result.Images, []string{"edited-image"}) {
		t.Fatalf("unexpected images: %+v", result.Images)
	}
}
