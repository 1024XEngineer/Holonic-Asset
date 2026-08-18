package videoclient

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func (p *QNAProvider) logRequestResult(
	trace qnaRequestTrace,
	method string,
	endpoint string,
	statusCode int,
	startedAt time.Time,
	server string,
	cfRay string,
	upstreamRequestID string,
	responseBytes int,
	responseState *qnaVideoResponse,
) {
	if p.logger == nil {
		return
	}
	inProgress := trace.Stage == "poll" && statusCode == http.StatusBadRequest &&
		responseState != nil && taskInProgress(*responseState)
	success := statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
	var kind ErrorKind
	var transient bool
	if !success && !inProgress {
		kind, transient = classifyQNAStatus(statusCode)
	}
	fields := p.requestLogFields(
		trace,
		method,
		endpoint,
		statusCode,
		startedAt,
		server,
		cfRay,
		upstreamRequestID,
		trace.Attempt < trace.MaxAttempts && transient,
	)
	fields = append(fields, logger.Int("response_bytes", responseBytes))
	if inProgress {
		fields = append(fields,
			logger.String("task_status", strings.TrimSpace(responseState.Status)),
			logger.String("detail_type", strings.TrimSpace(responseState.Detail.Type)),
			logger.Any("will_poll_again", true),
		)
		p.logger.Debug("qna video task still in progress", fields...)
		return
	}

	if success {
		message := "qna video API request completed"
		if trace.Stage == "download" {
			message = "qna video download completed"
		}
		p.logger.Debug(message, fields...)
		return
	}
	fields = append(fields,
		logger.String("error_kind", string(kind)),
		logger.Any("transient", transient),
	)
	message := "qna video API request failed"
	if trace.Stage == "download" {
		message = "qna video download failed"
	}
	p.logger.Warn(message, fields...)
}

func (p *QNAProvider) logRequestFailure(
	message string,
	trace qnaRequestTrace,
	method string,
	endpoint string,
	statusCode int,
	startedAt time.Time,
	server string,
	cfRay string,
	upstreamRequestID string,
	willRetry bool,
	err error,
) {
	if p.logger == nil {
		return
	}
	fields := p.requestLogFields(
		trace,
		method,
		endpoint,
		statusCode,
		startedAt,
		server,
		cfRay,
		upstreamRequestID,
		willRetry,
	)
	kind, transient := classifyQNARequestFailure(statusCode, err)
	fields = append(fields,
		logger.String("error_kind", string(kind)),
		logger.Any("transient", transient),
		logger.Error(qnaLogError(err, endpoint)),
	)
	p.logger.Warn(message, fields...)
}

func (p *QNAProvider) requestLogFields(
	trace qnaRequestTrace,
	method string,
	endpoint string,
	statusCode int,
	startedAt time.Time,
	server string,
	cfRay string,
	upstreamRequestID string,
	willRetry bool,
) []logger.Field {
	network := trace.Network.snapshot(startedAt, trace.Protocol)
	return []logger.Field{
		logger.String("provider", qnaProviderName),
		logger.String("stage", trace.Stage),
		logger.String("method", method),
		logger.String("endpoint", endpointLogValue(endpoint)),
		logger.String("request_id", trace.RequestID),
		logger.Int("attempt", trace.Attempt),
		logger.Int("max_attempts", trace.MaxAttempts),
		logger.Int("status_code", statusCode),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		logger.Int64("dns_ms", network.dnsMS),
		logger.Int64("connect_ms", network.connectMS),
		logger.Int64("tls_ms", network.tlsMS),
		logger.Int64("time_to_first_byte_ms", network.timeToFirstByte),
		logger.String("remote_addr", strings.TrimSpace(network.remoteAddr)),
		logger.String("tls_version", network.tlsVersion),
		logger.String("http_protocol", network.protocol),
		logger.Any("connection_reused", network.connectionReused),
		logger.Any("connection_was_idle", network.connectionIdle),
		logger.String("upstream_server", strings.TrimSpace(server)),
		logger.String("upstream_request_id", strings.TrimSpace(upstreamRequestID)),
		logger.String("cf_ray", strings.TrimSpace(cfRay)),
		logger.Any("will_retry", willRetry),
	}
}

func qnaLogError(err error, endpoint string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	parsed, parseErr := parseHTTPURL(endpoint)
	if parseErr == nil && parsed.RawQuery != "" {
		unsafeURL := parsed.String()
		parsed.RawQuery = ""
		parsed.ForceQuery = false
		parsed.Fragment = ""
		message = strings.ReplaceAll(message, unsafeURL, parsed.String())
	}
	return errors.New(message)
}
