package asset

type AssetType string
type AssetResourceType string
type Status uint

const (
	AssetTypeCharacter AssetType = "character"

	AssetTypeTileSet AssetType = "tileSet"

	AssetTypeAudio AssetType = "audio"

	AssetTypeUI AssetType = "ui"

	AssetTypeObject AssetType = "object"

	AssetTypeScenery AssetType = "scenery"
)

const (
	AssetResourceTypeProtoType AssetResourceType = "protoType"

	// Image is bound to a protoType.
	AssetResourceTypeImage AssetResourceType = "image"

	AssetResourceTypeAnimation AssetResourceType = "animation"

	// Frame is bound to an animation.
	AssetResourceTypeFrame AssetResourceType = "frame"

	AssetResourceTypeItem AssetResourceType = "item"

	// Tile is bound to an item.
	AssetResourceTypeTile AssetResourceType = "tile"

	AssetResourceTypeUI AssetResourceType = "ui"

	AssetResourceTypeScenery AssetResourceType = "scenery"
)

const (
	StatusPending Status = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)
