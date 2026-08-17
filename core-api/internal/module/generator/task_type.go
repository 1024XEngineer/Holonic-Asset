package generator

type TaskType string

const (
	GenerateCharacterProtoType TaskType = "generate_character_prototype"
	EditCharacterProtoType     TaskType = "edit_character_prototype"
	EditCharacterFrames        TaskType = "edit_character_frames"

	GenerateObjectProtoType TaskType = "generate_object_prototype"
	EditObjectProtoType     TaskType = "edit_object_prototype"
	EditObjectFrames        TaskType = "edit_object_frames"

	GenerateAnimation   TaskType = "generate_animation"
	EditAnimation       TaskType = "edit_animation"
	GenerateScenery     TaskType = "generate_scenery"
	GenerateTileSet     TaskType = "generate_tileset"
	EditTilesetItem     TaskType = "edit_tileset_item"
	EditTiles           TaskType = "edit_tiles"
	GenerateUISet       TaskType = "generate_uiset"
	EditUISetComponents TaskType = "edit_uiset_components"
)

func ProjectLevelTaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		GenerateObjectProtoType,
		GenerateScenery,
		GenerateTileSet,
		GenerateUISet,
	}
}

func TaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		EditCharacterProtoType,
		EditCharacterFrames,
		GenerateObjectProtoType,
		EditObjectProtoType,
		EditObjectFrames,
		GenerateAnimation,
		EditAnimation,
		GenerateScenery,
		GenerateTileSet,
		EditTilesetItem,
		EditTiles,
		GenerateUISet,
		EditUISetComponents,
	}
}
