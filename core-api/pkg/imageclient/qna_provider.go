package imageclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultQNABaseURL is the production endpoint documented by QNA.
	DefaultQNABaseURL = "https://api.qnaigc.com"
	// DefaultQNAModel is used when a request does not specify a model.
	DefaultQNAModel = "openai/gpt-image-2"

	qnaProviderName       = "qna"
	qnaGeneratePath       = "/v1/images/generations"
	qnaEditPath           = "/v1/images/edits"
	maxErrorBodyBytes     = 1 << 20
	defaultQNAHTTPTimeout = 5 * time.Minute
)

// QNAConfig configures the QNA image provider.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
}

// QNAProvider calls QNA's text-to-image and image-to-image APIs.
type QNAProvider struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
}

// NewQNAProvider creates a QNA provider with documented production defaults.
func NewQNAProvider(config QNAConfig) *QNAProvider {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = DefaultQNAModel
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultQNAHTTPTimeout}
	}

	return &QNAProvider{
		baseURL:      baseURL,
		apiKey:       config.APIKey,
		defaultModel: defaultModel,
		httpClient:   httpClient,
	}
}

// Generate calls QNA's text-to-image endpoint.
func (p *QNAProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, qnaGeneratePath, request, nil)
}

// Edit calls QNA's image-to-image endpoint.
func (p *QNAProvider) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, qnaEditPath, request, request.ReferenceImages)
}

func (p *QNAProvider) call(
	ctx context.Context,
	path string,
	request *ProviderRequest,
	referenceImages []string,
) (*ProviderResult, error) {
	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	payload := qnaImageRequest{
		Model:   model,
		Prompt:  request.Prompt,
		Image:   referenceImages,
		Size:    request.Size,
		Quality: request.Params["quality"],
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, newQNAError(
			ErrorKindInvalidRequest,
			0,
			false,
			"encode image request",
			err,
		)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, newQNAError(
			ErrorKindInvalidRequest,
			0,
			false,
			"create image request",
			err,
		)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, classifyQNARequestError(ctx, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, qnaStatusError(response)
	}

	var decoded qnaImageResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, newQNAError(
			ErrorKindInvalidResponse,
			response.StatusCode,
			true,
			"decode image response",
			err,
		)
	}
	if len(decoded.Data) == 0 {
		return nil, newQNAError(
			ErrorKindInvalidResponse,
			response.StatusCode,
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
				response.StatusCode,
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
		Model:        model,
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

func classifyQNARequestError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return newQNAError(ErrorKindCanceled, 0, false, "request canceled", err)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return newQNAError(ErrorKindTimeout, 0, true, "request timed out", err)
	}
	return newQNAError(ErrorKindTransport, 0, true, "request failed", err)
}

func qnaStatusError(response *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
	message := qnaErrorMessage(body)
	if message == "" {
		message = response.Status
	}

	kind, transient := classifyQNAStatus(response.StatusCode)
	return newQNAError(kind, response.StatusCode, transient, message, readErr)
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

func qnaErrorMessage(body []byte) string {
	var envelope struct {
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Message != "" {
			return envelope.Message
		}
		if len(envelope.Error) > 0 {
			var nested struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(envelope.Error, &nested); err == nil && nested.Message != "" {
				return nested.Message
			}
			var text string
			if err := json.Unmarshal(envelope.Error, &text); err == nil && text != "" {
				return text
			}
		}
	}
	return strings.TrimSpace(string(body))
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

var _ ImageProvider = (*QNAProvider)(nil)
