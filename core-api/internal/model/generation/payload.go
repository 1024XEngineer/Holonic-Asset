package generation

// CreateCharacterPrototypePayload is the complete input consumed by the
// character prototype task handler.
type CreateCharacterPrototypePayload struct {
	AssetName      string `json:"asset_name"`
	CreativeBrief  string `json:"creative_brief"`
	CanvasSize     string `json:"canvas_size"`
	Perspective    string `json:"perspective"`
	DirectionCount string `json:"direction_count"`
	Reference      string `json:"reference"`
	ProjectID      uint   `json:"project_id"`
}

// CreateCharacterAnimationPayload is the complete input consumed by the
// character animation task handler.
type CreateCharacterAnimationPayload struct {
	AssetName     string `json:"asset_name"`
	ProjectID     uint   `json:"project_id"`
	ParentID      uint   `json:"parent_id"`
	CreativeBrief string `json:"creative_brief"`
}

// CreateObjectPrototypePayload is the complete input consumed by the object
// prototype task handler.
type CreateObjectPrototypePayload struct {
	AssetName     string `json:"asset_name"`
	CreativeBrief string `json:"creative_brief"`
	CanvasSize    string `json:"canvas_size"`
	Perspective   string `json:"perspective"`
	Reference     string `json:"reference"`
	ProjectID     uint   `json:"project_id"`
}

// CreateObjectAnimationPayload is the complete input consumed by the object
// animation task handler.
type CreateObjectAnimationPayload struct {
	AssetName     string `json:"asset_name"`
	ProjectID     uint   `json:"project_id"`
	ParentID      uint   `json:"parent_id"`
	CreativeBrief string `json:"creative_brief"`
}

// CreateTileSetPayload is the complete input consumed by the tileset task
// handler. TileNum is stored explicitly so a queued task is self-contained.
type CreateTileSetPayload struct {
	AssetName        string   `json:"asset_name"`
	ProjectID        uint     `json:"project_id"`
	CreativeBrief    string   `json:"creative_brief"`
	TileNum          uint     `json:"tile_num"`
	TileDescriptions []string `json:"tile_descriptions"`
	Reference        string   `json:"reference"`
}
