package service

import (
	"context"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/model/generation"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// Transaction exposes generation persistence and module services bound to the
// same database transaction.
type Transaction interface {
	Repository() repository.Repository
	CandidateConfirmer() CandidateConfirmer
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, work func(context.Context, Transaction) error) error
}

// CandidateConfirmer hands an accepted candidate to the asset module without
// exposing Asset persistence or version internals to generation.
type CandidateConfirmer interface {
	Confirm(ctx context.Context, command *domain.ConfirmCandidateCommand) error
}
