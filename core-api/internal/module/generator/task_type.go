package generator

import "slices"

type TaskType string

const (
	GenerateCharacterProtoType TaskType = "generate_character_prototype"
	EditCharacterProtoType     TaskType = "edit_character_prototype"
	EditFrames                 TaskType = "edit_frames"

	GenerateObjectProtoType TaskType = "generate_object_prototype"
	EditObjectProtoType     TaskType = "edit_object_prototype"

	GenerateAnimation TaskType = "generate_animation"
	EditAnimation     TaskType = "edit_animation"
	GenerateScenery   TaskType = "generate_scenery"
	GenerateTileSet   TaskType = "generate_tileset"
	EditTilesetItem   TaskType = "edit_tileset_item"
	EditTiles         TaskType = "edit_tiles"
)

func ProjectLevelTaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		GenerateObjectProtoType,
		GenerateScenery,
		GenerateTileSet,
	}
}

func (t TaskType) AwaitsApplication() bool {
	switch t {
	case EditCharacterProtoType,
		EditObjectProtoType,
		EditFrames,
		GenerateAnimation,
		EditAnimation,
		EditTilesetItem,
		EditTiles:
		return true
	default:
		return false
	}
}

func (t TaskType) Valid() bool {
	return slices.Contains(TaskTypes(), t)
}

func TaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		EditCharacterProtoType,
		EditFrames,
		GenerateObjectProtoType,
		EditObjectProtoType,
		GenerateAnimation,
		EditAnimation,
		GenerateScenery,
		GenerateTileSet,
		EditTilesetItem,
		EditTiles,
	}
}
