package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

func (s *animationGenerationService) loadAnimationReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", fmt.Errorf("generator: animation reference image is required")
	}

	if s.referenceResolver != nil {
		resolved, err := s.referenceResolver.ResolveReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("generator: resolve animation reference: %w", err)
		}
		reference = strings.TrimSpace(resolved)
		if reference == "" {
			return "", fmt.Errorf("generator: resolve animation reference: empty result")
		}
	}

	if !strings.HasPrefix(reference, "http://") && !strings.HasPrefix(reference, "https://") {
		return reference, nil
	}

	parsed, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("generator: parse animation reference URL: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("generator: parse animation reference URL: host is required")
	}
	body, err := s.downloadAnimationReference(ctx, parsed)
	if err != nil {
		return "", err
	}
	if len(body) > maxAnimationReferenceBytes {
		return "", fmt.Errorf("generator: animation reference exceeds %d bytes", maxAnimationReferenceBytes)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("generator: download animation reference: empty response")
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	decoded, err := imageprocessor.DecodeBase64Image(encoded)
	if err != nil {
		return "", fmt.Errorf("generator: decode downloaded animation reference: %w", err)
	}
	canonical, err := imageprocessor.EncodePNGBase64(decoded)
	if err != nil {
		return "", fmt.Errorf("generator: encode downloaded animation reference: %w", err)
	}
	return canonical, nil
}

func (s *animationGenerationService) downloadAnimationReference(ctx context.Context, parsed *url.URL) ([]byte, error) {
	client := s.referenceHTTPClient
	if client == nil {
		client = newDefaultAnimationReferenceHTTPClient()
	}
	maxRetries := max(s.referenceMaxRetries, 0)
	attempts := maxRetries + 1
	timeout := s.referenceTimeout
	if timeout <= 0 {
		timeout = defaultAnimationReferenceTimeout
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		startedAt := time.Now()
		requestContext, cancel := context.WithTimeout(ctx, timeout)
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, parsed.String(), nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("generator: create animation reference download request: %w", err)
		}
		response, err := client.Do(request)
		if err != nil {
			cancel()
			willRetry := attempt < attempts && ctx.Err() == nil && animationReferenceRequestTransient(err)
			s.logAnimationReferenceFailure(
				"animation reference download transport failure",
				parsed,
				attempt,
				attempts,
				0,
				startedAt,
				0,
				"",
				"",
				willRetry,
				err,
			)
			if !willRetry {
				return nil, fmt.Errorf("generator: download animation reference: %w", err)
			}
			if err := s.sleepBeforeAnimationReferenceRetry(ctx, attempt); err != nil {
				return nil, fmt.Errorf("generator: download animation reference: %w", err)
			}
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxAnimationReferenceBytes+1))
		closeErr := response.Body.Close()
		cancel()
		if readErr == nil {
			readErr = closeErr
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			willRetry := attempt < attempts && ctx.Err() == nil && animationReferenceStatusTransient(response.StatusCode)
			s.logAnimationReferenceResponse(
				parsed,
				attempt,
				attempts,
				response.StatusCode,
				startedAt,
				len(body),
				response.Header.Get("Server"),
				animationReferenceRequestID(response.Header),
				willRetry,
			)
			if willRetry {
				if err := s.sleepBeforeAnimationReferenceRetry(ctx, attempt); err != nil {
					return nil, fmt.Errorf("generator: download animation reference: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf(
				"generator: download animation reference: HTTP %d: %s",
				response.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}
		if readErr != nil {
			willRetry := attempt < attempts && ctx.Err() == nil && animationReferenceRequestTransient(readErr)
			s.logAnimationReferenceFailure(
				"animation reference response read failure",
				parsed,
				attempt,
				attempts,
				response.StatusCode,
				startedAt,
				len(body),
				response.Header.Get("Server"),
				animationReferenceRequestID(response.Header),
				willRetry,
				readErr,
			)
			if !willRetry {
				return nil, fmt.Errorf("generator: read animation reference: %w", readErr)
			}
			if err := s.sleepBeforeAnimationReferenceRetry(ctx, attempt); err != nil {
				return nil, fmt.Errorf("generator: read animation reference: %w", err)
			}
			continue
		}
		s.logAnimationReferenceResponse(
			parsed,
			attempt,
			attempts,
			response.StatusCode,
			startedAt,
			len(body),
			response.Header.Get("Server"),
			animationReferenceRequestID(response.Header),
			false,
		)
		return body, nil
	}
	return nil, fmt.Errorf("generator: download animation reference: exhausted retries")
}

func animationReferenceRequestID(header http.Header) string {
	for _, name := range []string{"X-Reqid", "Request-Id", "X-Request-Id"} {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func animationReferenceStatusTransient(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooEarly ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func animationReferenceRequestTransient(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled)
}

func (s *animationGenerationService) sleepBeforeAnimationReferenceRetry(ctx context.Context, attempt int) error {
	delay := s.referenceRetryDelay
	if delay <= 0 {
		delay = defaultAnimationReferenceRetryDelay
	}
	for range min(attempt-1, 4) {
		delay *= 2
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *animationGenerationService) animationReferenceLogFields(
	parsed *url.URL,
	attempt int,
	maxAttempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
	server string,
	upstreamRequestID string,
	willRetry bool,
) []logger.Field {
	return []logger.Field{
		logger.String("provider", "reference_storage"),
		logger.String("stage", "download_animation_reference"),
		logger.String("method", http.MethodGet),
		logger.String("source_host", parsed.Hostname()),
		logger.String("endpoint", parsed.EscapedPath()),
		logger.Int("attempt", attempt),
		logger.Int("max_attempts", maxAttempts),
		logger.Int("status_code", statusCode),
		logger.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		logger.Int("response_bytes", responseBytes),
		logger.String("upstream_server", strings.TrimSpace(server)),
		logger.String("upstream_request_id", strings.TrimSpace(upstreamRequestID)),
		logger.Any("will_retry", willRetry),
	}
}

func (s *animationGenerationService) logAnimationReferenceResponse(
	parsed *url.URL,
	attempt int,
	maxAttempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
	server string,
	upstreamRequestID string,
	willRetry bool,
) {
	if s.logger == nil {
		return
	}
	fields := s.animationReferenceLogFields(
		parsed,
		attempt,
		maxAttempts,
		statusCode,
		startedAt,
		responseBytes,
		server,
		upstreamRequestID,
		willRetry,
	)
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		s.logger.Debug("animation reference download completed", fields...)
		return
	}
	fields = append(fields,
		logger.String("error_kind", "http_status"),
		logger.Any("transient", animationReferenceStatusTransient(statusCode)),
	)
	s.logger.Warn("animation reference download failed", fields...)
}

func (s *animationGenerationService) logAnimationReferenceFailure(
	message string,
	parsed *url.URL,
	attempt int,
	maxAttempts int,
	statusCode int,
	startedAt time.Time,
	responseBytes int,
	server string,
	upstreamRequestID string,
	willRetry bool,
	err error,
) {
	if s.logger == nil {
		return
	}
	kind := "transport"
	if statusCode > 0 {
		kind = "response_read"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		kind = "timeout"
	} else if errors.Is(err, context.Canceled) {
		kind = "canceled"
	}
	fields := s.animationReferenceLogFields(
		parsed,
		attempt,
		maxAttempts,
		statusCode,
		startedAt,
		responseBytes,
		server,
		upstreamRequestID,
		willRetry,
	)
	fields = append(fields,
		logger.String("error_kind", kind),
		logger.Any("transient", animationReferenceRequestTransient(err)),
		logger.Error(err),
	)
	s.logger.Warn(message, fields...)
}
