package generator

type TaskType string

const (
	GenerateCharacterProtoType TaskType = "generate_character_prototype"
	EditCharacterProtoType     TaskType = "edit_character_prototype"
	EditCharacterAnimation     TaskType = "edit_character_animation"
	EditCharacterFrames        TaskType = "edit_character_frames"

	GenerateObjectProtoType TaskType = "generate_object_prototype"
	EditObjectProtoType     TaskType = "edit_object_prototype"
	EditObjectAnimation     TaskType = "edit_object_animation"
	EditObjectFrames        TaskType = "edit_object_frames"

	GenerateAnimation TaskType = "generate_animation"
	GenerateTileSet   TaskType = "generate_tileset"
	EditTilesetItem   TaskType = "edit_tileset_item"
	EditTiles         TaskType = "edit_tiles"
)

func ProjectLevelTaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		GenerateObjectProtoType,
		GenerateTileSet,
	}
}

func TaskTypes() []TaskType {
	return []TaskType{
		GenerateCharacterProtoType,
		EditCharacterProtoType,
		EditCharacterAnimation,
		EditCharacterFrames,
		GenerateObjectProtoType,
		EditObjectProtoType,
		EditObjectAnimation,
		EditObjectFrames,
		GenerateAnimation,
		GenerateTileSet,
		EditTilesetItem,
		EditTiles,
	}
}
