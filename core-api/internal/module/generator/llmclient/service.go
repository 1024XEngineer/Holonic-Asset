package llmclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

// LLMService completes provider-independent multimodal calls with structured
// JSON output. Business prompts and domain validation remain with callers.
type LLMService interface {
	Complete(context.Context, *CompletionRequest) (*CompletionResult, error)
}

type llmService struct {
	provider LLMProvider
}

// NewLLMService creates the call layer backed by provider.
func NewLLMService(provider LLMProvider) LLMService {
	return &llmService{provider: provider}
}

func (s *llmService) Complete(
	ctx context.Context,
	request *CompletionRequest,
) (*CompletionResult, error) {
	providerRequest, err := normalizeRequest(request)
	if err != nil {
		return nil, err
	}
	if s.provider == nil {
		return nil, invalidRequestError("LLM provider is required", nil)
	}

	providerResult, err := s.provider.Complete(ctx, providerRequest)
	if err != nil {
		return nil, err
	}
	if providerResult == nil {
		return nil, invalidResponseError("provider returned no completion", nil)
	}

	structuredJSON := bytes.TrimSpace(providerResult.JSON)
	if len(structuredJSON) == 0 {
		return nil, invalidResponseError("provider returned empty structured JSON", nil)
	}
	if !json.Valid(structuredJSON) {
		return nil, invalidResponseError("provider returned invalid structured JSON", nil)
	}

	return &CompletionResult{
		ID:    providerResult.ID,
		Model: providerResult.Model,
		JSON:  append(json.RawMessage(nil), structuredJSON...),
		Usage: providerResult.Usage,
	}, nil
}

func normalizeRequest(request *CompletionRequest) (*ProviderRequest, error) {
	if request == nil {
		return nil, invalidRequestError("LLM completion request is nil", nil)
	}
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return nil, invalidRequestError("LLM prompt is required", nil)
	}
	if len(request.Images) == 0 {
		return nil, invalidRequestError("at least one image is required", nil)
	}

	imageURLs := make([]string, len(request.Images))
	for index, image := range request.Images {
		imageURL := strings.TrimSpace(image.URL)
		if err := validateImageURL(imageURL); err != nil {
			return nil, invalidRequestError("image input is invalid", err)
		}
		imageURLs[index] = imageURL
	}

	schemaName := strings.TrimSpace(request.ResponseSchema.Name)
	if schemaName == "" {
		return nil, invalidRequestError("response schema name is required", nil)
	}
	schema := bytes.TrimSpace(request.ResponseSchema.Schema)
	var schemaObject map[string]any
	if len(schema) == 0 || json.Unmarshal(schema, &schemaObject) != nil || schemaObject == nil {
		return nil, invalidRequestError("response schema must be a valid JSON object", nil)
	}

	return &ProviderRequest{
		Prompt:    prompt,
		ImageURLs: imageURLs,
		Model:     strings.TrimSpace(request.Model),
		ResponseSchema: JSONSchema{
			Name:   schemaName,
			Schema: append(json.RawMessage(nil), schema...),
		},
	}, nil
}

func validateImageURL(value string) error {
	if strings.HasPrefix(value, "data:image/") {
		separator := strings.IndexByte(value, ',')
		if separator < 0 || !strings.HasSuffix(value[:separator], ";base64") || separator == len(value)-1 {
			return &url.Error{Op: "parse", URL: "data:image", Err: base64.CorruptInputError(0)}
		}
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(value[separator+1:]))
		if _, err := io.Copy(io.Discard, decoder); err != nil {
			return err
		}
		return nil
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return &url.Error{Op: "parse", URL: value, Err: errUnsupportedImageURL}
	}
	return nil
}

var errUnsupportedImageURL = &imageURLError{}

type imageURLError struct{}

func (*imageURLError) Error() string { return "image URL must use HTTP(S) or an image data URI" }

func invalidResponseError(message string, cause error) *ProviderError {
	return &ProviderError{
		Kind:      ErrorKindInvalidResponse,
		Transient: true,
		Message:   message,
		Cause:     cause,
	}
}

var _ LLMService = (*llmService)(nil)
