package asset

import (
	"encoding/json"
)

type Asset struct {
	ID          uint
	Name        string
	ProjectID   uint
	Type        AssetType
	Description string
	Tags        []string        `json:"tags"`
	Attributes  json.RawMessage `json:"attributes"`
	Version     uint
}

type AssetVersion struct {
	ID        uint
	AssetID   uint
	Version   uint
	CreatedAt int64
}

type AssetResource struct {
	ID           uint
	Name         string
	ParentID     *uint
	AssetID      uint
	AssetVersion uint
	Type         AssetResourceType
	Url          *string
	Status       Status
}
