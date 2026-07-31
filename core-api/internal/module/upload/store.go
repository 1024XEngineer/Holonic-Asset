package upload

import "context"

// Store defines the object operations used by Core API modules.
type Store interface {
	CreateUploadTarget(context.Context, UploadRequest) (*UploadTarget, error)
	GetObjectMetadata(context.Context, string) (*ObjectMetadata, error)
	DeleteObject(context.Context, string) error
}
