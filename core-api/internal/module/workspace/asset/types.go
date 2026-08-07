package asset

import perspectivedomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/perspective"

type Perspective = perspectivedomain.Perspective

const (
	PerspectiveTopDown   = perspectivedomain.TopDown
	PerspectiveSideOn    = perspectivedomain.SideOn
	PerspectiveIsometric = perspectivedomain.Isometric
)

type AssetType string

const (
	AssetTypeCharacter AssetType = "character"
	AssetTypeTileSet   AssetType = "tileSet"
	AssetTypeAudio     AssetType = "audio"
	AssetTypeUI        AssetType = "ui"
	AssetTypeObject    AssetType = "object"
	AssetTypeScenery   AssetType = "scenery"
)
