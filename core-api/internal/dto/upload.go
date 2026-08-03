package dto

// CreateUploadTargetRequest describes a file uploaded directly to object storage.
type CreateUploadTargetRequest struct {
	ContentType   string `json:"contentType" minLength:"1"`
	ContentLength int64  `json:"contentLength" minimum:"1"`
}

// UploadTarget is the HTTP response for a temporary upload target.
type UploadTarget struct {
	ObjectKey   string `json:"objectKey"`
	ObjectURL   string `json:"objectURL"`
	UploadURL   string `json:"uploadURL"`
	UploadToken string `json:"uploadToken"`
}
