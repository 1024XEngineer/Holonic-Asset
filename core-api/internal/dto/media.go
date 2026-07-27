package dto

// CreateProjectPreviewUploadRequest describes a Project preview uploaded directly to object storage.
type CreateProjectPreviewUploadRequest struct {
	ContentType string `json:"contentType"`
}

// ProjectPreviewUploadTarget is the HTTP response for a temporary upload target.
type ProjectPreviewUploadTarget struct {
	ObjectKey string `json:"objectKey"`
	UploadURL string `json:"uploadURL"`
}
