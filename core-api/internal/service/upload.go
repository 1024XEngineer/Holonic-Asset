package service

import (
	"context"

	"github.com/1024XEngineer/Holonic-Asset/pkg/storage"
)

// UploadService creates direct upload targets for files.
type UploadService interface {
	CreateUploadTarget(
		ctx context.Context,
		request *CreateUploadTargetRequest,
	) (*UploadTarget, error)
}

// uploadService provides the upload application skeleton.
type uploadService struct {
	storage storage.Storage
}

// NewUploadService creates the upload application service used by the HTTP handler.
func NewUploadService(objectStorage storage.Storage) UploadService {
	return &uploadService{storage: objectStorage}
}

func (*uploadService) CreateUploadTarget(
	context.Context,
	*CreateUploadTargetRequest,
) (*UploadTarget, error) {
	return &UploadTarget{}, nil
}

var _ UploadService = (*uploadService)(nil)
