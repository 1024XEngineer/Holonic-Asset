package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
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
	// DefaultQNAChatCompletionsPath is the OpenAI-compatible chat endpoint.
	DefaultQNAChatCompletionsPath = "/v1/chat/completions"

	qnaProviderName              = "qna"
	qnaResponseFormatJSONSchema  = "json_schema"
	qnaResponseFormatJSONObject  = "json_object"
	qnaThinkingDisabled          = "disabled"
	invalidStructuredJSONMessage = "LLM response contains invalid structured JSON"
	qnaStructuredMaxTokens       = 8192
	defaultQNAHTTPTimeout        = 5 * time.Minute
)

// QNAConfig configures the QNA multimodal LLM provider.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
	SDKClient    *qnasdk.Client
	Logger       logger.Logger
}

// QNAProvider calls QNA's OpenAI-compatible chat completions API.
type QNAProvider struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
	sdkClient    *qnasdk.Client
	logger       logger.Logger
}

// NewQNAProvider creates a QNA provider.
func NewQNAProvider(config QNAConfig) *QNAProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultQNAHTTPTimeout}
	}
	sdkClient := config.SDKClient
	if sdkClient == nil {
		sdkClient = qnasdk.NewClient(baseURL, config.APIKey, httpClient)
	}
	provider := &QNAProvider{
		baseURL:      baseURL,
		apiKey:       strings.TrimSpace(config.APIKey),
		defaultModel: strings.TrimSpace(config.DefaultModel),
		httpClient:   httpClient,
		sdkClient:    sdkClient,
		logger:       config.Logger,
	}
	if provider.logger != nil {
		provider.logger.Debug("qna llm provider configured",
			logger.String("provider", qnaProviderName),
			logger.String("base_url", provider.baseURL),
			logger.String("default_model", provider.defaultModel),
			logger.Any("api_key_configured", provider.apiKey != ""),
		)
	}
	return provider
}

// Complete submits one multimodal structured completion.
func (p *QNAProvider) Complete(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	startedAt := time.Now()
	if request == nil {
		providerErr := newQNAError(ErrorKindInvalidRequest, 0, false, "LLM request is nil", nil)
		p.logRequestFailure("", 0, "", p.baseURL+DefaultQNAChatCompletionsPath, startedAt, 0, 0, providerErr)
		return nil, providerErr
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		providerErr := newQNAError(ErrorKindInvalidRequest, 0, false, "LLM model is required", nil)
		p.logRequestFailure(model, len(request.ImageURLs), request.ResponseSchema.Name, p.baseURL+DefaultQNAChatCompletionsPath, startedAt, 0, 0, providerErr)
		return nil, providerErr
	}

	result, err := p.completeWithResponseFormat(ctx, request, model, qnaResponseFormatJSONSchema)
	if !shouldFallbackToJSONObject(err) {
		return result, err
	}

	p.logJSONObjectFallback(model, request.ResponseSchema.Name, err, false)
	fallbackRequest := *request
	fallbackRequest.Prompt = qnaJSONObjectPrompt(request, false)
	result, err = p.completeWithResponseFormat(ctx, &fallbackRequest, model, qnaResponseFormatJSONObject)
	if !isInvalidStructuredJSONError(err) {
		return result, err
	}

	// Modelink recommends validating structured output and retrying failures in
	// production. Retry once without echoing the invalid model output back into
	// the prompt, which avoids amplifying malformed or user-derived content.
	p.logJSONObjectFallback(model, request.ResponseSchema.Name, err, true)
	retryRequest := *request
	retryRequest.Prompt = qnaJSONObjectPrompt(request, true)
	return p.completeWithResponseFormat(ctx, &retryRequest, model, qnaResponseFormatJSONObject)
}

func (p *QNAProvider) completeWithResponseFormat(
	ctx context.Context,
	request *ProviderRequest,
	model string,
	responseFormatType string,
) (*ProviderResult, error) {
	startedAt := time.Now()
	content := make([]qnaContentPart, 0, len(request.ImageURLs)+1)
	content = append(content, qnaContentPart{Type: "text", Text: request.Prompt})
	for _, imageURL := range request.ImageURLs {
		content = append(content, qnaContentPart{
			Type:     "image_url",
			ImageURL: &qnaImageURL{URL: imageURL},
		})
	}
	responseFormat := qnaResponseFormat{Type: responseFormatType}
	if responseFormatType == qnaResponseFormatJSONSchema {
		responseFormat.JSONSchema = &qnaJSONSchema{
			Name:   request.ResponseSchema.Name,
			Strict: true,
			Schema: request.ResponseSchema.Schema,
		}
	}
	payload := qnaChatRequest{
		Model:     model,
		MaxTokens: qnaStructuredMaxTokens,
		Messages: []qnaMessage{{
			Role:    "user",
			Content: content,
		}},
		ResponseFormat: responseFormat,
	}
	if isDeepSeekModel(model) {
		// Modelink documents DeepSeek reasoning control through the thinking
		// field. Structured generation does not need chain-of-thought, and
		// disabling it prevents long reasoning output from destabilizing JSON.
		payload.Thinking = &qnaThinking{Type: qnaThinkingDisabled}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		providerErr := newQNAError(ErrorKindInvalidRequest, 0, false, "encode LLM request", err)
		p.logRequestFailure(model, len(request.ImageURLs), request.ResponseSchema.Name, p.baseURL+DefaultQNAChatCompletionsPath, startedAt, 0, 0, providerErr, logger.String("response_format", responseFormatType))
		return nil, providerErr
	}

	var decoded qnaChatResponse
	metadata, err := p.sdkClient.ExecuteWithMetadata(
		ctx,
		http.MethodPost,
		"chat/completions",
		body,
		&decoded,
	)
	if err != nil {
		providerErr := classifyQNASDKError(ctx, err)
		p.logRequestFailure(model, len(request.ImageURLs), request.ResponseSchema.Name, p.baseURL+DefaultQNAChatCompletionsPath, startedAt, metadata.StatusCode, metadata.BodyBytes, providerErr, logger.String("response_format", responseFormatType))
		return nil, providerErr
	}
	if len(decoded.Choices) == 0 {
		providerErr := newQNAError(ErrorKindInvalidResponse, metadata.StatusCode, true, "LLM response contains no choices", nil)
		p.logRequestFailure(model, len(request.ImageURLs), request.ResponseSchema.Name, p.baseURL+DefaultQNAChatCompletionsPath, startedAt, metadata.StatusCode, metadata.BodyBytes, providerErr, logger.String("response_format", responseFormatType))
		return nil, providerErr
	}
	choice := decoded.Choices[0]
	structuredJSON, contentFormat, structuredErr := extractQNAStructuredJSON(choice.Message.Content)
	if structuredErr != nil {
		providerErr := newQNAError(ErrorKindInvalidResponse, metadata.StatusCode, true, invalidStructuredJSONMessage, structuredErr)
		p.logRequestFailure(
			model, len(request.ImageURLs), request.ResponseSchema.Name, p.baseURL+DefaultQNAChatCompletionsPath,
			startedAt, metadata.StatusCode, metadata.BodyBytes, providerErr,
			logger.String("response_format", responseFormatType),
			logger.String("finish_reason", choice.FinishReason),
			logger.Int("content_bytes", len(choice.Message.Content)),
			logger.Int("reasoning_content_bytes", len(choice.Message.ReasoningContent)),
			logger.String("content_format", contentFormat),
		)
		return nil, providerErr
	}
	responseModel := strings.TrimSpace(decoded.Model)
	if responseModel == "" {
		responseModel = model
	}
	p.logRequestSuccess(
		model,
		len(request.ImageURLs),
		request.ResponseSchema.Name,
		p.baseURL+DefaultQNAChatCompletionsPath,
		startedAt,
		metadata.StatusCode,
		metadata.BodyBytes,
		responseModel,
		decoded.ID,
		logger.String("response_format", responseFormatType),
		logger.String("finish_reason", choice.FinishReason),
		logger.Int("content_bytes", len(choice.Message.Content)),
		logger.Int("reasoning_content_bytes", len(choice.Message.ReasoningContent)),
		logger.String("content_format", contentFormat),
	)

	return &ProviderResult{
		ID:    decoded.ID,
		Model: responseModel,
		JSON:  append(json.RawMessage(nil), structuredJSON...),
		Usage: Usage{
			PromptTokens:     decoded.Usage.PromptTokens,
			CompletionTokens: decoded.Usage.CompletionTokens,
			TotalTokens:      decoded.Usage.TotalTokens,
		},
	}, nil
}

func (p *QNAProvider) logJSONObjectFallback(model string, schemaName string, err error, retry bool) {
	if p.logger == nil {
		return
	}
	message := "qna structured output falling back to json_object"
	if retry {
		message = "qna json_object output invalid; retrying once"
	}
	p.logger.Warn(message,
		logger.String("provider", qnaProviderName),
		logger.String("model", model),
		logger.String("response_schema", schemaName),
		logger.String("requested_response_format", qnaResponseFormatJSONSchema),
		logger.String("fallback_response_format", qnaResponseFormatJSONObject),
		logger.Any("retry", retry),
		logger.Error(err),
	)
}

func qnaJSONObjectPrompt(request *ProviderRequest, retry bool) string {
	prefix := "The provider is using JSON object mode for this response."
	if retry {
		prefix = "A previous response was not valid structured JSON. Correct the format and try the task again."
	}
	return fmt.Sprintf(`%s

%s
Follow this JSON Schema exactly:
%s

Output requirements:
- Return exactly one valid JSON object matching the schema.
- Do not use Markdown code fences.
- Do not add explanations, analysis, or text before or after the object.
- Do not repeat the prompt or schema.
- Keep string values concise but complete.`,
		strings.TrimSpace(request.Prompt),
		prefix,
		strings.TrimSpace(string(request.ResponseSchema.Schema)),
	)
}

func isDeepSeekModel(model string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(model)), "deepseek")
}

func extractQNAStructuredJSON(content string) (json.RawMessage, string, error) {
	candidate := bytes.TrimSpace([]byte(content))
	if len(candidate) == 0 {
		return nil, "empty", errors.New("empty content")
	}

	contentFormat := "plain"
	if bytes.HasPrefix(candidate, []byte("```")) {
		unwrapped, ok := unwrapQNAJSONFence(candidate)
		if !ok {
			return nil, "invalid_markdown_fence", errors.New("content is not one complete JSON code fence")
		}
		candidate = unwrapped
		contentFormat = "markdown_json_fence"
	}

	if len(candidate) == 0 || !json.Valid(candidate) {
		return nil, contentFormat, errors.New("content is not one valid JSON value")
	}
	return append(json.RawMessage(nil), candidate...), contentFormat, nil
}

func unwrapQNAJSONFence(content []byte) ([]byte, bool) {
	text := string(bytes.TrimSpace(content))
	headerText, bodyAndFence, ok := strings.Cut(text, "\n")
	if !ok {
		return nil, false
	}
	header := strings.ToLower(strings.TrimSpace(headerText))
	if header != "```" && header != "```json" {
		return nil, false
	}
	closing := strings.LastIndex(bodyAndFence, "\n```")
	if closing < 0 || strings.TrimSpace(bodyAndFence[closing+1:]) != "```" {
		return nil, false
	}
	body := bytes.TrimSpace([]byte(bodyAndFence[:closing]))
	if bytes.Contains(body, []byte("```")) {
		return nil, false
	}
	return body, true
}

func shouldFallbackToJSONObject(err error) bool {
	if isInvalidStructuredJSONError(err) {
		return true
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != ErrorKindInvalidRequest || providerErr.StatusCode != http.StatusBadRequest {
		return false
	}
	message := strings.ToLower(providerErr.Message)
	return strings.Contains(message, "response_format") &&
		(strings.Contains(message, "unavailable") || strings.Contains(message, "unsupported") || strings.Contains(message, "not available"))
}

func isInvalidStructuredJSONError(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) &&
		providerErr.Kind == ErrorKindInvalidResponse &&
		providerErr.Message == invalidStructuredJSONMessage
}

func classifyQNASDKError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return classifyQNARequestError(ctx, ctxErr)
	}
	var sdkErr *qnasdk.Error
	if errors.As(err, &sdkErr) {
		kind, transient := classifyQNAStatus(sdkErr.StatusCode)
		return newQNAError(kind, sdkErr.StatusCode, transient, sdkErr.Message, err)
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "withbaseurl failed") ||
		strings.Contains(message, "unsupported protocol scheme") {
		return newQNAError(ErrorKindInvalidRequest, 0, false, "configure QNA SDK request", err)
	}
	if strings.Contains(message, "decode sdk response json") {
		return newQNAError(ErrorKindInvalidResponse, http.StatusOK, true, "decode LLM response", err)
	}
	return classifyQNARequestError(ctx, err)
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

type qnaChatRequest struct {
	Model          string            `json:"model"`
	Messages       []qnaMessage      `json:"messages"`
	ResponseFormat qnaResponseFormat `json:"response_format"`
	Thinking       *qnaThinking      `json:"thinking,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
}

type qnaThinking struct {
	Type string `json:"type"`
}

type qnaMessage struct {
	Role    string           `json:"role"`
	Content []qnaContentPart `json:"content"`
}

type qnaContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *qnaImageURL `json:"image_url,omitempty"`
}

type qnaImageURL struct {
	URL string `json:"url"`
}

type qnaResponseFormat struct {
	Type       string         `json:"type"`
	JSONSchema *qnaJSONSchema `json:"json_schema,omitempty"`
}

type qnaJSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type qnaChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

var _ LLMProvider = (*QNAProvider)(nil)
