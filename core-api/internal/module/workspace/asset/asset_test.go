package asset_test

import (
	"encoding/json"
	"reflect"
	"testing"

	assetdomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/asset"
)

func TestTagDecodesLegacyStringAndStructuredObject(t *testing.T) {
	var tags []assetdomain.Tag
	if err := json.Unmarshal([]byte(`[
		"knight",
		{"name":"villager","description":"town resident","color":"#123456"}
	]`), &tags); err != nil {
		t.Fatalf("decode tags: %v", err)
	}
	want := []assetdomain.Tag{
		{Name: "knight", Color: assetdomain.DefaultTagColor},
		{Name: "villager", Description: "town resident", Color: "#123456"},
	}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("unexpected tags: got %+v want %+v", tags, want)
	}
}

func TestAssetTagNamesSkipsBlankNames(t *testing.T) {
	value := assetdomain.Asset{Tags: []assetdomain.Tag{
		{Name: " knight "},
		{Name: "  "},
		{Name: "villager"},
	}}
	if got, want := value.TagNames(), []string{"knight", "villager"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tag names: got %v want %v", got, want)
	}
}
