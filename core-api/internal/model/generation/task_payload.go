package generation

type GenerateCharacterProtoTypeJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (GenerateCharacterProtoTypeJob) Kind() string { return string(GenerateCharacterProtoType) }

type GenerateCharacterAnimationJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (GenerateCharacterAnimationJob) Kind() string { return string(GenerateCharacterAnimation) }

type RegenerateCharacterProtoTypeJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateCharacterProtoTypeJob) Kind() string { return string(RegenerateCharacterProtoType) }

type RegenerateCharacterAnimationJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateCharacterAnimationJob) Kind() string { return string(RegenerateCharacterAnimation) }

type RegenerateCharacterFramesJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateCharacterFramesJob) Kind() string { return string(RegenerateCharacterFrames) }

type GenerateObjectProtoTypeJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (GenerateObjectProtoTypeJob) Kind() string { return string(GenerateObjectProtoType) }

type GenerateObjectAnimationJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (GenerateObjectAnimationJob) Kind() string { return string(GenerateObjectAnimation) }

type RegenerateObjectProtoTypeJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateObjectProtoTypeJob) Kind() string { return string(RegenerateObjectProtoType) }

type RegenerateObjectAnimationJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateObjectAnimationJob) Kind() string { return string(RegenerateObjectAnimation) }

type RegenerateObjectFramesJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateObjectFramesJob) Kind() string { return string(RegenerateObjectFrames) }

type GenerateTileSetJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
}

func (GenerateTileSetJob) Kind() string { return string(GenerateTileSet) }

type RegenerateItemJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
	ItemIndex int  `json:"item_index"`
}

func (RegenerateItemJob) Kind() string { return string(RegenerateItem) }

type RegenerateTilesJob struct {
	TaskID    uint `json:"task_id"`
	ProjectID uint `json:"project_id"`
	AssetID   uint `json:"asset_id"`
}

func (RegenerateTilesJob) Kind() string { return string(RegenerateTiles) }
