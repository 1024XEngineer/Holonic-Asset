package generator

// CreateCharacterPrototypePayload is the complete input consumed by the
// character prototype task handler.
type CreateCharacterPrototypePayload struct {
	AssetName     string `json:"asset_name"`
	CreativeBrief string `json:"creative_brief"`
	CanvasSize    string `json:"canvas_size"`
	Perspective   string `json:"perspective"`
	Reference     string `json:"reference"`
	ProjectID     uint   `json:"project_id"`
}

// CreateAnimationPayload is the common input consumed by character and object
// animation generation.
type CreateAnimationPayload struct {
	AnimationName string `json:"animation_name"`
	ProjectID     uint   `json:"project_id"`
	AssetID       uint   `json:"asset_id"`
	// Direction is an English name shared by character and object assets, such as
	// "front", "left", or "back_right". The name is resolved against the
	// asset's direction_count and prototype ordering.
	Direction     string `json:"direction"`
	CreativeBrief string `json:"creative_brief"`
	Style         string `json:"style,omitempty"`
	FrameCount    int    `json:"frame_count,omitempty"`
	Columns       int    `json:"columns,omitempty"`
	FrameWidth    int    `json:"frame_width,omitempty"`
	FrameHeight   int    `json:"frame_height,omitempty"`
	FPS           int    `json:"fps,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	AspectRatio   string `json:"aspect_ratio,omitempty"`
}

// CreateObjectPrototypePayload is the complete input consumed by the object
// prototype task handler.
type CreateObjectPrototypePayload struct {
	AssetName      string `json:"asset_name"`
	CreativeBrief  string `json:"creative_brief"`
	CanvasSize     string `json:"canvas_size"`
	Perspective    string `json:"perspective"`
	DirectionCount string `json:"direction_count"`
	Reference      string `json:"reference"`
	ProjectID      uint   `json:"project_id"`
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
