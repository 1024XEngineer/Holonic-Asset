package imageclient

import (
	"bytes"
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

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

const (
	// DefaultGeminiChatModel is the default multimodal image generation model.
	DefaultGeminiChatModel = "google/nano-banana-2"

	geminiChatProviderName = "gemini_chat"
	chatCompletionsPath    = "/chat/completions"
	maxChatErrorBodyBytes  = 1 << 20
	defaultChatHTTPTimeout = 5 * time.Minute
)

var (
	markdownImageRegex = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
	httpURLRegex       = regexp.MustCompile(`https?://[^\s"'>)]+`)
)

// GeminiChatConfig configures the Gemini chat image provider.
type GeminiChatConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
	Logger       logger.Logger
}

// GeminiChatProvider calls Modelink/OpenAI-compatible Chat Completions API for image generation.
type GeminiChatProvider struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
	logger       logger.Logger
}

// NewGeminiChatProvider creates a provider targeting /v1/chat/completions.
func NewGeminiChatProvider(config GeminiChatConfig) *GeminiChatProvider {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}

	defaultModel := config.DefaultModel
	if defaultModel == "" {
		defaultModel = DefaultGeminiChatModel
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultChatHTTPTimeout}
	}

	return &GeminiChatProvider{
		baseURL:      baseURL,
		apiKey:       config.APIKey,
		defaultModel: defaultModel,
		httpClient:   httpClient,
		logger:       config.Logger,
	}
}

// Generate calls Chat Completions endpoint for text-to-image generation.
func (p *GeminiChatProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, request, nil)
}

// Edit calls Chat Completions endpoint with reference images for image-to-image or editing.
func (p *GeminiChatProvider) Edit(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	return p.call(ctx, request, request.ReferenceImages)
}

func (p *GeminiChatProvider) call(
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

	// Add optional mask image
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
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: contents,
			},
		},
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, newChatProviderError(
			ErrorKindInvalidRequest,
			0,
			false,
			"encode chat image request",
			err,
		)
	}

	endpoint := p.endpointURL()
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, newChatProviderError(
			ErrorKindInvalidRequest,
			0,
			false,
			"create chat image request",
			err,
		)
	}

	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, classifyChatRequestError(ctx, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, chatStatusError(response)
	}

	var decoded chatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, newChatProviderError(
			ErrorKindInvalidResponse,
			response.StatusCode,
			true,
			"decode chat completion response",
			err,
		)
	}

	if len(decoded.Choices) == 0 {
		return nil, newChatProviderError(
			ErrorKindInvalidResponse,
			response.StatusCode,
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
		return nil, newChatProviderError(
			ErrorKindInvalidResponse,
			response.StatusCode,
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

func (p *GeminiChatProvider) endpointURL() string {
	baseURL := strings.TrimRight(p.baseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + chatCompletionsPath
	}
	return baseURL + "/v1" + chatCompletionsPath
}

func (p *GeminiChatProvider) extractImages(ctx context.Context, choices []chatChoice) ([]string, error) {
	images := make([]string, 0, len(choices))
	for _, choice := range choices {
		// 1. Check choice.Message.Images (returned by Modelink/Gemini Chat image API)
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
			b64 := stripDataURLPrefix(trimmed)
			images = append(images, b64)
		}
	}
	return images, nil
}

func (p *GeminiChatProvider) resolveImageToB64(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "data:image/") {
		return stripDataURLPrefix(rawURL), nil
	}
	if isLikelyBase64(rawURL) {
		return rawURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", newChatProviderError(ErrorKindTransport, 0, true, "create image download request: "+err.Error(), err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", newChatProviderError(ErrorKindTransport, 0, true, "download generated image: "+err.Error(), err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", newChatProviderError(
			ErrorKindTransport,
			resp.StatusCode,
			true,
			fmt.Sprintf("download generated image failed with status %d", resp.StatusCode),
			nil,
		)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", newChatProviderError(ErrorKindTransport, 0, true, "read downloaded image data: "+err.Error(), err)
	}
	if len(data) == 0 {
		return "", newChatProviderError(ErrorKindInvalidResponse, 0, true, "downloaded image is empty", nil)
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

func stripDataURLPrefix(dataURL string) string {
	commaIndex := strings.Index(dataURL, ",")
	if commaIndex != -1 && strings.HasPrefix(dataURL, "data:") {
		return dataURL[commaIndex+1:]
	}
	return dataURL
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

func newChatProviderError(
	kind ErrorKind,
	statusCode int,
	transient bool,
	message string,
	cause error,
) *ProviderError {
	return &ProviderError{
		Provider:   geminiChatProviderName,
		Kind:       kind,
		StatusCode: statusCode,
		Transient:  transient,
		Message:    message,
		Cause:      cause,
	}
}

func classifyChatRequestError(ctx context.Context, err error) *ProviderError {
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.Canceled) {
			return newChatProviderError(
				ErrorKindCanceled,
				0,
				false,
				"chat image request canceled",
				ctxErr,
			)
		}
		return newChatProviderError(
			ErrorKindTimeout,
			0,
			true,
			"chat image request timed out",
			ctxErr,
		)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return newChatProviderError(
			ErrorKindTimeout,
			0,
			true,
			"chat image request timed out",
			err,
		)
	}
	return newChatProviderError(
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
	return newChatProviderError(kind, response.StatusCode, transient, message, readErr)
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
