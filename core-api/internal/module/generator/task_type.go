package generator

type TaskType string

const (
	GenerateCharacterProtoType TaskType = "generate_character_prototype"
	EditCharacterProtoType     TaskType = "edit_character_prototype"
	EditCharacterFrames        TaskType = "edit_character_frames"

	GenerateObjectProtoType TaskType = "generate_object_prototype"
	EditObjectProtoType     TaskType = "edit_object_prototype"
	EditObjectFrames        TaskType = "edit_object_frames"

	GenerateAnimation TaskType = "generate_animation"
	EditAnimation     TaskType = "edit_animation"
	GenerateTileSet   TaskType = "generate_tileset"
	EditItem          TaskType = "edit_item"
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
		EditCharacterFrames,
		GenerateObjectProtoType,
		EditObjectProtoType,
		EditObjectFrames,
		GenerateAnimation,
		EditAnimation,
		GenerateTileSet,
		EditItem,
		EditTiles,
	}
}
