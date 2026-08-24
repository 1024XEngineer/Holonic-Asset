package imageclient

import (
	"context"
	"errors"
	"strings"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/qnasdk"
)

func qnaSDKAPIError(err error) (statusCode int, message string, ok bool) {
	var apiErr *qnasdk.Error
	if !errors.As(err, &apiErr) {
		return 0, "", false
	}
	message = strings.TrimSpace(apiErr.Message)
	if message == "" {
		message = strings.TrimSpace(err.Error())
	}
	return apiErr.StatusCode, message, true
}

func isQNASDKConfigurationError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "withbaseurl failed") ||
		strings.Contains(message, "unsupported protocol scheme")
}

func isQNASDKResponseDecodeError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "error parsing response json") ||
		strings.Contains(message, "decode sdk response json")
}

func classifyQNAImageSDKError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return classifyQNARequestError(ctx, ctxErr)
	}
	if statusCode, message, ok := qnaSDKAPIError(err); ok {
		kind, transient := classifyQNAStatus(statusCode)
		return newQNAError(kind, statusCode, transient, message, err)
	}
	if isQNASDKConfigurationError(err) {
		return newQNAError(ErrorKindInvalidRequest, 0, false, "configure QNA SDK request", err)
	}
	if isQNASDKResponseDecodeError(err) {
		return newQNAError(ErrorKindInvalidResponse, 200, true, "decode image response", err)
	}
	return classifyQNARequestError(ctx, err)
}

func classifyChatSDKError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return classifyChatRequestError(ctx, ctxErr)
	}
	if statusCode, message, ok := qnaSDKAPIError(err); ok {
		kind, transient := classifyChatStatus(statusCode)
		return newChatProviderError(kind, statusCode, transient, message, err)
	}
	if isQNASDKConfigurationError(err) {
		return newChatProviderError(ErrorKindInvalidRequest, 0, false, "configure QNA SDK request", err)
	}
	if isQNASDKResponseDecodeError(err) {
		return newChatProviderError(ErrorKindInvalidResponse, 200, true, "decode chat completion response", err)
	}
	return classifyChatRequestError(ctx, err)
}
