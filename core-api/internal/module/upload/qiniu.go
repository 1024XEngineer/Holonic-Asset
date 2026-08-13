package upload

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
)

const (
	defaultQiniuUploadURL  = "https://upload.qiniup.com"
	defaultUploadTokenTTL  = time.Hour
	defaultDownloadURLTTL  = 30 * time.Minute
	maxS3DownloadURLTTL    = 7 * 24 * time.Hour
	generatedObjectPrefix  = "uploads/"
	generatedObjectIDBytes = 16
	maxBatchOperations     = 1000
)

type qiniuBucketManager interface {
	Stat(bucket, key string) (qiniustorage.FileInfo, error)
	BatchWithContext(context.Context, string, []string) ([]qiniustorage.BatchOpRet, error)
}

type qiniuFormUploader interface {
	Put(context.Context, any, string, string, io.Reader, int64, *qiniustorage.PutExtra) error
}

// QiniuStorage stores objects in Qiniu Kodo and signs direct-upload tokens.
type QiniuStorage struct {
	bucket            string
	domain            string
	uploadURL         string
	uploadTokenExpiry time.Duration
	downloadURLExpiry time.Duration
	credentials       *qbox.Mac
	bucketManager     qiniuBucketManager
	uploader          qiniuFormUploader
	random            io.Reader
	now               func() time.Time
}

// NewQiniuStorage creates a Qiniu-backed storage service.
func NewQiniuStorage(cfg config.QiniuConfig) (*QiniuStorage, error) {
	accessKey := strings.TrimSpace(cfg.AccessKey)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	bucket := strings.TrimSpace(cfg.Bucket)
	if accessKey == "" {
		return nil, fmt.Errorf("%w: accessKey is required", ErrInvalidStorageConfig)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("%w: secretKey is required", ErrInvalidStorageConfig)
	}
	if bucket == "" {
		return nil, fmt.Errorf("%w: bucket is required", ErrInvalidStorageConfig)
	}

	domain, err := normalizeHTTPURL(cfg.Domain, "domain", "")
	if err != nil {
		return nil, err
	}
	uploadURL, err := normalizeHTTPURL(cfg.UploadURL, "uploadURL", defaultQiniuUploadURL)
	if err != nil {
		return nil, err
	}

	uploadTokenExpiry := cfg.UploadTokenExpiry
	if uploadTokenExpiry == 0 {
		uploadTokenExpiry = defaultUploadTokenTTL
	}
	if uploadTokenExpiry < time.Second {
		return nil, fmt.Errorf("%w: uploadTokenExpiry must be at least 1s", ErrInvalidStorageConfig)
	}
	downloadURLExpiry := cfg.DownloadURLExpiry
	if downloadURLExpiry == 0 {
		downloadURLExpiry = defaultDownloadURLTTL
	}
	if downloadURLExpiry < time.Second {
		return nil, fmt.Errorf("%w: downloadURLExpiry must be at least 1s", ErrInvalidStorageConfig)
	}
	if _, isS3Endpoint := qiniuS3Region(domain); isS3Endpoint && downloadURLExpiry > maxS3DownloadURLTTL {
		return nil, fmt.Errorf("%w: downloadURLExpiry must not exceed 168h for an S3 endpoint", ErrInvalidStorageConfig)
	}

	credentials := qbox.NewMac(accessKey, secretKey)
	return &QiniuStorage{
		bucket:            bucket,
		domain:            domain,
		uploadURL:         uploadURL,
		uploadTokenExpiry: uploadTokenExpiry,
		downloadURLExpiry: downloadURLExpiry,
		credentials:       credentials,
		bucketManager:     qiniustorage.NewBucketManager(credentials, qiniustorage.NewConfig()),
		uploader:          qiniustorage.NewFormUploader(qiniustorage.NewConfig()),
		random:            rand.Reader,
		now:               time.Now,
	}, nil
}

// CreateUploadTarget creates a temporary Qiniu upload target.
func (s *QiniuStorage) CreateUploadTarget(ctx context.Context, request UploadRequest) (*UploadTarget, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	contentType, _, err := mime.ParseMediaType(strings.TrimSpace(request.ContentType))
	mediaTypeParts := strings.Split(contentType, "/")
	if err != nil || len(mediaTypeParts) != 2 || mediaTypeParts[0] == "" || mediaTypeParts[1] == "" || strings.ContainsAny(contentType, "!*;, \t\r\n") {
		return nil, fmt.Errorf("%w: content type must be a concrete MIME type", ErrInvalidUploadRequest)
	}
	if request.ContentLength <= 0 {
		return nil, fmt.Errorf("%w: content length must be positive", ErrInvalidUploadRequest)
	}

	objectKey := strings.TrimSpace(request.ObjectKey)
	if objectKey == "" {
		generatedKey, err := s.generateObjectKey("")
		if err != nil {
			return nil, fmt.Errorf("upload: generate object key: %w", err)
		}
		objectKey = generatedKey
	}
	if err := validateObjectKey(objectKey); err != nil {
		return nil, err
	}

	policy := qiniustorage.PutPolicy{
		Scope:        s.bucket + ":" + objectKey,
		Expires:      validatedDurationSeconds(s.uploadTokenExpiry),
		InsertOnly:   1,
		SaveKey:      objectKey,
		ForceSaveKey: true,
		FsizeMin:     request.ContentLength,
		FsizeLimit:   request.ContentLength,
		DetectMime:   1,
		MimeLimit:    contentType,
	}

	privateURL, err := s.privateURL(ctx, objectKey)
	if err != nil {
		return nil, err
	}
	return &UploadTarget{
		ObjectKey:   objectKey,
		ObjectURL:   privateURL,
		UploadURL:   s.uploadURL,
		UploadToken: policy.UploadToken(s.credentials),
	}, nil
}

// GetObjectMetadata retrieves metadata for an object.
func (s *QiniuStorage) GetObjectMetadata(ctx context.Context, objectKey string) (*ObjectMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	objectKey = strings.TrimSpace(objectKey)
	if err := validateObjectKey(objectKey); err != nil {
		return nil, err
	}

	info, err := s.bucketManager.Stat(s.bucket, objectKey)
	if err != nil {
		return nil, fmt.Errorf("upload: get object metadata %q: %w", objectKey, err)
	}
	privateURL, err := s.privateURL(ctx, objectKey)
	if err != nil {
		return nil, err
	}
	return &ObjectMetadata{
		ObjectKey:     objectKey,
		ObjectURL:     privateURL,
		ContentType:   info.MimeType,
		ContentLength: info.Fsize,
	}, nil
}

func (s *QiniuStorage) privateURL(ctx context.Context, objectKey string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	objectKey = strings.TrimSpace(objectKey)
	if err := validateObjectKey(objectKey); err != nil {
		return "", err
	}
	now := s.now().UTC()
	expiresAt := now.Add(s.downloadURLExpiry).Unix()
	if region, ok := qiniuS3Region(s.domain); ok {
		signedURL, err := makeS3PrivateURL(
			ctx,
			s.credentials.AccessKey,
			string(s.credentials.SecretKey),
			s.domain,
			objectKey,
			region,
			now,
			s.downloadURLExpiry,
		)
		if err != nil {
			return "", fmt.Errorf("upload: sign S3 private URL: %w", err)
		}
		return signedURL, nil
	}
	return qiniustorage.MakePrivateURLv2(s.credentials, s.domain, objectKey, expiresAt), nil
}

// ResolveReference converts a persisted object key into a temporary private
// URL. Legacy data URLs and URLs hosted outside this storage are passed through.
func (s *QiniuStorage) ResolveReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" || strings.HasPrefix(reference, "data:") {
		return reference, nil
	}
	if parsed, err := url.Parse(reference); err == nil && parsed.IsAbs() {
		objectKey, ownDomain := s.objectKeyFromURL(parsed)
		if !ownDomain {
			return reference, nil
		}
		return s.privateURL(ctx, objectKey)
	}
	return s.privateURL(ctx, reference)
}

func (s *QiniuStorage) normalizeReference(reference string) string {
	reference = strings.TrimSpace(reference)
	parsed, err := url.Parse(reference)
	if err != nil || !parsed.IsAbs() {
		return reference
	}
	if objectKey, ownDomain := s.objectKeyFromURL(parsed); ownDomain {
		return objectKey
	}
	return reference
}

// PersistReference stores generated data URLs and otherwise performs the
// normalization needed before a reference is persisted.
func (s *QiniuStorage) PersistReference(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if !strings.HasPrefix(reference, "data:") {
		return s.normalizeReference(reference), nil
	}
	mediaType, data, err := decodeDataURL(reference)
	if err != nil {
		return "", err
	}
	return s.putObject(ctx, mediaType, data)
}

// NewObjectKey allocates an image object key without uploading content.
func (s *QiniuStorage) NewObjectKey(mediaType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(mediaType))
	if err != nil || !strings.HasPrefix(mediaType, "image/") {
		return "", fmt.Errorf("%w: concrete image media type is required", ErrInvalidObjectData)
	}
	return s.generateObjectKey(fileExtension(mediaType))
}

// PersistReferenceAt uploads a generated data URL to an exact object key.
func (s *QiniuStorage) PersistReferenceAt(
	ctx context.Context,
	objectKey string,
	reference string,
) error {
	objectKey = strings.TrimSpace(objectKey)
	if err := validateObjectKey(objectKey); err != nil {
		return err
	}
	mediaType, data, err := decodeDataURL(strings.TrimSpace(reference))
	if err != nil {
		return err
	}
	return s.putObjectAt(ctx, objectKey, mediaType, data)
}

// DeleteObjects removes exact immutable keys allocated by a failed workflow.
func (s *QiniuStorage) DeleteObjects(ctx context.Context, objectKeys []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(objectKeys) == 0 {
		return nil
	}
	operations := make([]string, len(objectKeys))
	for index, objectKey := range objectKeys {
		objectKey = strings.TrimSpace(objectKey)
		if err := validateObjectKey(objectKey); err != nil {
			return err
		}
		operations[index] = qiniustorage.URIDelete(s.bucket, objectKey)
	}
	var deleteErrors []error
	for offset := 0; offset < len(operations); offset += maxBatchOperations {
		end := min(offset+maxBatchOperations, len(operations))
		responses, err := s.bucketManager.BatchWithContext(ctx, s.bucket, operations[offset:end])
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("delete object batch %d-%d: %w", offset, end-1, err))
			continue
		}
		if len(responses) != end-offset {
			deleteErrors = append(deleteErrors, fmt.Errorf(
				"delete object batch %d-%d: got %d responses, want %d",
				offset, end-1, len(responses), end-offset,
			))
		}
		for index, response := range responses[:min(len(responses), end-offset)] {
			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				deleteErrors = append(deleteErrors, fmt.Errorf(
					"delete object %q: status %d: %s",
					objectKeys[offset+index], response.Code, response.Data.Error,
				))
			}
		}
	}
	if err := errors.Join(deleteErrors...); err != nil {
		return fmt.Errorf("upload: delete objects: %w", err)
	}
	return nil
}

func (s *QiniuStorage) putObject(ctx context.Context, mediaType string, data []byte) (string, error) {
	objectKey, err := s.NewObjectKey(mediaType)
	if err != nil {
		return "", fmt.Errorf("upload: generate object key: %w", err)
	}
	if err := s.putObjectAt(ctx, objectKey, mediaType, data); err != nil {
		return "", err
	}
	return objectKey, nil
}

func (s *QiniuStorage) putObjectAt(
	ctx context.Context,
	objectKey string,
	mediaType string,
	data []byte,
) error {
	policy := qiniustorage.PutPolicy{
		Scope:        s.bucket + ":" + objectKey,
		Expires:      validatedDurationSeconds(s.uploadTokenExpiry),
		InsertOnly:   1,
		SaveKey:      objectKey,
		ForceSaveKey: true,
		FsizeMin:     int64(len(data)),
		FsizeLimit:   int64(len(data)),
		DetectMime:   1,
		MimeLimit:    mediaType,
	}
	var result qiniustorage.PutRet
	err := s.uploader.Put(
		ctx,
		&result,
		policy.UploadToken(s.credentials),
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		&qiniustorage.PutExtra{MimeType: mediaType, UpHost: s.uploadURL},
	)
	if err != nil {
		return fmt.Errorf("upload: put object %q: %w", objectKey, err)
	}
	return nil
}

func (s *QiniuStorage) generateObjectKey(suffix string) (string, error) {
	randomBytes := make([]byte, generatedObjectIDBytes)
	if _, err := io.ReadFull(s.random, randomBytes); err != nil {
		return "", err
	}
	return generatedObjectPrefix + hex.EncodeToString(randomBytes) + suffix, nil
}

func (s *QiniuStorage) objectKeyFromURL(reference *url.URL) (string, bool) {
	domain, err := url.Parse(s.domain)
	if err != nil || !strings.EqualFold(reference.Host, domain.Host) {
		return "", false
	}
	path := reference.EscapedPath()
	objectKey, err := url.PathUnescape(strings.TrimPrefix(path, "/"))
	if err != nil || objectKey == "" {
		return "", false
	}
	return objectKey, true
}

func decodeDataURL(value string) (string, []byte, error) {
	header, payload, ok := strings.Cut(value, ",")
	if !ok || !strings.HasPrefix(header, "data:") || !strings.HasSuffix(strings.ToLower(header), ";base64") {
		return "", nil, fmt.Errorf("%w: expected a base64 data URL", ErrInvalidObjectData)
	}
	mediaType := strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if !strings.HasPrefix(mediaType, "image/") {
		return "", nil, fmt.Errorf("%w: only image data URLs are supported", ErrInvalidObjectData)
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(data) == 0 {
		return "", nil, fmt.Errorf("%w: invalid base64 payload", ErrInvalidObjectData)
	}
	return mediaType, data, nil
}

func fileExtension(mediaType string) string {
	if extensions, err := mime.ExtensionsByType(mediaType); err == nil && len(extensions) > 0 {
		return extensions[0]
	}
	return ""
}

func validatedDurationSeconds(value time.Duration) uint64 {
	// The storage constructor rejects non-positive token durations.
	return uint64(value / time.Second) //nolint:gosec // Conversion is safe after configuration validation.
}

func qiniuS3Region(domain string) (string, bool) {
	parsed, err := url.Parse(domain)
	if err != nil {
		return "", false
	}
	labels := strings.Split(strings.ToLower(parsed.Hostname()), ".")
	for index, label := range labels {
		if label == "s3" && index+1 < len(labels) {
			return labels[index+1], true
		}
	}
	return "", false
}

func makeS3PrivateURL(
	ctx context.Context,
	accessKey string,
	secretKey string,
	domain string,
	objectKey string,
	region string,
	now time.Time,
	expiry time.Duration,
) (string, error) {
	objectURL := qiniustorage.MakePublicURLv2(domain, objectKey)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, objectURL, nil)
	if err != nil {
		return "", err
	}
	query := request.URL.Query()
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", int64(expiry/time.Second)))
	request.URL.RawQuery = query.Encode()
	signedURL, _, err := awsv4.NewSigner().PresignHTTP(
		ctx,
		aws.Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey},
		request,
		"UNSIGNED-PAYLOAD",
		"s3",
		region,
		now,
		func(options *awsv4.SignerOptions) {
			options.DisableURIPathEscaping = true
		},
	)
	return signedURL, err
}

func validateObjectKey(objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("%w: object key is required", ErrInvalidUploadRequest)
	}
	if len(objectKey) > 750 {
		return fmt.Errorf("%w: object key exceeds 750 bytes", ErrInvalidUploadRequest)
	}
	if strings.HasPrefix(objectKey, "/") || strings.ContainsAny(objectKey, "\r\n\x00") {
		return fmt.Errorf("%w: object key is invalid", ErrInvalidUploadRequest)
	}
	return nil
}

func normalizeHTTPURL(value, field, fallback string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidStorageConfig, field)
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: %s must be an HTTP(S) URL", ErrInvalidStorageConfig, field)
	}
	return strings.TrimRight(value, "/"), nil
}

var _ Store = (*QiniuStorage)(nil)
var _ ReferenceStore = (*QiniuStorage)(nil)
