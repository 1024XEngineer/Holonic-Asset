package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

func TestCreateProjectPreviewUploadRequestJSONContract(t *testing.T) {
	request := dto.CreateProjectPreviewUploadRequest{ContentType: "image/png"}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	want := `{"contentType":"image/png"}`
	if string(encoded) != want {
		t.Fatalf("expected %s, got %s", want, encoded)
	}
}

func TestProjectPreviewUploadTargetJSONContract(t *testing.T) {
	target := dto.ProjectPreviewUploadTarget{
		ObjectKey: "users/7/project-previews/uuid",
		UploadURL: "https://account-id.r2.cloudflarestorage.com/bucket/users/7/project-previews/uuid?X-Amz-Signature=...",
	}

	encoded, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("marshal target: %v", err)
	}

	want := `{"objectKey":"users/7/project-previews/uuid","uploadURL":"https://account-id.r2.cloudflarestorage.com/bucket/users/7/project-previews/uuid?X-Amz-Signature=..."}`
	if string(encoded) != want {
		t.Fatalf("expected %s, got %s", want, encoded)
	}
}
