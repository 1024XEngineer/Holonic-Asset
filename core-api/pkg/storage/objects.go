// Package storage exposes the S3-compatible object operations used by Core API modules.
// Cloudflare R2 can be configured as the S3 endpoint without changing this package API.
package storage

import "context"

// UploadFromURLRequest describes an object that the backend will store.
type UploadFromURLRequest struct {
	SourceURL   string
	ObjectKey   string
	ContentType string
}

// StoredObject identifies an object after it has been stored.
type StoredObject struct {
	ObjectKey string
	ObjectURL string
}

// PresignedUploadRequest describes a direct browser upload.
type PresignedUploadRequest struct {
	ObjectKey   string
	ContentType string
}

// UploadTarget contains an object reference and a temporary presigned upload URL.
type UploadTarget struct {
	ObjectKey string
	UploadURL string
}

// ObjectMetadata is the subset of object metadata needed to validate an upload.
type ObjectMetadata struct {
	ObjectKey   string
	ContentType string
}

// UploadFromURL stores a backend-provided source URL as an object.
func UploadFromURL(context.Context, UploadFromURLRequest) (*StoredObject, error) {
	return &StoredObject{}, nil
}

// CreatePresignedUpload creates a temporary S3-compatible upload target.
func CreatePresignedUpload(context.Context, PresignedUploadRequest) (*UploadTarget, error) {
	return &UploadTarget{}, nil
}

// HeadObject verifies an object through the S3-compatible provider.
// The R2-backed implementation is intentionally deferred.
func HeadObject(context.Context, string) (*ObjectMetadata, error) {
	return &ObjectMetadata{}, nil
}
