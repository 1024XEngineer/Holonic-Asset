package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

func TestNewTypedSuccessResponse(t *testing.T) {
	response := dto.NewTypedSuccessResponse(struct {
		ID uint `json:"id"`
	}{ID: 7})

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if string(payload) != `{"code":200,"message":"success","data":{"id":7}}` {
		t.Fatalf("unexpected response JSON: %s", payload)
	}
}
