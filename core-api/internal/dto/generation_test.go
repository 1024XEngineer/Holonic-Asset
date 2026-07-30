package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	taskdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/task"
)

func TestCreateGenerationRequestUsesPromptField(t *testing.T) {
	var request dto.CreateGenerationRequest
	if err := json.Unmarshal([]byte(`{"prompt":"hero"}`), &request); err != nil {
		t.Fatalf("unmarshal generation request: %v", err)
	}

	if request.Prompt != "hero" {
		t.Fatalf("expected prompt %q, got %q", "hero", request.Prompt)
	}
}

func TestGenerationResponseUsesTaskStatus(t *testing.T) {
	response := dto.GetGenerationResponse{Status: taskdomain.StatusProcessing}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal generation response: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal generation response: %v", err)
	}
	if body["status"] != float64(taskdomain.StatusProcessing) {
		t.Fatalf("unexpected task status: %v", body["status"])
	}
	if _, exists := body["lifecycle"]; exists {
		t.Fatalf("generation response must not expose lifecycle: %s", payload)
	}
}
