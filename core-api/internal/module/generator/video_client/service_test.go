package videoclient_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	videoclient "github.com/1024XEngineer/Holonic-Asset/internal/module/generator/video_client"
)

type videoProviderStub struct {
	request      *videoclient.ProviderRequest
	result       *videoclient.ProviderResult
	generateErr  error
	downloadURL  string
	downloadData []byte
	downloadErr  error
}

func (s *videoProviderStub) Generate(
	_ context.Context,
	request *videoclient.ProviderRequest,
) (*videoclient.ProviderResult, error) {
	copied := *request
	s.request = &copied
	return s.result, s.generateErr
}

func (s *videoProviderStub) Download(_ context.Context, videoURL string) ([]byte, error) {
	s.downloadURL = videoURL
	return append([]byte(nil), s.downloadData...), s.downloadErr
}

func TestVideoGenerationServiceBuildsReferenceDataURL(t *testing.T) {
	provider := &videoProviderStub{result: &videoclient.ProviderResult{
		RequestID: "request-1",
		VideoURL:  "https://cdn.example.test/video.mp4",
	}}
	service := videoclient.NewVideoGenerationService(provider)

	result, err := service.Generate(context.Background(), &videoclient.GenerateRequest{
		Prompt:                  "  fixed camera walk cycle  ",
		ReferenceImageBase64:    "cG5n",
		ReferenceImageMediaType: "image/webp",
		Resolution:              "1080p",
		Duration:                8,
		AspectRatio:             "16:9",
		GenerateAudio:           true,
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}

	wantRequest := &videoclient.ProviderRequest{
		Prompt:            "fixed camera walk cycle",
		ReferenceImageURL: "data:image/webp;base64,cG5n",
		Resolution:        "1080p",
		Duration:          8,
		AspectRatio:       "16:9",
		GenerateAudio:     true,
	}
	if !reflect.DeepEqual(provider.request, wantRequest) {
		t.Fatalf("unexpected provider request:\nwant: %+v\n got: %+v", wantRequest, provider.request)
	}
	if result.RequestID != "request-1" || result.VideoURL != "https://cdn.example.test/video.mp4" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVideoGenerationServiceUsesPNGMediaTypeAndDownloads(t *testing.T) {
	provider := &videoProviderStub{
		result:       &videoclient.ProviderResult{VideoURL: "https://cdn.example.test/video.mp4"},
		downloadData: []byte("mp4"),
	}
	service := videoclient.NewVideoGenerationService(provider)

	_, err := service.Generate(context.Background(), &videoclient.GenerateRequest{
		Prompt:               "idle animation",
		ReferenceImageBase64: "cG5n",
	})
	if err != nil {
		t.Fatalf("generate video: %v", err)
	}
	if provider.request.ReferenceImageURL != "data:image/png;base64,cG5n" {
		t.Fatalf("unexpected reference image URL: %q", provider.request.ReferenceImageURL)
	}

	video, err := service.Download(context.Background(), "https://cdn.example.test/video.mp4")
	if err != nil {
		t.Fatalf("download video: %v", err)
	}
	if string(video) != "mp4" || provider.downloadURL != "https://cdn.example.test/video.mp4" {
		t.Fatalf("unexpected download: %q from %q", video, provider.downloadURL)
	}
}

func TestVideoGenerationServiceRejectsInvalidRequestsAndResponses(t *testing.T) {
	service := videoclient.NewVideoGenerationService(&videoProviderStub{})

	for name, request := range map[string]*videoclient.GenerateRequest{
		"nil request":       nil,
		"missing prompt":    {ReferenceImageBase64: "cG5n"},
		"missing reference": {Prompt: "idle"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.Generate(context.Background(), request)
			assertProviderErrorKind(t, err, videoclient.ErrorKindInvalidRequest)
		})
	}

	provider := &videoProviderStub{result: &videoclient.ProviderResult{}}
	_, err := videoclient.NewVideoGenerationService(provider).Generate(
		context.Background(),
		&videoclient.GenerateRequest{Prompt: "idle", ReferenceImageBase64: "cG5n"},
	)
	assertProviderErrorKind(t, err, videoclient.ErrorKindInvalidResponse)
}

func assertProviderErrorKind(t *testing.T, err error, kind videoclient.ErrorKind) {
	t.Helper()
	var providerErr *videoclient.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Kind != kind {
		t.Fatalf("error kind = %q, want %q", providerErr.Kind, kind)
	}
}

var _ videoclient.VideoProvider = (*videoProviderStub)(nil)
