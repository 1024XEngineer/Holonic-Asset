package upload

import "context"

// ArtifactStore stores generated non-image artifacts such as export packages.
type ArtifactStore interface {
	PutArtifact(context.Context, string, string, []byte) error
	ResolveReference(context.Context, string) (string, error)
}
