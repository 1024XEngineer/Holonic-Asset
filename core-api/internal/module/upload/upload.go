package upload

import "errors"

var (
	ErrInvalidUploadRequest = errors.New("upload: invalid upload request")
	ErrInvalidStorageConfig = errors.New("upload: invalid storage config")
	ErrInvalidObjectData    = errors.New("upload: invalid object data")
	ErrObjectNotFound       = errors.New("upload: object not found")
	ErrUntrustedReference   = errors.New("upload: untrusted reference")
)

// CreateUploadTargetRequest is the use-case input for creating an upload target.
type CreateUploadTargetRequest struct {
	ContentType   string
	ContentLength int64
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
