package videoclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// DefaultQNABaseURL is the production endpoint documented by QNA.
	DefaultQNABaseURL = "https://api.qnaigc.com"
	// DefaultQNACreatePath is QNA's Seedance image-to-video endpoint.
	DefaultQNACreatePath = "/queue/bytedance/seedance-2.0/image-to-video"
	// DefaultQNAResultPath is QNA's Seedance task result endpoint prefix.
	DefaultQNAResultPath = "/queue/bytedance/seedance-2.0/requests"
	// DefaultQNAResolution is used when a request does not specify a resolution.
	DefaultQNAResolution = "720p"
	// DefaultQNADuration is used when a request duration is outside 4-15 seconds.
	DefaultQNADuration = 5
	// DefaultQNAAspectRatio is used when a request does not specify an aspect ratio.
	DefaultQNAAspectRatio = "1:1"

	qnaProviderName              = "qna"
	maxQNAPromptCharacters       = 2450
	maxQNAResponseBytes    int64 = 128 << 20
	maxQNAVideoBytes       int64 = 512 << 20
	defaultQNAHTTPTimeout        = 10 * time.Minute
	defaultQNAPollInterval       = 4 * time.Second
)

// QNAConfig configures the QNA video provider.
type QNAConfig struct {
	BaseURL      string
	APIKey       string
	CreatePath   string
	ResultPath   string
	Resolution   string
	Duration     int
	AspectRatio  string
	PollInterval time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
	HTTPClient   *http.Client
}

// QNAProvider calls QNA's asynchronous Seedance image-to-video API.
type QNAProvider struct {
	baseURL      string
	apiKey       string
	createPath   string
	resultPath   string
	resolution   string
	duration     int
	aspectRatio  string
	pollInterval time.Duration
	maxRetries   int
	retryDelay   time.Duration
	httpClient   *http.Client
}

// NewQNAProvider creates a QNA video provider with production defaults.
func NewQNAProvider(config QNAConfig) *QNAProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultQNABaseURL
	}
	createPath := strings.TrimSpace(config.CreatePath)
	if createPath == "" {
		createPath = DefaultQNACreatePath
	}
	resultPath := strings.TrimSpace(config.ResultPath)
	if resultPath == "" {
		resultPath = DefaultQNAResultPath
	}
	resolution := strings.TrimSpace(config.Resolution)
	if resolution == "" {
		resolution = DefaultQNAResolution
	}
	duration := config.Duration
	if duration < 4 || duration > 15 {
		duration = DefaultQNADuration
	}
	aspectRatio := strings.TrimSpace(config.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = DefaultQNAAspectRatio
	}
	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultQNAPollInterval
	}
	maxRetries := max(config.MaxRetries, 0)
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultQNAHTTPTimeout}
	}

	return &QNAProvider{
		baseURL:      baseURL,
		apiKey:       strings.TrimSpace(config.APIKey),
		createPath:   createPath,
		resultPath:   resultPath,
		resolution:   resolution,
		duration:     duration,
		aspectRatio:  aspectRatio,
		pollInterval: pollInterval,
		maxRetries:   maxRetries,
		retryDelay:   config.RetryDelay,
		httpClient:   httpClient,
	}
}

// Generate creates a video task and waits until QNA returns a video URL. The
// caller should use a context deadline to bound the complete polling operation.
func (p *QNAProvider) Generate(
	ctx context.Context,
	request *ProviderRequest,
) (*ProviderResult, error) {
	if request == nil {
		return nil, p.error(ErrorKindInvalidRequest, 0, false, "video request is nil", nil)
	}
	if p.apiKey == "" {
		return nil, p.error(ErrorKindAuthentication, 0, false, "API key is empty", nil)
	}
	prompt := limitCharacters(request.Prompt, maxQNAPromptCharacters)
	if prompt == "" || strings.TrimSpace(request.ReferenceImageURL) == "" {
		return nil, p.error(
			ErrorKindInvalidRequest,
			0,
			false,
			"reference image and prompt are required",
			nil,
		)
	}
	createURL, err := p.endpoint(p.createPath)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(qnaVideoRequest{
		Prompt:        prompt,
		ImageURL:      request.ReferenceImageURL,
		Resolution:    firstNonEmpty(request.Resolution, p.resolution),
		Duration:      fmt.Sprintf("%d", validDuration(request.Duration, p.duration)),
		AspectRatio:   firstNonEmpty(request.AspectRatio, p.aspectRatio),
		GenerateAudio: request.GenerateAudio,
	})
	if err != nil {
		return nil, p.error(ErrorKindInvalidRequest, 0, false, "encode video request", err)
	}

	created, body, err := p.createTask(ctx, createURL, payload)
	if err != nil {
		return nil, err
	}
	if videoURL := responseVideoURL(created); videoURL != "" {
		return &ProviderResult{RequestID: created.RequestID, VideoURL: videoURL}, nil
	}
	if strings.TrimSpace(created.RequestID) == "" {
		return nil, p.error(
			ErrorKindInvalidResponse,
			http.StatusOK,
			true,
			"video create response has no request_id: "+responseMessage(body),
			nil,
		)
	}

	return p.waitForTask(ctx, strings.TrimSpace(created.RequestID))
}

func (p *QNAProvider) createTask(
	ctx context.Context,
	endpoint string,
	payload []byte,
) (qnaVideoResponse, []byte, error) {
	attempts := p.maxRetries + 1
	var decoded qnaVideoResponse
	var body []byte
	for attempt := 1; attempt <= attempts; attempt++ {
		decoded = qnaVideoResponse{}
		status, responseBody, err := p.doJSON(ctx, http.MethodPost, endpoint, payload, &decoded)
		body = responseBody
		if err != nil {
			// A POST transport failure is ambiguous: QNA may already have accepted
			// the billable task, so only explicit transient HTTP responses are retried.
			return qnaVideoResponse{}, body, err
		}
		if status >= http.StatusOK && status < http.StatusMultipleChoices {
			return decoded, body, nil
		}
		if attempt == attempts || !retryableQNAStatus(status) {
			return qnaVideoResponse{}, body, p.statusError(status, body)
		}
		if err := p.sleepBeforeRetry(ctx, attempt); err != nil {
			return qnaVideoResponse{}, body, err
		}
	}
	return qnaVideoResponse{}, body, p.error(
		ErrorKindInvalidResponse,
		0,
		true,
		"video create failed without a response",
		nil,
	)
}

func (p *QNAProvider) waitForTask(ctx context.Context, requestID string) (*ProviderResult, error) {
	resultURL, err := p.endpoint(path.Join(p.resultPath, url.PathEscape(requestID)))
	if err != nil {
		return nil, err
	}
	for {
		if err := sleepWithContext(ctx, p.pollInterval); err != nil {
			return nil, p.contextError(ctx, err)
		}
		status, body, state, err := p.pollTask(ctx, resultURL)
		if err != nil {
			return nil, err
		}
		if status == http.StatusOK {
			if videoURL := responseVideoURL(state); videoURL != "" {
				return &ProviderResult{RequestID: requestID, VideoURL: videoURL}, nil
			}
			if taskFailed(state.Status) {
				return nil, p.error(
					ErrorKindTaskFailed,
					status,
					false,
					"video task "+state.Status+": "+responseMessage(body),
					nil,
				)
			}
			continue
		}
		if status == http.StatusBadRequest && taskInProgress(state) {
			continue
		}
		return nil, p.statusError(status, body)
	}
}

func (p *QNAProvider) pollTask(
	ctx context.Context,
	endpoint string,
) (int, []byte, qnaVideoResponse, error) {
	attempts := p.maxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		var decoded qnaVideoResponse
		status, body, err := p.doJSON(ctx, http.MethodGet, endpoint, nil, &decoded)
		if err == nil && !retryableQNAStatus(status) {
			return status, body, decoded, nil
		}
		if attempt == attempts {
			if err != nil {
				return status, body, qnaVideoResponse{}, err
			}
			return status, body, decoded, p.statusError(status, body)
		}
		if err := p.sleepBeforeRetry(ctx, attempt); err != nil {
			return status, body, qnaVideoResponse{}, err
		}
	}
	return 0, nil, qnaVideoResponse{}, p.error(
		ErrorKindInvalidResponse,
		0,
		true,
		"video result failed without a response",
		nil,
	)
}

// Download downloads a generated video with bounded memory usage and retries
// transient GET failures. Provider credentials are not forwarded to arbitrary
// result hosts; QNA video URLs are expected to be public or signed URLs.
func (p *QNAProvider) Download(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, p.error(ErrorKindInvalidRequest, 0, false, "invalid video URL", err)
	}
	attempts := p.maxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
		if requestErr != nil {
			return nil, p.error(ErrorKindInvalidRequest, 0, false, "create video download request", requestErr)
		}
		response, requestErr := p.httpClient.Do(request)
		if requestErr != nil {
			if attempt == attempts {
				return nil, p.contextError(ctx, requestErr)
			}
			if err := p.sleepBeforeRetry(ctx, attempt); err != nil {
				return nil, err
			}
			continue
		}

		data, readErr := readLimited(response.Body, maxQNAVideoBytes)
		closeErr := response.Body.Close()
		if readErr == nil {
			readErr = closeErr
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			if attempt < attempts && retryableQNAStatus(response.StatusCode) {
				if err := p.sleepBeforeRetry(ctx, attempt); err != nil {
					return nil, err
				}
				continue
			}
			return nil, p.statusError(response.StatusCode, data)
		}
		if readErr != nil {
			if attempt == attempts {
				return nil, p.error(
					ErrorKindInvalidResponse,
					response.StatusCode,
					true,
					"read generated video",
					readErr,
				)
			}
			if err := p.sleepBeforeRetry(ctx, attempt); err != nil {
				return nil, err
			}
			continue
		}
		return data, nil
	}
	return nil, p.error(
		ErrorKindInvalidResponse,
		0,
		true,
		"download generated video failed without a response",
		nil,
	)
}

func (p *QNAProvider) doJSON(
	ctx context.Context,
	method string,
	endpoint string,
	payload []byte,
	target any,
) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, nil, p.error(ErrorKindInvalidRequest, 0, false, "create video API request", err)
	}
	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return 0, nil, p.contextError(ctx, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	data, err := readLimited(response.Body, maxQNAResponseBytes)
	if err != nil {
		return response.StatusCode, nil, p.error(
			ErrorKindInvalidResponse,
			response.StatusCode,
			true,
			"read video API response",
			err,
		)
	}
	if len(bytes.TrimSpace(data)) > 0 && target != nil {
		if err := json.Unmarshal(data, target); err != nil &&
			response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return response.StatusCode, data, p.error(
				ErrorKindInvalidResponse,
				response.StatusCode,
				true,
				"decode video API response",
				err,
			)
		}
	}
	return response.StatusCode, data, nil
}

func (p *QNAProvider) endpoint(endpointPath string) (string, error) {
	base, err := parseHTTPURL(p.baseURL)
	if err != nil {
		return "", p.error(ErrorKindInvalidRequest, 0, false, "invalid video API base", err)
	}
	base.Path = path.Join(strings.TrimSuffix(base.Path, "/"), endpointPath)
	base.RawPath = ""
	base.RawQuery = ""
	base.Fragment = ""
	return base.String(), nil
}

func (p *QNAProvider) sleepBeforeRetry(ctx context.Context, attempt int) error {
	if err := sleepWithContext(ctx, retryDelayForAttempt(p.retryDelay, attempt)); err != nil {
		return p.contextError(ctx, err)
	}
	return nil
}

func (p *QNAProvider) statusError(statusCode int, body []byte) error {
	kind, transient := classifyQNAStatus(statusCode)
	return p.error(kind, statusCode, transient, responseMessage(body), nil)
}

func (p *QNAProvider) contextError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return p.error(ErrorKindCanceled, 0, false, "request canceled", err)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return p.error(ErrorKindTimeout, 0, true, "request timed out", err)
	}
	return p.error(ErrorKindTransport, 0, true, "request failed", err)
}

func (p *QNAProvider) error(
	kind ErrorKind,
	statusCode int,
	transient bool,
	message string,
	cause error,
) *ProviderError {
	if strings.TrimSpace(message) == "" && cause != nil {
		message = cause.Error()
	}
	return &ProviderError{
		Provider:   qnaProviderName,
		Kind:       kind,
		StatusCode: statusCode,
		Transient:  transient,
		Message:    strings.TrimSpace(message),
		Cause:      cause,
	}
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
	case 520, 521, 522, 523, 524, 525:
		return ErrorKindUnavailable, true
	default:
		if statusCode >= http.StatusInternalServerError {
			return ErrorKindUnavailable, true
		}
		return ErrorKindInvalidRequest, false
	}
}

func retryableQNAStatus(statusCode int) bool {
	_, transient := classifyQNAStatus(statusCode)
	return transient
}

func taskFailed(status string) bool {
	return strings.EqualFold(status, "FAILED") || strings.EqualFold(status, "CANCELLED")
}

func taskInProgress(response qnaVideoResponse) bool {
	return response.Detail.Type == "request_in_progress" ||
		strings.EqualFold(response.Status, "IN_QUEUE") ||
		strings.EqualFold(response.Status, "IN_PROGRESS")
}

func responseVideoURL(response qnaVideoResponse) string {
	if value := strings.TrimSpace(response.Result.Video.URL); value != "" {
		return value
	}
	return strings.TrimSpace(response.Video.URL)
}

func responseMessage(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return "empty response"
	}
	var response qnaVideoResponse
	if err := json.Unmarshal(body, &response); err == nil {
		if value := strings.TrimSpace(response.Detail.Msg); value != "" {
			return value
		}
		if value := strings.TrimSpace(response.Message); value != "" {
			return value
		}
		if value := strings.TrimSpace(response.Error.Message); value != "" {
			return value
		}
	}
	value := strings.TrimSpace(string(body))
	if utf8.RuneCountInString(value) > 1200 {
		value = string([]rune(value)[:1200]) + "…"
	}
	return value
}

func limitCharacters(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if maximum <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return strings.TrimSpace(string(runes[:maximum]))
}

func validDuration(requested int, fallback int) int {
	if requested >= 4 && requested <= 15 {
		return requested
	}
	return fallback
}

func firstNonEmpty(value string, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func parseHTTPURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, fmt.Errorf("invalid HTTP URL %q", raw)
	}
	return parsed, nil
}

func readLimited(reader io.Reader, maximum int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("response exceeds %d bytes", maximum)
	}
	return data, nil
}

func retryDelayForAttempt(base time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	shift := min(attempt-1, 5)
	delay := base * time.Duration(1<<shift)
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type qnaVideoRequest struct {
	Prompt        string `json:"prompt"`
	ImageURL      string `json:"image_url"`
	Resolution    string `json:"resolution,omitempty"`
	Duration      string `json:"duration,omitempty"`
	AspectRatio   string `json:"aspect_ratio,omitempty"`
	GenerateAudio bool   `json:"generate_audio"`
}

type qnaVideoResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
	Detail    struct {
		Type string `json:"type"`
		Msg  string `json:"msg"`
	} `json:"detail"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Result struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	} `json:"result"`
	Video struct {
		URL string `json:"url"`
	} `json:"video"`
}

var _ VideoProvider = (*QNAProvider)(nil)
