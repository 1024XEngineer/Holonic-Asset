package asset

import "encoding/json"

type Asset struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        AssetType
	Description string
	Tags        []string        `json:"tags"`
	Perspective Perspective     `json:"perspective"`
	Scale       json.RawMessage `json:"scale"`
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
	Description *string
	Tags        *[]string
	Perspective *Perspective
	Scale       *json.RawMessage
}
