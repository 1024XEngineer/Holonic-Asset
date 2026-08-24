package dto_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestAssetTagInputAcceptsStructuredAndLegacyTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  assetdomain.Tag
	}{
		{
			name:  "structured",
			input: `{"name":"chair","description":"furniture","color":"#123456"}`,
			want:  assetdomain.Tag{Name: "chair", Description: "furniture", Color: "#123456"},
		},
		{
			name:  "legacy string",
			input: `"chair"`,
			want:  assetdomain.Tag{Name: "chair", Color: assetdomain.DefaultTagColor},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input dto.AssetTagInput
			if err := json.Unmarshal([]byte(tt.input), &input); err != nil {
				t.Fatalf("unmarshal tag input: %v", err)
			}
			if got := input.Domain(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected domain tag: got %#v want %#v", got, tt.want)
			}
		})
	}

	var invalid dto.AssetTagInput
	if err := json.Unmarshal([]byte(`{"name":"chair","unknown":true}`), &invalid); err == nil {
		t.Fatal("expected unknown structured tag field to fail")
	}
}

func TestAssetTagInputSchemaDocumentsBothRepresentations(t *testing.T) {
	schema := (dto.AssetTagInput{}).Schema(nil)
	if schema.Description == "" || len(schema.OneOf) != 2 {
		t.Fatalf("unexpected tag schema: %#v", schema)
	}
	if schema.OneOf[0].Type != huma.TypeString {
		t.Fatalf("expected legacy string schema, got %#v", schema.OneOf[0])
	}
	structured := schema.OneOf[1]
	additionalProperties, ok := structured.AdditionalProperties.(bool)
	if structured.Type != huma.TypeObject || !ok || additionalProperties {
		t.Fatalf("expected closed structured tag schema, got %#v", structured)
	}
	if !reflect.DeepEqual(structured.Required, []string{"name"}) {
		t.Fatalf("unexpected required fields: %#v", structured.Required)
	}
	for _, field := range []string{"name", "description", "color"} {
		property, ok := structured.Properties[field]
		if !ok || property.Type != huma.TypeString {
			t.Fatalf("expected string property %q, got %#v", field, property)
		}
	}
}
