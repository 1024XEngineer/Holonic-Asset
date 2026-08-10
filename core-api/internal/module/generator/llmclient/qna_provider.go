package llmclient

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
	// DefaultQNAChatCompletionsPath is the OpenAI-compatible chat endpoint.
	DefaultQNAChatCompletionsPath = "/v1/chat/completions"

	qnaProviderName       = "qna"
	maxErrorBodyBytes     = 1 << 20
	defaultQNAHTTPTimeout = 5 * time.Minute
)

// QNAConfig configures the QNA multimodal LLM provider.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
	HTTPClient   *http.Client
}

// QNAProvider calls QNA's OpenAI-compatible chat completions API.
type QNAProvider struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
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
	return &QNAProvider{
		baseURL:      baseURL,
		apiKey:       strings.TrimSpace(config.APIKey),
		defaultModel: strings.TrimSpace(config.DefaultModel),
		httpClient:   httpClient,
	}
}

// Complete submits one multimodal structured completion.
func (p *QNAProvider) Complete(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	if request == nil {
		return nil, newQNAError(ErrorKindInvalidRequest, 0, false, "LLM request is nil", nil)
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return nil, newQNAError(ErrorKindInvalidRequest, 0, false, "LLM model is required", nil)
	}

	content := make([]qnaContentPart, 0, len(request.ImageURLs)+1)
	content = append(content, qnaContentPart{Type: "text", Text: request.Prompt})
	for _, imageURL := range request.ImageURLs {
		content = append(content, qnaContentPart{
			Type:     "image_url",
			ImageURL: &qnaImageURL{URL: imageURL},
		})
	}
	payload := qnaChatRequest{
		Model: model,
		Messages: []qnaMessage{{
			Role:    "user",
			Content: content,
		}},
		ResponseFormat: qnaResponseFormat{
			Type: "json_schema",
			JSONSchema: qnaJSONSchema{
				Name:   request.ResponseSchema.Name,
				Strict: true,
				Schema: request.ResponseSchema.Schema,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, newQNAError(ErrorKindInvalidRequest, 0, false, "encode LLM request", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+DefaultQNAChatCompletionsPath,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, newQNAError(ErrorKindInvalidRequest, 0, false, "create LLM request", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, classifyQNARequestError(ctx, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, qnaStatusError(response)
	}

	var decoded qnaChatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, newQNAError(ErrorKindInvalidResponse, response.StatusCode, true, "decode LLM response", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, newQNAError(ErrorKindInvalidResponse, response.StatusCode, true, "LLM response contains no choices", nil)
	}
	structuredJSON := bytes.TrimSpace([]byte(decoded.Choices[0].Message.Content))
	if len(structuredJSON) == 0 || !json.Valid(structuredJSON) {
		return nil, newQNAError(ErrorKindInvalidResponse, response.StatusCode, true, "LLM response contains invalid structured JSON", nil)
	}
	responseModel := strings.TrimSpace(decoded.Model)
	if responseModel == "" {
		responseModel = model
	}

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

type qnaChatRequest struct {
	Model          string            `json:"model"`
	Messages       []qnaMessage      `json:"messages"`
	ResponseFormat qnaResponseFormat `json:"response_format"`
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
	Type       string        `json:"type"`
	JSONSchema qnaJSONSchema `json:"json_schema"`
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
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

var _ LLMProvider = (*QNAProvider)(nil)
