package asset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultTagColor = "#4F46E5"

// Tag is structured asset metadata. String decoding keeps generation and
// stored assets compatible with the legacy ["tag"] representation.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

func (t *Tag) UnmarshalJSON(data []byte) error {
	if t == nil {
		return fmt.Errorf("asset: tag is nil")
	}

	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		*t = Tag{Name: name, Color: DefaultTagColor}
		return nil
	}

	type tagAlias Tag
	var value tagAlias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("asset: decode tag: %w", err)
	}
	*t = Tag(value)
	return nil
}

type Asset struct {
	ID           uint
	Name         string
	ProjectID    uint
	Type         AssetType
	Description  string
	Tags         []Tag           `json:"tags"`
	Perspective  Perspective     `json:"perspective"`
	Dimensions   json.RawMessage `json:"dimensions"`
	ThumbnailURL string
	Content      json.RawMessage `json:"content,omitempty"`
	Version      uint
}

func (a Asset) TagNames() []string {
	names := make([]string, 0, len(a.Tags))
	for _, tag := range a.Tags {
		if name := strings.TrimSpace(tag.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

type AssetListFilter struct {
	Query string
	Tags  []string
	Types []AssetType
}

type AssetUpdate struct {
	Name        *string
	Description *string
	Tags        *[]Tag
	Perspective *Perspective
	Dimensions  *json.RawMessage
}
