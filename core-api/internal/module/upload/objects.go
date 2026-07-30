package upload

import (
	"context"
	"time"
)

// QiniuConfig describes the configuration required by Qiniu Object Storage.
type QiniuConfig struct {
	AccessKey         string
	SecretKey         string
	Bucket            string
	Domain            string
	UploadURL         string
	UploadTokenExpiry time.Duration
}

// UploadRequest describes a direct browser upload.
type UploadRequest struct {
	ObjectKey     string
	ContentType   string
	ContentLength int64
}

// UploadTarget contains the information required for a direct upload.
type UploadTarget struct {
	ObjectKey   string
	ObjectURL   string
	UploadURL   string
	UploadToken string
}

// ObjectMetadata is the subset of object metadata needed to validate an upload.
type ObjectMetadata struct {
	ObjectKey     string
	ObjectURL     string
	ContentType   string
	ContentLength int64
}

// Storage defines the object operations used by Core API modules.
type Storage interface {
	CreateUploadTarget(context.Context, UploadRequest) (*UploadTarget, error)
	GetObjectMetadata(context.Context, string) (*ObjectMetadata, error)
	DeleteObject(context.Context, string) error
}

// QiniuStorage is the Qiniu Object Storage implementation skeleton.
type QiniuStorage struct{}

// NewQiniuStorage creates a Qiniu-backed storage service.
// Qiniu client initialization is intentionally deferred.
func NewQiniuStorage(QiniuConfig) (*QiniuStorage, error) {
	return &QiniuStorage{}, nil
}

// CreateUploadTarget creates a temporary Qiniu upload target.
// Upload token generation is intentionally deferred.
func (*QiniuStorage) CreateUploadTarget(context.Context, UploadRequest) (*UploadTarget, error) {
	return &UploadTarget{}, nil
}

// GetObjectMetadata retrieves metadata for an object.
// Qiniu resource access is intentionally deferred.
func (*QiniuStorage) GetObjectMetadata(context.Context, string) (*ObjectMetadata, error) {
	return &ObjectMetadata{}, nil
}

// DeleteObject deletes an object.
// Qiniu resource access is intentionally deferred.
func (*QiniuStorage) DeleteObject(context.Context, string) error {
	return nil
}

var _ Storage = (*QiniuStorage)(nil)
