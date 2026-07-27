package service

// CreateProjectPreviewUploadRequest is the use-case input for creating an upload target.
type CreateProjectPreviewUploadRequest struct {
	ContentType string
}

// ProjectPreviewUploadTarget is the use-case result returned by the storage boundary.
type ProjectPreviewUploadTarget struct {
	ObjectKey string
	UploadURL string
}
