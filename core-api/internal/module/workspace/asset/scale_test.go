package asset_test

import (
	"encoding/json"
	"testing"

	domain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestValidateScaleAcceptsTypeSpecificShapes(t *testing.T) {
	tests := []struct {
		assetType domain.AssetType
		scale     string
	}{
		{domain.AssetTypeCharacter, `{"width":64,"height":64}`},
		{domain.AssetTypeObject, `{"width":32,"height":48}`},
		{domain.AssetTypeUI, `{"width":1024,"height":768}`},
		{domain.AssetTypeScenery, `{"width":1920,"height":1080}`},
		{domain.AssetTypeTileSet, `{"tileSize":{"width":16,"height":16},"tileAmount":{"columns":10,"rows":8}}`},
		{domain.AssetTypeAudio, `null`},
	}
	for _, test := range tests {
		t.Run(string(test.assetType), func(t *testing.T) {
			if err := domain.ValidateScale(test.assetType, json.RawMessage(test.scale)); err != nil {
				t.Fatalf("validate scale: %v", err)
			}
		})
	}
}

func TestValidateScaleRejectsInvalidOrUnknownFields(t *testing.T) {
	tests := []struct {
		name      string
		assetType domain.AssetType
		scale     string
	}{
		{"missing dimension", domain.AssetTypeCharacter, `{"width":64}`},
		{"zero dimension", domain.AssetTypeObject, `{"width":0,"height":32}`},
		{"unknown field", domain.AssetTypeUI, `{"width":64,"height":64,"unit":"px"}`},
		{"wrong tileset shape", domain.AssetTypeTileSet, `{"width":64,"height":64}`},
		{"non-null audio", domain.AssetTypeAudio, `{"width":64,"height":64}`},
		{"trailing data", domain.AssetTypeScenery, `{"width":64,"height":64} {}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := domain.ValidateScale(test.assetType, json.RawMessage(test.scale)); err == nil {
				t.Fatal("expected scale validation error")
			}
		})
	}
}
