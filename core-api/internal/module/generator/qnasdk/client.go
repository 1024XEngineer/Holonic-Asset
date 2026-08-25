// Package qnasdk contains the shared OpenAI-compatible client configuration
// used for requests sent through the Modelink/QNA gateway.
package qnasdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultBaseURL = "https://api.qnaigc.com"

// Client wraps the official OpenAI Go client with QNA gateway defaults.
// Protocol adapters own the request and response DTOs because QNA extends the
// OpenAI-compatible wire format for image generation and multimodal responses.
type Client struct {
	client *openai.Client
}

// ResponseMetadata contains non-sensitive diagnostics for one SDK request.
type ResponseMetadata struct {
	StatusCode int
	BodyBytes  int
}

// Error represents an HTTP error returned by the QNA gateway.
type Error struct {
	StatusCode int
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.StatusCode)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// NewClient creates a QNA client for one gateway and credential pair.
// SDK retries are disabled so service-level retries and model fallback remain
// the only retry policies in the application.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}

	options := []option.RequestOption{
		option.WithBaseURL(baseURL + "/"),
		option.WithAPIKey(strings.TrimSpace(apiKey)),
		option.WithMaxRetries(0),
	}
	if httpClient != nil {
		options = append(options, option.WithHTTPClient(httpClient))
	}

	client := openai.NewClient(options...)
	return &Client{client: &client}
}

// Execute sends a request through the configured QNA gateway client.
func (c *Client) Execute(ctx context.Context, method, path string, params, result any) error {
	_, err := c.ExecuteWithMetadata(ctx, method, path, params, result)
	return err
}

// ExecuteWithMetadata sends a request and reports status and response size for
// provider diagnostics without exposing response content.
func (c *Client) ExecuteWithMetadata(
	ctx context.Context,
	method, path string,
	params, result any,
) (ResponseMetadata, error) {
	var (
		response     *http.Response
		responseBody []byte
	)
	err := c.client.Execute(
		ctx,
		method,
		path,
		params,
		nil,
		option.WithResponseInto(&response),
		option.WithResponseBodyInto(&responseBody),
	)
	metadata := ResponseMetadata{BodyBytes: len(responseBody)}
	if response != nil {
		metadata.StatusCode = response.StatusCode
	}
	if err != nil {
		if response == nil || response.StatusCode < http.StatusBadRequest {
			return metadata, err
		}

		body := responseBody
		cause := err
		if len(body) == 0 && response.Body != nil {
			var readErr error
			body, readErr = io.ReadAll(response.Body)
			if readErr != nil {
				cause = errors.Join(err, readErr)
			}
		}
		metadata.BodyBytes = len(body)
		return metadata, &Error{
			StatusCode: response.StatusCode,
			Message:    responseErrorMessage(body, response.Status),
			Cause:      cause,
		}
	}
	if result != nil {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return metadata, fmt.Errorf("decode SDK response JSON: %w", err)
		}
	}
	return metadata, nil
}

func responseErrorMessage(body []byte, status string) string {
	var envelope struct {
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if message := strings.TrimSpace(envelope.Message); message != "" {
			return message
		}
		if len(envelope.Error) > 0 {
			var nested struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(envelope.Error, &nested); err == nil && strings.TrimSpace(nested.Message) != "" {
				return strings.TrimSpace(nested.Message)
			}
			var message string
			if err := json.Unmarshal(envelope.Error, &message); err == nil && strings.TrimSpace(message) != "" {
				return strings.TrimSpace(message)
			}
		}
	}
	if message := strings.TrimSpace(string(body)); message != "" {
		return message
	}
	return strings.TrimSpace(status)
}
