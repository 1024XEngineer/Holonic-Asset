package upload

import "context"

// Store defines the object operations used by Core API modules.
type Store interface {
	CreateUploadTarget(context.Context, UploadRequest) (*UploadTarget, error)
	GetObjectMetadata(context.Context, string) (*ObjectMetadata, error)
}

// ReferenceResolver converts persisted object keys to short-lived URLs at
// boundaries that need to read an object. It deliberately does not expose
// credentials or storage-specific configuration to callers.
type ReferenceResolver interface {
	ResolveReference(context.Context, string) (string, error)
}

// ReferenceStore adds persistence for generated data URLs. References are
// constrained to this store: data URLs are uploaded, configured-domain URLs
// are converted to object keys, and external URLs are rejected.
type ReferenceStore interface {
	ReferenceResolver
	PersistReference(context.Context, string) (string, error)
	NewObjectKey(string) (string, error)
	PersistReferenceAt(context.Context, string, string) error
	DeleteObjects(context.Context, []string) error
}

// ResourceStore supports generated resources whose stable object key is
// selected by the publishing workflow.
type ResourceStore interface {
	PutObject(context.Context, string, string, []byte) error
	DeleteObject(context.Context, string) error
}
