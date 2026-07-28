package service

// CreateUploadTargetRequest is the use-case input for creating an upload target.
type CreateUploadTargetRequest struct {
	ContentType   string
	ContentLength int64
}

// UploadTarget is the use-case result returned by the storage boundary.
type UploadTarget struct {
	ObjectKey   string
	ObjectURL   string
	UploadURL   string
	UploadToken string
}
