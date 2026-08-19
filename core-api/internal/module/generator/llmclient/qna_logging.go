package llmclient

import (
	"errors"
	"net/http"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

// The provider logs metadata only. Prompts, API keys, image data URIs and
// response content are intentionally excluded because scenery requests can be
// very large and may contain user/project data.
func (p *QNAProvider) logRequestSuccess(
	model string,
	imageCount int,
	schemaName string,
	endpoint string,
	startedAt time.Time,
	statusCode int,
	responseBytes int,
	responseModel string,
	requestID string,
	extra ...logger.Field,
) {
	if p.logger == nil {
		return
	}
	fields := qnaRequestFields(model, imageCount, schemaName, endpoint, startedAt, statusCode, responseBytes, responseModel, requestID)
	fields = append(fields, extra...)
	p.logger.Debug("qna llm request completed", fields...)
}

func (p *QNAProvider) logRequestFailure(
	model string,
	imageCount int,
	schemaName string,
	endpoint string,
	startedAt time.Time,
	statusCode int,
	responseBytes int,
	err error,
	extra ...logger.Field,
) {
	if p.logger == nil {
		return
	}
	fields := qnaRequestFields(model, imageCount, schemaName, endpoint, startedAt, statusCode, responseBytes, "", "")
	fields = append(fields, extra...)
	kind, transient := classifyQNARequestFailure(statusCode, err)
	fields = append(fields,
		logger.String("error_kind", string(kind)),
		logger.Any("transient", transient),
		logger.Error(err),
	)
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		fields = append(fields, logger.String("provider_message", providerErr.Message))
	}
	if errors.As(err, &providerErr) && providerErr.Cause != nil {
		fields = append(fields, logger.Any("cause", providerErr.Cause))
	}
	p.logger.Warn("qna llm request failed", fields...)
}

func qnaRequestFields(
	model string,
	imageCount int,
	schemaName string,
	endpoint string,
	startedAt time.Time,
	statusCode int,
	responseBytes int,
	responseModel string,
	requestID string,
) []logger.Field {
	return []logger.Field{
		logger.String("provider", qnaProviderName),
		logger.String("stage", "chat_completion"),
		logger.String("method", http.MethodPost),
		logger.String("endpoint", endpoint),
		logger.String("model", model),
		logger.String("response_schema", schemaName),
		logger.Int("image_count", imageCount),
		logger.Int("status_code", statusCode),
		logger.Int("response_bytes", responseBytes),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		logger.String("upstream_model", responseModel),
		logger.String("upstream_request_id", requestID),
	}
}

func classifyQNARequestFailure(statusCode int, err error) (ErrorKind, bool) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Kind, providerErr.Transient
	}
	if statusCode > 0 {
		return classifyQNAStatus(statusCode)
	}
	return ErrorKindTransport, true
}
