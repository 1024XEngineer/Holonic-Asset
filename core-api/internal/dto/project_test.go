package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

func TestCreateProjectRequestKeepsReferenceContract(t *testing.T) {
	const objectURL = "https://media.example/project-previews/new.png"
	var request dto.CreateProjectRequest

	if err := json.Unmarshal([]byte(`{"reference":"`+objectURL+`"}`), &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if request.Reference != objectURL {
		t.Fatalf("expected project reference %q, got %q", objectURL, request.Reference)
	}
}

func TestUpdateProjectRequestJSONContractSupportsPartialFields(t *testing.T) {
	reference := "https://media.example/project-previews/new.png"
	request := dto.UpdateProjectRequest{
		ProjectID: 42,
		Reference: &reference,
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	want := `{"projectID":42,"reference":"https://media.example/project-previews/new.png"}`
	if string(encoded) != want {
		t.Fatalf("expected %s, got %s", want, encoded)
	}
}

func TestUpdateProjectRequestJSONContractPreservesExplicitEmptyValue(t *testing.T) {
	emptyReference := ""
	request := dto.UpdateProjectRequest{ProjectID: 42, Reference: &emptyReference}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	want := `{"projectID":42,"reference":""}`
	if string(encoded) != want {
		t.Fatalf("expected %s, got %s", want, encoded)
	}
}
