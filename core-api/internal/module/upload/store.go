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

// ReferenceStore adds persistence for generated data URLs. PersistReference
// leaves external URLs untouched and converts URLs on the configured object
// domain back to their object key.
type ReferenceStore interface {
	ReferenceResolver
	PersistReference(context.Context, string) (string, error)
	NewObjectKey(string) (string, error)
	PersistReferenceAt(context.Context, string, string) error
}
