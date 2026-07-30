package upload

import (
	"context"
)

// CreateUploadTargetRequest is the use-case input for creating an upload target.
type CreateUploadTargetRequest struct {
	ContentType   string
	ContentLength int64
}

// Uploader creates direct upload targets for files.
type Uploader interface {
	CreateUploadTarget(context.Context, *CreateUploadTargetRequest) (*UploadTarget, error)
}

// UploaderImpl coordinates upload-target creation with object storage.
type UploaderImpl struct {
	storage Storage
}

// NewUploader creates an upload use-case implementation backed by objectStorage.
func NewUploader(objectStorage Storage) *UploaderImpl {
	return &UploaderImpl{storage: objectStorage}
}

func (u *UploaderImpl) CreateUploadTarget(
	ctx context.Context,
	request *CreateUploadTargetRequest,
) (*UploadTarget, error) {
	if u.storage == nil {
		return &UploadTarget{}, nil
	}

	return u.storage.CreateUploadTarget(ctx, UploadRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
}

var _ Uploader = (*UploaderImpl)(nil)
