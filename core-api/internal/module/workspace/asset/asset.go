package asset

import "encoding/json"

type Asset struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        AssetType
	Description string
	Tags        []string        `json:"tags"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
	Version     uint
}

type AssetListFilter struct {
	Query string
	Tags  []string
	Types []AssetType
}

type AssetUpdate struct {
	Name        *string
	ProjectID   *uint
	Type        *AssetType
	Description *string
	Tags        *[]string
	Attributes  *json.RawMessage
}
