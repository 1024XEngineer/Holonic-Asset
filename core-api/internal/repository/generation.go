package repository

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
)

// GenerationTaskReader builds generation list projections from task records.
// Generation owns the payload semantics but does not persist a separate run.
type GenerationTaskReader interface {
	ListRuns(ctx context.Context, filter *domain.RunListFilter) (*domain.RunListPage, error)
}
