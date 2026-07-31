package handler

import (
	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/echox"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
)

type UploadHandler struct {
	manager upload.Manager
}

func NewUploadHandler(manager upload.Manager) *UploadHandler {
	return &UploadHandler{manager: manager}
}

func (h *UploadHandler) CreateUploadTarget(
	c *echox.Context,
	request dto.CreateUploadTargetRequest,
) (*dto.UploadTarget, error) {
	target, err := h.manager.CreateUploadTarget(c, &upload.CreateUploadTargetRequest{
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
