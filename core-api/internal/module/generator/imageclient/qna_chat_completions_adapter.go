package imageclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

const (
	// DefaultQNAChatCompletionsModel is used when no chat-completions model is configured.
	DefaultQNAChatCompletionsModel = "google/nano-banana-2"

	maxChatErrorBodyBytes  = 1 << 20
	maxGeneratedImageBytes = 32 << 20
	defaultChatHTTPTimeout = 5 * time.Minute
)

var (
	markdownImageRegex = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
	httpURLRegex       = regexp.MustCompile(`https?://[^\s"'>)]+`)
)

// QNAChatCompletionsAdapterConfig configures QNA's OpenAI-compatible Chat Completions adapter.
type QNAChatCompletionsAdapterConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
	SDKClient    *qnasdk.Client
	// DownloadHTTPClient overrides the secure client used for model-returned image URLs.
	// It is intended for tests and other trusted transports.
	DownloadHTTPClient *http.Client
	Logger             logger.Logger
}

// QNAChatCompletionsAdapter calls QNA's OpenAI-compatible Chat Completions endpoint.
type QNAChatCompletionsAdapter struct {
	baseURL            string
	apiKey             string
	defaultModel       string
	httpClient         *http.Client
	sdkClient          *qnasdk.Client
	downloadHTTPClient *http.Client
	logger             logger.Logger
}

// NewQNAChatCompletionsAdapter creates an adapter targeting /v1/chat/completions.
func NewQNAChatCompletionsAdapter(config QNAChatCompletionsAdapterConfig) *QNAChatCompletionsAdapter {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = DefaultQNAChatCompletionsModel
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultChatHTTPTimeout}
	}
	downloadHTTPClient := config.DownloadHTTPClient
	if downloadHTTPClient == nil {
		downloadHTTPClient = newGeneratedImageHTTPClient()
	}
	sdkClient := config.SDKClient
	if sdkClient == nil {
		sdkClient = qnasdk.NewClient(baseURL, config.APIKey, httpClient)
	}

	return &QNAChatCompletionsAdapter{
		baseURL:            baseURL,
		apiKey:             config.APIKey,
		defaultModel:       defaultModel,
		httpClient:         httpClient,
		sdkClient:          sdkClient,
		downloadHTTPClient: downloadHTTPClient,
		logger:             config.Logger,
	}
}

// Generate calls Chat Completions endpoint for text-to-image generation.
func (p *QNAChatCompletionsAdapter) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, request, nil)
}

// Edit calls Chat Completions endpoint with reference images for image-to-image or editing.
func (p *QNAChatCompletionsAdapter) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, request, request.ReferenceImages)
}

func (p *QNAChatCompletionsAdapter) call(
	ctx context.Context,
	request *ProviderRequest,
	referenceImages []string,
) (*ProviderResult, error) {
	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	contents := make([]chatContentPart, 0, len(referenceImages)+2)

	// Add reference images first
	for _, ref := range referenceImages {
		formatted := formatChatImageRef(ref)
		if formatted != "" {
			contents = append(contents, chatContentPart{
				Type: "image_url",
				ImageURL: &chatImageURL{
					URL: formatted,
				},
			})
		}
	}

	// Chat Completions has no native mask field, so pass a mask as another image input.
	if request.MaskImage != "" {
		formattedMask := formatChatImageRef(request.MaskImage)
		if formattedMask != "" {
			contents = append(contents, chatContentPart{
				Type: "image_url",
				ImageURL: &chatImageURL{
					URL: formattedMask,
				},
			})
		}
	}

	// Add prompt text
	contents = append(contents, chatContentPart{
		Type: "text",
		Text: request.Prompt,
	})

	payload := chatCompletionRequest{
		Model: model,
		N:     request.N,
		Seed:  request.Params["seed"],
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: contents,
			},
		},
		Stream: false,
	}

	var decoded chatCompletionResponse
	if err := p.sdkClient.Execute(ctx, http.MethodPost, "chat/completions", payload, &decoded); err != nil {
		return nil, classifyChatSDKError(ctx, err)
	}

	if len(decoded.Choices) == 0 {
		return nil, newChatProtocolError(
			ErrorKindInvalidResponse,
			http.StatusOK,
			true,
			"chat completion response contains no choices",
			nil,
		)
	}

	images, err := p.extractImages(ctx, decoded.Choices)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, newChatProtocolError(
			ErrorKindInvalidResponse,
			http.StatusOK,
			true,
			"chat completion response contains no image data in choices",
			nil,
		)
	}

	return &ProviderResult{
		Images:       images,
		OutputFormat: "png",
		Size:         request.Size,
		CreatedAt:    decoded.Created,
		Usage: Usage{
			TotalTokens:  decoded.Usage.TotalTokens,
			InputTokens:  decoded.Usage.PromptTokens,
			OutputTokens: decoded.Usage.CompletionTokens,
			RequestCount: 1,
		},
	}, nil
}

func (p *QNAChatCompletionsAdapter) extractImages(ctx context.Context, choices []chatChoice) ([]string, error) {
	images := make([]string, 0, len(choices))
	for _, choice := range choices {
		// QNA documents generated images in choice.Message.Images.
		for _, imgPart := range choice.Message.Images {
			if imgPart.ImageURL != nil && imgPart.ImageURL.URL != "" {
				b64, err := p.resolveImageToB64(ctx, imgPart.ImageURL.URL)
				if err != nil {
					return nil, err
				}
				images = append(images, b64)
			}
		}

		// 2. Check structured ContentParts
		for _, part := range choice.Message.ContentParts {
			if part.Type == "image_url" && part.ImageURL != nil && part.ImageURL.URL != "" {
				b64, err := p.resolveImageToB64(ctx, part.ImageURL.URL)
				if err != nil {
					return nil, err
				}
				images = append(images, b64)
			}
		}

		contentStr := choice.Message.contentString()
		if contentStr == "" {
			continue
		}

		// Look for markdown image links: ![...](url)
		mdMatches := markdownImageRegex.FindAllStringSubmatch(contentStr, -1)
		if len(mdMatches) > 0 {
			for _, match := range mdMatches {
				if len(match) > 1 && match[1] != "" {
					b64, err := p.resolveImageToB64(ctx, match[1])
					if err != nil {
						return nil, err
					}
					images = append(images, b64)
				}
			}
			continue
		}

		// Look for URLs: http(s)://
		urls := httpURLRegex.FindAllString(contentStr, -1)
		if len(urls) > 0 {
			for _, u := range urls {
				b64, err := p.resolveImageToB64(ctx, u)
				if err != nil {
					return nil, err
				}
				images = append(images, b64)
			}
			continue
		}

		// Check if it is a data URL or raw base64
		trimmed := strings.TrimSpace(contentStr)
		if strings.HasPrefix(trimmed, "data:image/") || isLikelyBase64(trimmed) {
			b64, err := p.resolveImageToB64(ctx, trimmed)
			if err != nil {
				return nil, err
			}
			images = append(images, b64)
		}
	}
	return images, nil
}

func (p *QNAChatCompletionsAdapter) resolveImageToB64(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "data:image/") {
		b64, err := parseImageDataURL(rawURL)
		if err != nil {
			return "", newChatProtocolError(
				ErrorKindInvalidResponse,
				0,
				false,
				"invalid generated image data URL",
				err,
			)
		}
		return b64, nil
	}
	if isLikelyBase64(rawURL) {
		return rawURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", newChatProtocolError(ErrorKindTransport, 0, true, "create image download request: "+err.Error(), err)
	}
	if err := validateGeneratedImageURL(req.URL); err != nil {
		return "", newChatProtocolError(ErrorKindInvalidResponse, 0, false, "reject generated image URL", err)
	}

	resp, err := p.downloadHTTPClient.Do(req)
	if err != nil {
		return "", newChatProtocolError(ErrorKindTransport, 0, true, "download generated image: "+err.Error(), err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", newChatProtocolError(
			ErrorKindTransport,
			resp.StatusCode,
			true,
			fmt.Sprintf("download generated image failed with status %d", resp.StatusCode),
			nil,
		)
	}
	if resp.ContentLength > maxGeneratedImageBytes {
		return "", newChatProtocolError(
			ErrorKindInvalidResponse,
			resp.StatusCode,
			false,
			fmt.Sprintf("generated image exceeds %d bytes", maxGeneratedImageBytes),
			nil,
		)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" &&
		!strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return "", newChatProtocolError(
			ErrorKindInvalidResponse,
			resp.StatusCode,
			false,
			"generated image response has non-image content type",
			nil,
		)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxGeneratedImageBytes+1))
	if err != nil {
		return "", newChatProtocolError(ErrorKindTransport, 0, true, "read downloaded image data: "+err.Error(), err)
	}
	if len(data) > maxGeneratedImageBytes {
		return "", newChatProtocolError(
			ErrorKindInvalidResponse,
			resp.StatusCode,
			false,
			fmt.Sprintf("generated image exceeds %d bytes", maxGeneratedImageBytes),
			nil,
		)
	}
	if len(data) == 0 {
		return "", newChatProtocolError(ErrorKindInvalidResponse, 0, true, "downloaded image is empty", nil)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func formatChatImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	// Fallback to base64 PNG data URI if raw base64 string was provided
	if isLikelyBase64(ref) {
		return "data:image/png;base64," + ref
	}
	return ref
}

func parseImageDataURL(dataURL string) (string, error) {
	commaIndex := strings.IndexByte(dataURL, ',')
	if commaIndex < 0 || !strings.HasPrefix(dataURL, "data:image/") ||
		!strings.HasSuffix(strings.ToLower(dataURL[:commaIndex]), ";base64") {
		return "", errors.New("image data URL must contain a base64 payload")
	}

	payload := dataURL[commaIndex+1:]
	if payload == "" {
		return "", errors.New("image data URL payload is empty")
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return "", fmt.Errorf("decode image data URL payload: %w", err)
	}
	return payload, nil
}

func isLikelyBase64(s string) bool {
	if len(s) < 32 {
		return false
	}
	if strings.Contains(s, " ") || strings.Contains(s, "\n") {
		s = strings.TrimSpace(s)
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func newChatProtocolError(
	kind ErrorKind,
	statusCode int,
	transient bool,
	message string,
	cause error,
) *ProviderError {
	return &ProviderError{
		Provider:   qnaProviderName,
		Kind:       kind,
		StatusCode: statusCode,
		Transient:  transient,
		Message:    message,
		Cause:      cause,
	}
}

var _ protocolAdapter = (*QNAChatCompletionsAdapter)(nil)

func classifyChatRequestError(ctx context.Context, err error) *ProviderError {
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.Canceled) {
			return newChatProtocolError(
				ErrorKindCanceled,
				0,
				false,
				"chat image request canceled",
				ctxErr,
			)
		}
		return newChatProtocolError(
			ErrorKindTimeout,
			0,
			true,
			"chat image request timed out",
			ctxErr,
		)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return newChatProtocolError(
			ErrorKindTimeout,
			0,
			true,
			"chat image request timed out",
			err,
		)
	}
	return newChatProtocolError(
		ErrorKindTransport,
		0,
		true,
		"execute chat image request",
		err,
	)
}

func classifyChatStatus(statusCode int) (ErrorKind, bool) {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrorKindInvalidRequest, false
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrorKindAuthentication, false
	case http.StatusTooManyRequests:
		return ErrorKindRateLimited, true
	case http.StatusRequestTimeout:
		return ErrorKindTimeout, true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ErrorKindUnavailable, true
	default:
		if statusCode >= 500 {
			return ErrorKindUnavailable, true
		}
		return ErrorKindInvalidResponse, false
	}
}

func chatStatusError(response *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxChatErrorBodyBytes))
	message := chatErrorMessage(body)
	if message == "" {
		message = response.Status
	}
	kind, transient := classifyChatStatus(response.StatusCode)
	return newChatProtocolError(kind, response.StatusCode, transient, message, readErr)
}

func chatErrorMessage(body []byte) string {
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

// Request and Response Types for Chat Completions API

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	N        int           `json:"n,omitempty"`
	Seed     string        `json:"seed,omitempty"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role         string            `json:"role"`
	Content      any               `json:"content"`
	Images       []chatContentPart `json:"images,omitempty"`
	ContentParts []chatContentPart `json:"-"`
}

func (m *chatMessage) contentString() string {
	if str, ok := m.Content.(string); ok {
		return str
	}
	return ""
}

func (m *chatMessage) UnmarshalJSON(data []byte) error {
	type rawChatMessage struct {
		Role    string            `json:"role"`
		Content json.RawMessage   `json:"content"`
		Images  []chatContentPart `json:"images,omitempty"`
	}
	var raw rawChatMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.Images = raw.Images

	if len(raw.Content) == 0 {
		m.Content = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(raw.Content, &str); err == nil {
		m.Content = str
		return nil
	}

	var parts []chatContentPart
	if err := json.Unmarshal(raw.Content, &parts); err == nil {
		m.Content = parts
		m.ContentParts = parts
		return nil
	}

	m.Content = string(raw.Content)
	return nil
}

type chatContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *chatImageURL `json:"image_url,omitempty"`
}

type chatImageURL struct {
	URL string `json:"url"`
}

type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	Index   int         `json:"index"`
	Message chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
