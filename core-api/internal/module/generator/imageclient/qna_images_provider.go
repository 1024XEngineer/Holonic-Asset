package imageclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

const (
	// DefaultQNABaseURL is the production endpoint documented by QNA.
	DefaultQNABaseURL = "https://api.qnaigc.com"
	// DefaultQNAImagesModel is used when a request does not specify a model.
	DefaultQNAImagesModel = "openai/gpt-image-2"

	qnaProviderName       = "qna"
	qnaGeneratePath       = "/v1/images/generations"
	qnaEditPath           = "/v1/images/edits"
	defaultQNAHTTPTimeout = 5 * time.Minute
)

// QNAImagesConfig configures QNA's OpenAI-compatible Images adapter.
type QNAImagesConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
	SDKClient    *qnasdk.Client
	Logger       logger.Logger
}

// QNAImagesProvider calls QNA's /v1/images generation and edit endpoints.
type QNAImagesProvider struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
	sdkClient    *qnasdk.Client
	logger       logger.Logger
}

// NewQNAImagesProvider creates the QNA Images adapter with production defaults.
func NewQNAImagesProvider(config QNAImagesConfig) *QNAImagesProvider {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = DefaultQNAImagesModel
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultQNAHTTPTimeout}
	}
	sdkClient := config.SDKClient
	if sdkClient == nil {
		sdkClient = qnasdk.NewClient(baseURL, config.APIKey, httpClient)
	}

	return &QNAImagesProvider{
		baseURL:      baseURL,
		apiKey:       config.APIKey,
		defaultModel: defaultModel,
		httpClient:   httpClient,
		sdkClient:    sdkClient,
		logger:       config.Logger,
	}
}

// Generate calls QNA's text-to-image endpoint.
func (p *QNAImagesProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, qnaGeneratePath, request, nil)
}

// Edit calls QNA's image-to-image endpoint.
func (p *QNAImagesProvider) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	result, err := p.call(ctx, qnaEditPath, request, request.ReferenceImages)
	if !shouldRetryQNAEditWithoutMask(request, err) {
		return result, err
	}
	if p.logger != nil {
		p.logger.Warn(
			"qna rejected the documented edit mask format; retrying without native mask",
			logger.Error(err),
		)
	}
	fallback := *request
	fallback.MaskImage = ""
	return p.call(ctx, qnaEditPath, &fallback, fallback.ReferenceImages)
}

func shouldRetryQNAEditWithoutMask(request *ProviderRequest, err error) bool {
	if request == nil || strings.TrimSpace(request.MaskImage) == "" || err == nil {
		return false
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != ErrorKindInvalidRequest ||
		providerErr.StatusCode != http.StatusBadRequest {
		return false
	}
	message := strings.ToLower(providerErr.Message)
	return strings.Contains(message, "mask must be an object") ||
		strings.Contains(message, "unable to download content from the provided url")
}

func (p *QNAImagesProvider) call(
	ctx context.Context,
	path string,
	request *ProviderRequest,
	referenceImages []string,
) (*ProviderResult, error) {
	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	providerSize := normalizeQNAImageSize(request.Size)
	payload := qnaImageRequest{
		Model:   model,
		Prompt:  request.Prompt,
		Image:   referenceImages,
		Mask:    request.MaskImage,
		N:       request.N,
		Size:    providerSize,
		Quality: request.Params["quality"],
	}
	var decoded qnaImageResponse
	if err := p.sdkClient.Execute(ctx, http.MethodPost, strings.TrimPrefix(path, "/v1/"), payload, &decoded); err != nil {
		return nil, classifyQNAImageSDKError(ctx, err)
	}
	if len(decoded.Data) == 0 {
		return nil, newQNAError(
			ErrorKindInvalidResponse,
			http.StatusOK,
			true,
			"image response contains no data",
			nil,
		)
	}

	images := make([]string, 0, len(decoded.Data))
	for _, image := range decoded.Data {
		if image.Base64 == "" {
			return nil, newQNAError(
				ErrorKindInvalidResponse,
				http.StatusOK,
				true,
				"image response contains an empty b64_json field",
				nil,
			)
		}
		images = append(images, image.Base64)
	}

	return &ProviderResult{
		Images:       images,
		OutputFormat: decoded.OutputFormat,
		Size:         decoded.Size,
		CreatedAt:    decoded.Created,
		Usage: Usage{
			TotalTokens:       decoded.Usage.TotalTokens,
			InputTokens:       decoded.Usage.InputTokens,
			OutputTokens:      decoded.Usage.OutputTokens,
			TextToImageCount:  decoded.Usage.TextToImageCount,
			ImageToImageCount: decoded.Usage.ImageToImageCount,
			RequestCount:      decoded.Usage.RequestCount,
		},
	}, nil
}

const minimumQNAImagePixels = 655_360

func normalizeQNAImageSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return size
	}
	var width, height int
	if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil || width <= 0 || height <= 0 {
		return size
	}
	if int64(width)*int64(height) < minimumQNAImagePixels {
		return "1024x1024"
	}
	return size
}

func classifyQNARequestError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return newQNAError(ErrorKindCanceled, 0, false, "request canceled", err)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return newQNAError(ErrorKindTimeout, 0, true, "request timed out", err)
	}
	return newQNAError(ErrorKindTransport, 0, true, "request failed", err)
}

func classifyQNAStatus(statusCode int) (ErrorKind, bool) {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrorKindAuthentication, false
	case http.StatusRequestTimeout:
		return ErrorKindTimeout, true
	case http.StatusTooManyRequests:
		return ErrorKindRateLimited, true
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrorKindInvalidRequest, false
	default:
		if statusCode >= http.StatusInternalServerError {
			return ErrorKindUnavailable, true
		}
		return ErrorKindInvalidRequest, false
	}
}

func newQNAError(
	kind ErrorKind,
	statusCode int,
	transient bool,
	message string,
	cause error,
) *ProviderError {
	if cause != nil && message == "" {
		message = cause.Error()
	}
	return &ProviderError{
		Provider:   qnaProviderName,
		Kind:       kind,
		StatusCode: statusCode,
		Transient:  transient,
		Message:    message,
		Cause:      cause,
	}
}

type qnaImageRequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	Image   []string `json:"image,omitempty"`
	Mask    string   `json:"mask,omitempty"`
	N       int      `json:"n,omitempty"`
	Size    string   `json:"size,omitempty"`
	Quality string   `json:"quality,omitempty"`
}

type qnaImageResponse struct {
	Created      int64  `json:"created"`
	OutputFormat string `json:"output_format"`
	Size         string `json:"size"`
	Data         []struct {
		Base64 string `json:"b64_json"`
	} `json:"data"`
	Usage struct {
		TotalTokens       int `json:"total_tokens"`
		InputTokens       int `json:"input_tokens"`
		OutputTokens      int `json:"output_tokens"`
		TextToImageCount  int `json:"ti_quantity"`
		ImageToImageCount int `json:"ii_quantity"`
		RequestCount      int `json:"req_count"`
	} `json:"usage"`
}

var _ ImageProvider = (*QNAImagesProvider)(nil)
