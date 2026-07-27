package service

import (
	"context"
	"io"
	"time"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/ai"
)

type ProjectContextReader interface {
	GetProjectContext(ctx context.Context, projectID uint) (*domain.ProjectContext, error)
}

type MediaAccess struct {
	URL       string
	ExpiresAt time.Time
}

type MediaImport struct {
	Body        io.Reader
	ContentType string
	Filename    string
}

// MediaGateway resolves stable media references and imports provider outputs.
type MediaGateway interface {
	CreateAccess(ctx context.Context, ref domain.MediaRef) (*MediaAccess, error)
	Import(ctx context.Context, request *MediaImport) (*domain.MediaRef, error)
}

type UsageRepository interface {
	SaveUsage(ctx context.Context, usage domain.Usage) error
}
