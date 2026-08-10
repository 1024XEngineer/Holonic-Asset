package asset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type TileAmount struct {
	Columns uint `json:"columns"`
	Rows    uint `json:"rows"`
}

type TileSetDimensions struct {
	TileSize   Size       `json:"tileSize"`
	TileAmount TileAmount `json:"tileAmount"`
}

func ValidateDimensions(assetType AssetType, raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if assetType == AssetTypeAudio {
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil
		}
		return fmt.Errorf("asset: audio dimensions must be null")
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("asset: dimensions are required for %s", assetType)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	switch assetType {
	case AssetTypeCharacter, AssetTypeObject, AssetTypeUISet, AssetTypeScenery:
		var size Size
		if err := decoder.Decode(&size); err != nil {
			return fmt.Errorf("asset: decode %s dimensions: %w", assetType, err)
		}
		if size.Width == 0 || size.Height == 0 {
			return fmt.Errorf("asset: %s dimensions must be positive", assetType)
		}
	case AssetTypeTileSet:
		var dimensions TileSetDimensions
		if err := decoder.Decode(&dimensions); err != nil {
			return fmt.Errorf("asset: decode tileSet dimensions: %w", err)
		}
		if dimensions.TileSize.Width == 0 || dimensions.TileSize.Height == 0 ||
			dimensions.TileAmount.Columns == 0 || dimensions.TileAmount.Rows == 0 {
			return fmt.Errorf("asset: tileSet dimensions must be positive")
		}
	default:
		return fmt.Errorf("asset: unsupported asset type %q", assetType)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("asset: dimensions contains trailing JSON data")
	}
	return nil
}
