package handler

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/pkg/echox"
)

type UploadHandler struct {
	uploader upload.Uploader
}

func NewUploadHandler(uploader upload.Uploader) *UploadHandler {
	return &UploadHandler{uploader: uploader}
}

func (h *UploadHandler) CreateUploadTarget(
	c *echox.Context,
	request dto.CreateUploadTargetRequest,
) (*dto.UploadTarget, error) {
	target, err := h.uploader.CreateUploadTarget(c, &upload.CreateUploadTargetRequest{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
	if err != nil {
		return nil, err
	}
	return &dto.UploadTarget{
		ObjectKey:   target.ObjectKey,
		ObjectURL:   target.ObjectURL,
		UploadURL:   target.UploadURL,
		UploadToken: target.UploadToken,
	}, nil
}
