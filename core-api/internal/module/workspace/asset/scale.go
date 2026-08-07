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

type TileSetScale struct {
	TileSize   Size       `json:"tileSize"`
	TileAmount TileAmount `json:"tileAmount"`
}

func ValidateScale(assetType AssetType, raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if assetType == AssetTypeAudio {
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil
		}
		return fmt.Errorf("asset: audio scale must be null")
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("asset: scale is required for %s", assetType)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	switch assetType {
	case AssetTypeCharacter, AssetTypeObject, AssetTypeUI, AssetTypeScenery:
		var size Size
		if err := decoder.Decode(&size); err != nil {
			return fmt.Errorf("asset: decode %s scale: %w", assetType, err)
		}
		if size.Width == 0 || size.Height == 0 {
			return fmt.Errorf("asset: %s scale dimensions must be positive", assetType)
		}
	case AssetTypeTileSet:
		var scale TileSetScale
		if err := decoder.Decode(&scale); err != nil {
			return fmt.Errorf("asset: decode tileSet scale: %w", err)
		}
		if scale.TileSize.Width == 0 || scale.TileSize.Height == 0 ||
			scale.TileAmount.Columns == 0 || scale.TileAmount.Rows == 0 {
			return fmt.Errorf("asset: tileSet scale dimensions must be positive")
		}
	default:
		return fmt.Errorf("asset: unsupported asset type %q", assetType)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("asset: scale contains trailing JSON data")
	}
	return nil
}
